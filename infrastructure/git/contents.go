package git

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

// ContentsSource は GitHub Contents API 経由でマニフェストを読む ManifestSource。
// Cloud Run のようにローカル checkout が無い環境で、owner/repo（+ref）だけから
// package.json / go.mod / .anneal.yml などの数ファイルを取得する。clone しない。
type ContentsSource struct {
	http    *http.Client
	token   string
	apiBase string
	owner   string
	repo    string
	ref     string // 空なら default branch
}

// NewContentsSource は指定 repo の Contents API を読む ManifestSource を返す。
// ref が空ならデフォルトブランチを参照する。
func NewContentsSource(token, owner, repo, ref string) gateway.ManifestSource {
	return &ContentsSource{
		http:    &http.Client{Timeout: 15 * time.Second},
		token:   token,
		apiBase: githubAPIBase,
		owner:   owner,
		repo:    repo,
		ref:     ref,
	}
}

func (c *ContentsSource) Exists(ctx context.Context, relPath string) (bool, error) {
	status, _, err := c.get(ctx, relPath)
	if err != nil {
		return false, err
	}
	if status == http.StatusNotFound {
		return false, nil
	}
	if status != http.StatusOK {
		return false, fmt.Errorf("contents exists %s/%s %s: status %d", c.owner, c.repo, relPath, status)
	}
	return true, nil
}

func (c *ContentsSource) ReadFile(ctx context.Context, relPath string) ([]byte, error) {
	status, body, err := c.get(ctx, relPath)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("contents read %s/%s %s: status %d", c.owner, c.repo, relPath, status)
	}
	var payload struct {
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("contents decode %s: %w", relPath, err)
	}
	// 1MB 超のファイルは content が空で encoding="none" になり、blobs API が必要。
	// 空を「中身なし」と誤認しないよう明示エラーにする（NF: 大声で失敗）。
	if payload.Encoding != "base64" {
		return nil, fmt.Errorf("contents %s: unsupported encoding %q (file too large?)", relPath, payload.Encoding)
	}
	data, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(payload.Content, "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("contents base64 %s: %w", relPath, err)
	}
	return data, nil
}

// get は Contents API を 1 回叩き、HTTP status と body を返す。404 は body 側で
// 存在判定に使うため、非 2xx でもここでは error にしない（トランスポート障害のみ error）。
func (c *ContentsSource) get(ctx context.Context, relPath string) (int, []byte, error) {
	url := c.contentsURL(relPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("contents request %s: %w", relPath, err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("contents get %s: %w", relPath, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("contents body %s: %w", relPath, err)
	}
	return resp.StatusCode, body, nil
}

func (c *ContentsSource) contentsURL(relPath string) string {
	// path の各セグメントを個別にエスケープしつつ "/" は保つ。
	segs := strings.Split(relPath, "/")
	for i, s := range segs {
		segs[i] = escapePath(s)
	}
	base := strings.TrimRight(c.apiBase, "/") +
		"/repos/" + escapePath(c.owner) + "/" + escapePath(c.repo) +
		"/contents/" + strings.Join(segs, "/")
	if c.ref != "" {
		q := neturl.Values{}
		q.Set("ref", c.ref)
		base += "?" + q.Encode()
	}
	return base
}
