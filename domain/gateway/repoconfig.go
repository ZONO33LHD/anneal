package gateway

import "github.com/ZONO33LHD/anneal/domain/model"

// RepoConfigLoader はリポジトリごとの .anneal.yml を読み取る（存在しなければデフォルトを返す）。
type RepoConfigLoader interface {
	Load(repoPath string) model.RepoConfig
}
