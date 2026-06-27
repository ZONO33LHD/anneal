package gateway

import "github.com/ZONO33LHD/anneal/domain/model"

// RepoConfigLoader reads the per-repository .anneal.yml (or returns defaults).
type RepoConfigLoader interface {
	Load(repoPath string) model.RepoConfig
}
