// Package repoconfig は .anneal.yml を読み込むことで RepoConfigLoader のポートを実装する。
package repoconfig

import (
	"os"
	"path/filepath"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
	"gopkg.in/yaml.v3"
)

// Loader はリポジトリから .anneal.yml を読み込む。
type Loader struct{}

// NewLoader は RepoConfigLoader を返す。
func NewLoader() gateway.RepoConfigLoader {
	return Loader{}
}

// Load は .anneal.yml を読み込み、なければデフォルトにフォールバックする（ファイルが無くても問題ない）。
func (Loader) Load(repoPath string) model.RepoConfig {
	data, err := os.ReadFile(filepath.Join(repoPath, ".anneal.yml"))
	if err != nil {
		return model.DefaultRepoConfig()
	}
	return parseRepoConfig(data)
}

// parseRepoConfig は .anneal.yml のバイト列を RepoConfig にする。パース失敗時は
// デフォルトへフォールバックする（壊れた設定でパイプラインを止めない）。ローカル /
// リモート双方の Loader で共有する。
func parseRepoConfig(data []byte) model.RepoConfig {
	cfg := model.DefaultRepoConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return model.DefaultRepoConfig()
	}
	return cfg
}
