// Package repoconfig は .anneal.yml を読み込むことで RepoConfigLoader のポートを実装する。
package repoconfig

import (
	"errors"
	"fmt"
	"io/fs"
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

// Load は .anneal.yml を読み込む。ファイルが無ければ組み込み既定を返す（error なし）。
// ファイルはあるが読めない / 壊れている場合は error を返す（意図したポリシーを黙って
// 既定へ落とさない）。
func (Loader) Load(repoPath string) (model.RepoConfig, error) {
	data, err := os.ReadFile(filepath.Join(repoPath, ".anneal.yml"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return model.DefaultRepoConfig(), nil
		}
		return model.RepoConfig{}, fmt.Errorf("repoconfig: read .anneal.yml: %w", err)
	}
	return parseRepoConfig(data)
}

// parseRepoConfig は .anneal.yml のバイト列を RepoConfig にする。既定値の上に yaml を
// 重ねるため、書かれた項目だけが上書きされる。パース失敗は error として返す（壊れた
// 設定を黙って既定に落とさない）。ローカル / リモート双方の Loader で共有する。
func parseRepoConfig(data []byte) (model.RepoConfig, error) {
	cfg := model.DefaultRepoConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return model.RepoConfig{}, fmt.Errorf("repoconfig: parse .anneal.yml: %w", err)
	}
	return cfg, nil
}
