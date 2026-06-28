package git

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

// GitHub は GitHub REST API を直接呼び出す（SDK 依存なし）。
type GitHub struct {
	token   string
	http    *http.Client
	apiBase string
	logger  gateway.Logger
}

// GitHub Check Run の conclusion のうち「成功」とみなす値（GitHub API 固有の値）。
const (
	ghConclusionSuccess = "success"
	ghConclusionNeutral = "neutral"
	githubAPIBase       = "https://api.github.com"
)

// NewGitHub は GitHub クライアントを生成する。
func NewGitHub(token string, logger gateway.Logger) gateway.Git {
	return &GitHub{
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
		apiBase: githubAPIBase,
		logger:  logger,
	}
}

func (GitHub) ProviderName() string {
	return "github"
}

// splitOwnerRepo は "owner/repo" 形式の fullName を owner と repo に分解する
// （戻り値の順は owner, repo）。"/" を含まない不正入力時は owner にそのまま入れ、
// repo は空にする。fullName は GitHub API の full_name に相当する。
func splitOwnerRepo(fullName string) (owner, repo string) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return fullName, ""
}

type githubAPIError struct {
	status int
	body   string
}

func (e githubAPIError) Error() string {
	if e.body == "" {
		return fmt.Sprintf("github: status %d", e.status)
	}
	return fmt.Sprintf("github: status %d: %s", e.status, e.body)
}

func (g *GitHub) do(ctx context.Context, method, url string, body, out any) error {
	var rdr *bytes.Reader
	if body != nil {
		buf, _ := json.Marshal(body)
		rdr = bytes.NewReader(buf)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := g.http.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return githubAPIError{status: resp.StatusCode, body: strings.TrimSpace(string(data))}
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (g *GitHub) endpoint(parts ...string) string {
	base := strings.TrimRight(g.apiBase, "/")
	if base == "" {
		base = githubAPIBase
	}
	escaped := make([]string, 0, len(parts))
	for _, p := range parts {
		escaped = append(escaped, escapePath(p))
	}
	return base + "/" + strings.Join(escaped, "/")
}

func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		parts[i] = neturl.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func validateOwnerRepo(repository string) (string, string, error) {
	owner, repo := splitOwnerRepo(repository)
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("github: repository must be owner/repo, got %q", repository)
	}
	return owner, repo, nil
}

type githubRefResponse struct {
	Object struct {
		SHA string `json:"sha"`
	} `json:"object"`
}

type githubCommitResponse struct {
	SHA  string `json:"sha"`
	Tree struct {
		SHA string `json:"sha"`
	} `json:"tree"`
}

type githubBlobResponse struct {
	SHA string `json:"sha"`
}

type githubTreeResponse struct {
	SHA string `json:"sha"`
}

func (g *GitHub) getBranchCommit(ctx context.Context, owner, repo, branch string) (string, string, error) {
	if branch == "" {
		return "", "", fmt.Errorf("github: branch is required")
	}
	var ref githubRefResponse
	url := g.endpoint("repos", owner, repo, "git", "ref", "heads/"+branch)
	if err := g.do(ctx, http.MethodGet, url, nil, &ref); err != nil {
		return "", "", err
	}
	if ref.Object.SHA == "" {
		return "", "", fmt.Errorf("github: branch %q did not return commit sha", branch)
	}

	var commit githubCommitResponse
	url = g.endpoint("repos", owner, repo, "git", "commits", ref.Object.SHA)
	if err := g.do(ctx, http.MethodGet, url, nil, &commit); err != nil {
		return "", "", err
	}
	if commit.Tree.SHA == "" {
		return "", "", fmt.Errorf("github: commit %q did not return tree sha", ref.Object.SHA)
	}
	return ref.Object.SHA, commit.Tree.SHA, nil
}

func normalizeChangedPath(file string) (string, error) {
	if file == "" {
		return "", fmt.Errorf("github: changed file path is empty")
	}
	if filepath.IsAbs(file) {
		return "", fmt.Errorf("github: changed file must be repository-relative: %s", file)
	}
	clean := filepath.Clean(file)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("github: changed file escapes workdir: %s", file)
	}
	return filepath.ToSlash(clean), nil
}

