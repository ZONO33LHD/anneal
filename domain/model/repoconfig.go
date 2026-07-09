package model

import "strings"

// RepoConfig は .anneal.yml としてコミットされるリポジトリごとのポリシーである。
// 設定の単位はリポジトリ単位で、各 repo の .anneal.yml が
// DefaultRepoConfig（Anneal 組み込みの全 repo 共通の既定）を項目単位で上書きする。
type RepoConfig struct {
	BaseBranch           string   `yaml:"base_branch"`
	Ignore               []string `yaml:"ignore"`
	AutoPRTypes          []string `yaml:"auto_pr_types"`
	RegressionWindowDays int      `yaml:"regression_window_days"`
}

// DefaultRepoConfig は Anneal 組み込みの既定ポリシー（全 repo 共通）を返す。
// .anneal.yml を持たない repo に適用される。組織単位で切り替え可能な既定を持つ仕組みは
// 現状なく（要件 §14 の DB 既定は将来構想）、これはコード定数である。
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
