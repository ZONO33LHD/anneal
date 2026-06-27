package git

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

// GitHub は GitHub REST API を直接呼び出す（SDK 依存なし）。Git data API を使って
// ファイル内容をブランチへコミットする処理は、的を絞った後続対応として残している。
// ここでは real モード実行時に検証される PR 作成と CI ステータスを扱う。ブランチ作成と
// コミット作成はログ出力のみで何もしない。
type GitHub struct {
	token string
	http  *http.Client
}

// GitHub Check Run の conclusion のうち「成功」とみなす値（GitHub API 固有の値）。
const (
	ghConclusionSuccess = "success"
	ghConclusionNeutral = "neutral"
)

// NewGitHub は GitHub クライアントを生成する。
func NewGitHub(token string) gateway.Git {
	return &GitHub{token: token, http: &http.Client{Timeout: 30 * time.Second}}
}

func (GitHub) Name() string {
	return "github"
}

func split(repository string) (owner, repo string) {
	parts := strings.SplitN(repository, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return repository, ""
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
	resp, err := g.http.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("github: status %d", resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (g *GitHub) CreateBranchAndPR(ctx context.Context, opts gateway.CreatePROptions) (gateway.PRRef, error) {
	fmt.Printf("⚠ github: branch/commit creation is a no-op in this build (branch=%s)\n", opts.Branch)
	owner, repo := split(opts.Repository)
	var out struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	body := map[string]string{"title": opts.Title, "body": opts.Body, "head": opts.Branch, "base": opts.Base}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, repo)
	if err := g.do(ctx, http.MethodPost, url, body, &out); err != nil {
		return gateway.PRRef{}, err
	}
	return gateway.PRRef{Number: out.Number, URL: out.HTMLURL}, nil
}

func (g *GitHub) PushFix(_ context.Context, opts gateway.PushFixOptions) error {
	fmt.Printf("⚠ github: pushFix is a no-op in this build (pr=%d)\n", opts.PRNumber)
	return nil
}

func (g *GitHub) CheckCI(ctx context.Context, opts gateway.CICheckOptions) (gateway.CICheck, error) {
	owner, repo := split(opts.Repository)
	var out struct {
		CheckRuns []struct {
			Name       string `json:"name"`
			Conclusion string `json:"conclusion"`
		} `json:"check_runs"`
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s/check-runs", owner, repo, opts.Branch)
	if err := g.do(ctx, http.MethodGet, url, nil, &out); err != nil {
		return gateway.CICheck{}, err
	}
	failed := make([]string, 0, len(out.CheckRuns))
	for _, r := range out.CheckRuns {
		if r.Conclusion != "" && r.Conclusion != ghConclusionSuccess && r.Conclusion != ghConclusionNeutral {
			failed = append(failed, r.Name)
		}
	}
	if len(failed) == 0 {
		return gateway.CICheck{Passed: true}, nil
	}
	return gateway.CICheck{Passed: false, LogSummary: "Checks failed: " + strings.Join(failed, ", ")}, nil
}
