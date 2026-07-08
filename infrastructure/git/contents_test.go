package git

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestContentsSource(server *httptest.Server, ref string) *ContentsSource {
	return &ContentsSource{
		http:    &http.Client{Timeout: 5 * time.Second},
		token:   "t",
		apiBase: server.URL,
		owner:   "acme",
		repo:    "web",
		ref:     ref,
	}
}

// 正常系: base64 の content を復号し、path/ref が正しく URL に載ること。
func TestContentsSource_ReadFile(t *testing.T) {
	const want = `{"dependencies":{"axios":"1.6.0"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/web/contents/package.json" {
			t.Errorf("path=%q", r.URL.Path)
		}
		if r.URL.Query().Get("ref") != "main" {
			t.Errorf("ref=%q", r.URL.Query().Get("ref"))
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"encoding": "base64",
			// GitHub は 60 文字ごとに改行を挟むので \n を含めても復号できること。
			"content": base64.StdEncoding.EncodeToString([]byte(want)) + "\n",
		})
	}))
	defer srv.Close()

	got, err := newTestContentsSource(srv, "main").ReadFile(context.Background(), "package.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// 404 は Exists で (false, nil)。取得失敗（存在しない）を更新なしと区別できる。
func TestContentsSource_Exists404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	ok, err := newTestContentsSource(srv, "").Exists(context.Background(), "go.mod")
	if err != nil {
		t.Fatalf("Exists err=%v want nil", err)
	}
	if ok {
		t.Fatal("Exists=true want false for 404")
	}
}

// 大きすぎるファイル（encoding!=base64）は空を返さず明示エラーにする。
func TestContentsSource_TooLargeIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"encoding": "none", "content": ""})
	}))
	defer srv.Close()

	if _, err := newTestContentsSource(srv, "").ReadFile(context.Background(), "package.json"); err == nil {
		t.Fatal("ReadFile of oversized file should error")
	}
}