func (g *GitHub) createBlob(ctx context.Context, owner, repo, workDir, file string) (string, string, error) {
	if workDir == "" {
		return "", "", fmt.Errorf("github: work_dir is required to read changed files")
	}
	repoPath, err := normalizeChangedPath(file)
	if err != nil {
		return "", "", err
	}
	data, err := os.ReadFile(filepath.Join(workDir, filepath.FromSlash(repoPath)))
	if err != nil {
		return "", "", err
	}
	body := map[string]string{
		"content":  base64.StdEncoding.EncodeToString(data),
		"encoding": "base64",
	}
	var out githubBlobResponse
	url := g.endpoint("repos", owner, repo, "git", "blobs")
	if err := g.do(ctx, http.MethodPost, url, body, &out); err != nil {
		return "", "", err
	}
	if out.SHA == "" {
		return "", "", fmt.Errorf("github: blob for %s did not return sha", repoPath)
	}
	return repoPath, out.SHA, nil
}

func (g *GitHub) createCommitFromWorktree(
	ctx context.Context,
	owner string,
	repo string,
	parentSHA string,
	baseTreeSHA string,
	workDir string,
	changedFiles []string,
	message string,
) (string, error) {
	if len(changedFiles) == 0 {
		return "", fmt.Errorf("github: changed_files is required")
	}
	if message == "" {
		return "", fmt.Errorf("github: commit message is required")
	}

	type treeEntry struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
		Type string `json:"type"`
		SHA  string `json:"sha"`
	}
	entries := make([]treeEntry, 0, len(changedFiles))
	for _, file := range changedFiles {
		repoPath, blobSHA, err := g.createBlob(ctx, owner, repo, workDir, file)
		if err != nil {
			return "", err
		}
		entries = append(entries, treeEntry{
			Path: repoPath,
			Mode: "100644",
			Type: "blob",
			SHA:  blobSHA,
		})
	}

	var tree githubTreeResponse
	treeBody := struct {
		BaseTree string      `json:"base_tree"`
		Tree     []treeEntry `json:"tree"`
	}{BaseTree: baseTreeSHA, Tree: entries}
	url := g.endpoint("repos", owner, repo, "git", "trees")
	if err := g.do(ctx, http.MethodPost, url, treeBody, &tree); err != nil {
		return "", err
	}
	if tree.SHA == "" {
		return "", fmt.Errorf("github: created tree did not return sha")
	}

	var commit githubCommitResponse
	commitBody := struct {
		Message string   `json:"message"`
		Tree    string   `json:"tree"`
		Parents []string `json:"parents"`
	}{Message: message, Tree: tree.SHA, Parents: []string{parentSHA}}
	url = g.endpoint("repos", owner, repo, "git", "commits")
	if err := g.do(ctx, http.MethodPost, url, commitBody, &commit); err != nil {
		return "", err
	}
	if commit.SHA == "" {
		return "", fmt.Errorf("github: created commit did not return sha")
	}
	return commit.SHA, nil
}

func (g *GitHub) createOrUpdateBranch(ctx context.Context, owner, repo, branch, sha string) error {
	body := map[string]string{"ref": "refs/heads/" + branch, "sha": sha}
	url := g.endpoint("repos", owner, repo, "git", "refs")
	if err := g.do(ctx, http.MethodPost, url, body, nil); err != nil {
		if !isStatus(err, http.StatusUnprocessableEntity) {
			return err
		}
		if g.logger != nil {
			g.logger.Info(ctx, "github branch already exists; force updating", "branch", branch)
		}
		return g.updateBranch(ctx, owner, repo, branch, sha)
	}
	return nil
}

func isStatus(err error, status int) bool {
	if err == nil {
		return false
	}
	apiErr, ok := err.(githubAPIError)
	if !ok {
		return false
	}
	return apiErr.status == status
}

func (g *GitHub) updateBranch(ctx context.Context, owner, repo, branch, sha string) error {
	body := struct {
		SHA   string `json:"sha"`
		Force bool   `json:"force"`
	}{SHA: sha, Force: true}
	url := g.endpoint("repos", owner, repo, "git", "refs", "heads/"+branch)
	return g.do(ctx, http.MethodPatch, url, body, nil)
}

