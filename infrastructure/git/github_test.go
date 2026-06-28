package git

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

type githubRequestStep struct {
	method string
	path   string
	check  func(t *testing.T, body map[string]any)
	write  func(w http.ResponseWriter)
}

func newTestGitHub(t *testing.T, steps []githubRequestStep) *GitHub {
	t.Helper()
	i := 0
	t.Cleanup(func() {
		if i != len(steps) {
			t.Fatalf("handled %d requests, want %d", i, len(steps))
		}
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Helper()
		if i >= len(steps) {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		step := steps[i]
		i++
		if r.Method != step.method || r.URL.Path != step.path {
			t.Fatalf("step %d: got %s %s, want %s %s", i, r.Method, r.URL.Path, step.method, step.path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization header=%q", got)
		}
		var body map[string]any
		if r.Body != nil && step.check != nil {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			step.check(t, body)
		}
		if step.write != nil {
			step.write(w)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return &GitHub{
		token: "test-token",
		http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, r)
			return rec.Result(), nil
		})},
		apiBase: "https://api.github.test",
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func writeJSON(v any) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(v); err != nil {
			panic(err)
		}
	}
}

func writeStatus(status int, v any) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.WriteHeader(status)
		if v != nil {
			if err := json.NewEncoder(w).Encode(v); err != nil {
				panic(err)
			}
		}
	}
}

func requireString(t *testing.T, body map[string]any, key, want string) {
	t.Helper()
	if got, _ := body[key].(string); got != want {
		t.Fatalf("%s=%q, want %q in %#v", key, got, want, body)
	}
}

func requireBool(t *testing.T, body map[string]any, key string, want bool) {
	t.Helper()
	if got, _ := body[key].(bool); got != want {
		t.Fatalf("%s=%v, want %v in %#v", key, got, want, body)
	}
}

func TestGitHubCreateBranchAndPRCreatesGitDataThenPullRequest(t *testing.T) {
	workDir := t.TempDir()
	writeFile(t, workDir, "package.json", `{"dependencies":{"axios":"1.7.0"}}`)
	writeFile(t, workDir, "nested/file.txt", "updated")

	steps := []githubRequestStep{
		{
			method: http.MethodGet,
			path:   "/repos/acme/demo/git/ref/heads/main",
			write:  writeJSON(map[string]any{"object": map[string]any{"sha": "base-sha"}}),
		},
		{
			method: http.MethodGet,
			path:   "/repos/acme/demo/git/commits/base-sha",
			write:  writeJSON(map[string]any{"sha": "base-sha", "tree": map[string]any{"sha": "base-tree"}}),
		},
		{
			method: http.MethodPost,
			path:   "/repos/acme/demo/git/blobs",
			check: func(t *testing.T, body map[string]any) {
				requireString(t, body, "encoding", "base64")
				requireString(t, body, "content", base64.StdEncoding.EncodeToString([]byte(`{"dependencies":{"axios":"1.7.0"}}`)))
			},
			write: writeJSON(map[string]any{"sha": "blob-package"}),
		},
		{
			method: http.MethodPost,
			path:   "/repos/acme/demo/git/blobs",
			check: func(t *testing.T, body map[string]any) {
				requireString(t, body, "encoding", "base64")
				requireString(t, body, "content", base64.StdEncoding.EncodeToString([]byte("updated")))
			},
			write: writeJSON(map[string]any{"sha": "blob-nested"}),
		},
		{
			method: http.MethodPost,
			path:   "/repos/acme/demo/git/trees",
			check: func(t *testing.T, body map[string]any) {
				requireString(t, body, "base_tree", "base-tree")
				tree, ok := body["tree"].([]any)
				if !ok || len(tree) != 2 {
					t.Fatalf("tree=%#v", body["tree"])
				}
				first := tree[0].(map[string]any)
				if first["path"] != "package.json" || first["mode"] != "100644" || first["type"] != "blob" || first["sha"] != "blob-package" {
					t.Fatalf("first tree entry=%#v", first)
				}
				second := tree[1].(map[string]any)
				if second["path"] != "nested/file.txt" || second["sha"] != "blob-nested" {
					t.Fatalf("second tree entry=%#v", second)
				}
			},
			write: writeJSON(map[string]any{"sha": "new-tree"}),
		},
		{
			method: http.MethodPost,
			path:   "/repos/acme/demo/git/commits",
			check: func(t *testing.T, body map[string]any) {
				requireString(t, body, "message", "chore: bump axios")
				requireString(t, body, "tree", "new-tree")
				parents, ok := body["parents"].([]any)
				if !ok || len(parents) != 1 || parents[0] != "base-sha" {
					t.Fatalf("parents=%#v", body["parents"])
				}
			},
			write: writeJSON(map[string]any{"sha": "new-commit"}),
		},
		{
			method: http.MethodPost,
			path:   "/repos/acme/demo/git/refs",
			check: func(t *testing.T, body map[string]any) {
				requireString(t, body, "ref", "refs/heads/anneal/npm/axios")
				requireString(t, body, "sha", "new-commit")
			},
			write: writeStatus(http.StatusUnprocessableEntity, map[string]any{"message": "Reference already exists"}),
		},
		{
			method: http.MethodPatch,
			path:   "/repos/acme/demo/git/refs/heads/anneal/npm/axios",
			check: func(t *testing.T, body map[string]any) {
				requireString(t, body, "sha", "new-commit")
				requireBool(t, body, "force", true)
			},
			write: writeStatus(http.StatusOK, nil),
		},
		{
			method: http.MethodPost,
			path:   "/repos/acme/demo/pulls",
			check: func(t *testing.T, body map[string]any) {
				requireString(t, body, "title", "chore: bump axios")
				requireString(t, body, "body", "body")
				requireString(t, body, "head", "anneal/npm/axios")
				requireString(t, body, "base", "main")
			},
			write: writeJSON(map[string]any{"number": 42, "html_url": "https://github.com/acme/demo/pull/42"}),
		},
	}
	ref, err := newTestGitHub(t, steps).CreateBranchAndPR(context.Background(), gateway.CreatePROptions{
		Repository:   "acme/demo",
		WorkDir:      workDir,
		Base:         "main",
		Branch:       "anneal/npm/axios",
		Title:        "chore: bump axios",
		Body:         "body",
		ChangedFiles: []string{"package.json", "nested/file.txt"},
	})
	if err != nil {
		t.Fatalf("CreateBranchAndPR: %v", err)
	}
	if ref.Number != 42 || ref.URL != "https://github.com/acme/demo/pull/42" {
		t.Fatalf("ref=%+v", ref)
	}
}

