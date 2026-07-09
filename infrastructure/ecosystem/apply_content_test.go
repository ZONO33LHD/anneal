package ecosystem

import (
	"context"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

// memSource は ManifestSource の in-memory 実装（リモート content apply のテスト用）。
type memSource struct {
	files map[string][]byte
}

func (m memSource) Exists(_ context.Context, relPath string) (bool, error) {
	_, ok := m.files[relPath]
	return ok, nil
}

func (m memSource) ReadFile(_ context.Context, relPath string) ([]byte, error) {
	data, ok := m.files[relPath]
	if !ok {
		return nil, context.Canceled // 使わない想定。存在しないファイルの読取りはテストで起きない。
	}
	return data, nil
}

func TestGoModApplyUpdateContent(t *testing.T) {
	src := memSource{files: map[string][]byte{
		"go.mod": []byte("module example.com/x\n\ngo 1.22\n\nrequire github.com/foo/bar v1.2.0\n"),
	}}
	out, err := GoMod{}.ApplyUpdateContent(context.Background(), src, "github.com/foo/bar", "1.3.0")
	if err != nil {
		t.Fatalf("ApplyUpdateContent: %v", err)
	}
	got, ok := out["go.mod"]
	if !ok {
		t.Fatalf("go.mod not in result: %v", out)
	}
	want := "module example.com/x\n\ngo 1.22\n\nrequire github.com/foo/bar v1.3.0\n"
	if string(got) != want {
		t.Fatalf("go.mod content=\n%q\nwant\n%q", got, want)
	}
}

func TestGoModApplyUpdateContentNoMatch(t *testing.T) {
	src := memSource{files: map[string][]byte{
		"go.mod": []byte("module example.com/x\n\nrequire github.com/foo/bar v1.2.0\n"),
	}}
	out, err := GoMod{}.ApplyUpdateContent(context.Background(), src, "github.com/does/not-exist", "9.9.9")
	if err != nil {
		t.Fatalf("ApplyUpdateContent: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty result for no-match, got %v", out)
	}
}

func TestNPMApplyUpdateContent(t *testing.T) {
	src := memSource{files: map[string][]byte{
		"package.json": []byte(`{"dependencies":{"axios":"^1.7.0","react":"18.0.0"}}`),
	}}
	out, err := NPM{}.ApplyUpdateContent(context.Background(), src, "axios", "1.8.0")
	if err != nil {
		t.Fatalf("ApplyUpdateContent: %v", err)
	}
	got, ok := out["package.json"]
	if !ok {
		t.Fatalf("package.json not in result: %v", out)
	}
	want := `{"dependencies":{"axios":"^1.8.0","react":"18.0.0"}}`
	if string(got) != want {
		t.Fatalf("package.json content=%q want %q", got, want)
	}
}

// ManifestSource として使えることをコンパイル時に保証する。
var _ gateway.ManifestSource = memSource{}