func (g *GitHub) CreateBranchAndPR(ctx context.Context, opts gateway.CreatePROptions) (gateway.PRRef, error) {
	owner, repo, err := validateOwnerRepo(opts.Repository)
	if err != nil {
		return gateway.PRRef{}, err
	}
	baseSHA, baseTreeSHA, err := g.getBranchCommit(ctx, owner, repo, opts.Base)
	if err != nil {
		return gateway.PRRef{}, err
	}
	commitSHA, err := g.createCommitFromWorktree(
		ctx,
		owner,
		repo,
		baseSHA,
		baseTreeSHA,
		opts.WorkDir,
		opts.ChangedFiles,
		opts.Title,
	)
	if err != nil {
		return gateway.PRRef{}, err
	}
	if err := g.createOrUpdateBranch(ctx, owner, repo, opts.Branch, commitSHA); err != nil {
		return gateway.PRRef{}, err
	}
	if g.logger != nil {
		g.logger.Info(ctx, "github branch commit created", "branch", opts.Branch, "commit", commitSHA)
	}

	var out struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	body := map[string]string{"title": opts.Title, "body": opts.Body, "head": opts.Branch, "base": opts.Base}
	url := g.endpoint("repos", owner, repo, "pulls")
	if err := g.do(ctx, http.MethodPost, url, body, &out); err != nil {
		return gateway.PRRef{}, err
	}
	if out.Number == 0 || out.HTMLURL == "" {
		return gateway.PRRef{}, fmt.Errorf("github: pull request response missing number or url")
	}
	return gateway.PRRef{Number: out.Number, URL: out.HTMLURL}, nil
}

func (g *GitHub) PushFix(ctx context.Context, opts gateway.PushFixOptions) error {
	owner, repo, err := validateOwnerRepo(opts.Repository)
	if err != nil {
		return err
	}
	parentSHA, baseTreeSHA, err := g.getBranchCommit(ctx, owner, repo, opts.Branch)
	if err != nil {
		return err
	}
	commitSHA, err := g.createCommitFromWorktree(
		ctx,
		owner,
		repo,
		parentSHA,
		baseTreeSHA,
		opts.WorkDir,
		opts.ChangedFiles,
		opts.Message,
	)
	if err != nil {
		return err
	}
	if err := g.updateBranch(ctx, owner, repo, opts.Branch, commitSHA); err != nil {
		return err
	}
	if g.logger != nil {
		g.logger.Info(ctx, "github fix commit pushed", "branch", opts.Branch, "commit", commitSHA, "pr", opts.PRNumber)
	}
	return nil
}

func (g *GitHub) CheckCI(ctx context.Context, opts gateway.CICheckOptions) (gateway.CICheck, error) {
	owner, repo, err := validateOwnerRepo(opts.Repository)
	if err != nil {
		return gateway.CICheck{}, err
	}
	var out struct {
		CheckRuns []struct {
			Name       string `json:"name"`
			Status     string `json:"status"`     // queued | in_progress | completed
			Conclusion string `json:"conclusion"` // success | neutral | failure | ...
		} `json:"check_runs"`
	}
	// ブランチ名は "anneal/npm/..." のように "/" を含むため、各パス要素を個別に
	// エスケープしてパスが壊れないようにする。
	base := strings.TrimRight(g.apiBase, "/")
	if base == "" {
		base = githubAPIBase
	}
	url := base + "/repos/" +
		neturl.PathEscape(owner) + "/" + neturl.PathEscape(repo) +
		"/commits/" + neturl.PathEscape(opts.Branch) + "/check-runs"
	if err := g.do(ctx, http.MethodGet, url, nil, &out); err != nil {
		return gateway.CICheck{}, err
	}

	// CI が未開始（check run なし）や未完了（queued/in_progress）を成功扱いしない。
	// 完了済みの check がすべて success/neutral のときだけ pass とする。
	if len(out.CheckRuns) == 0 {
		return gateway.CICheck{Complete: false, Passed: false, LogSummary: "no check runs yet"}, nil
	}
	var pending, failed []string
	for _, r := range out.CheckRuns {
		if r.Status != "completed" {
			pending = append(pending, r.Name)
			continue
		}
		if r.Conclusion != ghConclusionSuccess && r.Conclusion != ghConclusionNeutral {
			failed = append(failed, r.Name)
		}
	}
	switch {
	case len(pending) > 0:
		return gateway.CICheck{Complete: false, Passed: false, LogSummary: "checks not complete: " + strings.Join(pending, ", ")}, nil
	case len(failed) > 0:
		return gateway.CICheck{Complete: true, Passed: false, LogSummary: "Checks failed: " + strings.Join(failed, ", ")}, nil
	default:
		return gateway.CICheck{Complete: true, Passed: true}, nil
	}
}
