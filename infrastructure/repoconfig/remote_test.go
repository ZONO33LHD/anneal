package repoconfig_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/infrastructure/repoconfig"
)

type fakeSource struct {
	data      []byte
	exists    bool
	existsErr error
}

func (f fakeSource) Exists(context.Context, string) (bool, error) { return f.exists, f.existsErr }
func (f fakeSource) ReadFile(context.Context, string) ([]byte, error) {
	return f.data, nil
}

func loaderWith(src gateway.ManifestSource) gateway.RepoConfigLoader {
	return repoconfig.NewRemoteLoader(func(string) gateway.ManifestSource { return src })
}

// リモートの .anneal.yml が読めれば、その ignore 等が反映される（remote でも
// 除外/上書きが効くことが T11 の要）。
func TestRemoteLoader_ParsesRemoteAnnealYml(t *testing.T) {
	src := fakeSource{exists: true, data: []byte("ignore:\n  - react\n")}
	cfg, err := loaderWith(src).Load("acme/web")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !slices.Contains(cfg.Ignore, "react") {
		t.Fatalf("ignore=%v want to contain react", cfg.Ignore)
	}
}

// .anneal.yml が無ければ組み込み既定へフォールバック（error なし）。
func TestRemoteLoader_MissingFallsBackToDefault(t *testing.T) {
	cfg, err := loaderWith(fakeSource{exists: false}).Load("acme/web")
	if err != nil {
		t.Fatalf("missing .anneal.yml should not error: %v", err)
	}
	if cfg.BaseBranch != model.DefaultRepoConfig().BaseBranch {
		t.Fatalf("got %+v want default", cfg)
	}
}

// 取得失敗（Exists error）は error にする。一時的な失敗を「ポリシー無し」と取り違え、
// 意図しない既定で scan してしまうのを防ぐ。
func TestRemoteLoader_ExistsErrorReturnsError(t *testing.T) {
	if _, err := loaderWith(fakeSource{existsErr: errors.New("network")}).Load("acme/web"); err == nil {
		t.Fatal("fetch error should propagate, not fall back to default")
	}
}

// 壊れた .anneal.yml は error にする。除外や設定を黙って既定へ落とし、ユーザーの
// 意図（例: ignore リスト）を握りつぶすのを防ぐ。
func TestRemoteLoader_MalformedYmlReturnsError(t *testing.T) {
	src := fakeSource{exists: true, data: []byte("ignore: [react\n  broken: :")}
	if _, err := loaderWith(src).Load("acme/web"); err == nil {
		t.Fatal("malformed .anneal.yml should error, not fall back to default")
	}
}
