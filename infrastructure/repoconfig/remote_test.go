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
	cfg := loaderWith(src).Load("acme/web")
	if !slices.Contains(cfg.Ignore, "react") {
		t.Fatalf("ignore=%v want to contain react", cfg.Ignore)
	}
}

// .anneal.yml が無ければデフォルトへフォールバック（ローカル Loader と同じ緩さ）。
func TestRemoteLoader_MissingFallsBackToDefault(t *testing.T) {
	cfg := loaderWith(fakeSource{exists: false}).Load("acme/web")
	if cfg.BaseBranch != model.DefaultRepoConfig().BaseBranch {
		t.Fatalf("got %+v want default", cfg)
	}
}

// 取得失敗（Exists error）も「更新なし」ではなくデフォルトで継続する。
func TestRemoteLoader_ExistsErrorFallsBackToDefault(t *testing.T) {
	cfg := loaderWith(fakeSource{existsErr: errors.New("network")}).Load("acme/web")
	if len(cfg.Ignore) != 0 {
		t.Fatalf("ignore=%v want empty default", cfg.Ignore)
	}
}
