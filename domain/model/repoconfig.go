package model

import "strings"

// RepoConfig は .anneal.yml としてコミットされるリポジトリごとのポリシーである。
// リポジトリの設定は DefaultRepoConfig が表す組織のデフォルトを上書きする。
type RepoConfig struct {
	BaseBranch           string   `yaml:"base_branch"`
	Ignore               []string `yaml:"ignore"`
	AutoPRTypes          []string `yaml:"auto_pr_types"`
	RegressionWindowDays int      `yaml:"regression_window_days"`
}

// DefaultRepoConfig は組織のデフォルト設定を返す。
func DefaultRepoConfig() RepoConfig {
	return RepoConfig{
		BaseBranch:           "main",
		Ignore:               []string{},
		AutoPRTypes:          []string{"patch", "minor"},
		RegressionWindowDays: 7,
	}
}

// IsIgnored はリポジトリ設定に従ってそのパッケージをスキップすべきかどうかを返す。
// 末尾に "*" が付くパターンのみ前方一致、それ以外は完全一致で判定する
// （"react" が "reactive-lib" まで誤って無視するのを防ぐ）。
func (c RepoConfig) IsIgnored(packageName string) bool {
	for _, p := range c.Ignore {
		if prefix, ok := strings.CutSuffix(p, "*"); ok {
			if strings.HasPrefix(packageName, prefix) {
				return true
			}
		} else if packageName == p {
			return true
		}
	}
	return false
}
