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
	cfg := model.DefaultRepoConfig()
	data, err := os.ReadFile(filepath.Join(repoPath, ".anneal.yml"))
	if err != nil {
		return cfg
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return model.DefaultRepoConfig()
	}
	return cfg
}