func TestGitHubPushFixCreatesCommitAndForceUpdatesBranch(t *testing.T) {
	workDir := t.TempDir()
	writeFile(t, workDir, "fix.txt", "fixed")

	steps := []githubRequestStep{
		{
			method: http.MethodGet,
			path:   "/repos/acme/demo/git/ref/heads/anneal/npm/axios",
			write:  writeJSON(map[string]any{"object": map[string]any{"sha": "tip-sha"}}),
		},
		{
			method: http.MethodGet,
			path:   "/repos/acme/demo/git/commits/tip-sha",
			write:  writeJSON(map[string]any{"sha": "tip-sha", "tree": map[string]any{"sha": "tip-tree"}}),
		},
		{
			method: http.MethodPost,
			path:   "/repos/acme/demo/git/blobs",
			check: func(t *testing.T, body map[string]any) {
				requireString(t, body, "content", base64.StdEncoding.EncodeToString([]byte("fixed")))
			},
			write: writeJSON(map[string]any{"sha": "fix-blob"}),
		},
		{
			method: http.MethodPost,
			path:   "/repos/acme/demo/git/trees",
			check: func(t *testing.T, body map[string]any) {
				requireString(t, body, "base_tree", "tip-tree")
			},
			write: writeJSON(map[string]any{"sha": "fix-tree"}),
		},
		{
			method: http.MethodPost,
			path:   "/repos/acme/demo/git/commits",
			check: func(t *testing.T, body map[string]any) {
				requireString(t, body, "message", "fix: address ci")
				requireString(t, body, "tree", "fix-tree")
			},
			write: writeJSON(map[string]any{"sha": "fix-commit"}),
		},
		{
			method: http.MethodPatch,
			path:   "/repos/acme/demo/git/refs/heads/anneal/npm/axios",
			check: func(t *testing.T, body map[string]any) {
				requireString(t, body, "sha", "fix-commit")
				requireBool(t, body, "force", true)
			},
			write: writeStatus(http.StatusOK, nil),
		},
	}
	err := newTestGitHub(t, steps).PushFix(context.Background(), gateway.PushFixOptions{
		Repository:   "acme/demo",
		WorkDir:      workDir,
		Branch:       "anneal/npm/axios",
		PRNumber:     42,
		Message:      "fix: address ci",
		ChangedFiles: []string{"fix.txt"},
	})
	if err != nil {
		t.Fatalf("PushFix: %v", err)
	}
}

func TestGitHubCreateBranchAndPRReturnsErrorOnGitDataFailure(t *testing.T) {
	workDir := t.TempDir()
	writeFile(t, workDir, "package.json", "{}")
	calledPR := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/demo/git/ref/heads/main":
			writeJSON(map[string]any{"object": map[string]any{"sha": "base-sha"}})(w)
		case "/repos/acme/demo/git/commits/base-sha":
			writeJSON(map[string]any{"sha": "base-sha", "tree": map[string]any{"sha": "base-tree"}})(w)
		case "/repos/acme/demo/git/blobs":
			writeStatus(http.StatusInternalServerError, map[string]any{"message": "boom"})(w)
		case "/repos/acme/demo/pulls":
			calledPR = true
			w.WriteHeader(http.StatusCreated)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		return rec.Result(), nil
	})}
	github := &GitHub{
		token:   "test-token",
		http:    client,
		apiBase: "https://api.github.test",
	}

	_, err := github.CreateBranchAndPR(context.Background(), gateway.CreatePROptions{
		Repository:   "acme/demo",
		WorkDir:      workDir,
		Base:         "main",
		Branch:       "anneal/npm/axios",
		Title:        "chore: bump axios",
		ChangedFiles: []string{"package.json"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calledPR {
		t.Fatal("PR should not be created after git data failure")
	}
	if err != nil && err.Error() == "" {
		t.Fatal("error should include context")
	}
}

func writeFile(t *testing.T, root, rel, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
