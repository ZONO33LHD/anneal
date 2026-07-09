package repoconfig

import (
	"context"
	"fmt"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
)

// RemoteLoader は ManifestSource 経由で .anneal.yml を読む RepoConfigLoader。
// ローカルパスに依存せず、owner/repo（factory が Contents API 等の source に解決）から
// repo 個別ポリシーを取得する。gateway.RepoConfigLoader の Load(string) 契約は不変で、
// 引数はローカルパスではなく repo 参照（owner/repo）として解釈する。
type RemoteLoader struct {
	newSource gateway.ManifestSourceFactory
}

// NewRemoteLoader は ManifestSource 経由の RepoConfigLoader を返す。
func NewRemoteLoader(newSource gateway.ManifestSourceFactory) gateway.RepoConfigLoader {
	return RemoteLoader{newSource: newSource}
}

// Load は repoRef（owner/repo）の .anneal.yml を取得する。存在しない／取得失敗時は
// デフォルトへフォールバックする（ローカル Loader と同じ緩さ）。RepoConfigLoader の
// 契約に ctx が無いため context.Background を用いる。
func (r RemoteLoader) Load(repoRef string) (model.RepoConfig, error) {
	ctx := context.Background()
	src := r.newSource(repoRef)
	ok, err := src.Exists(ctx, ".anneal.yml")
	if err != nil {
		return model.RepoConfig{}, fmt.Errorf("repoconfig: check remote .anneal.yml for %s: %w", repoRef, err)
	}
	if !ok {
		return model.DefaultRepoConfig(), nil
	}
	data, err := src.ReadFile(ctx, ".anneal.yml")
	if err != nil {
		return model.RepoConfig{}, fmt.Errorf("repoconfig: read remote .anneal.yml for %s: %w", repoRef, err)
	}
	return parseRepoConfig(data)
}
