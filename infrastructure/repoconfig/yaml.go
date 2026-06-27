// Package repoconfig implements the RepoConfigLoader port by reading .anneal.yml.
package repoconfig

import (
	"os"
	"path/filepath"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
	"gopkg.in/yaml.v3"
)

// Loader reads .anneal.yml from a repository.
type Loader struct{}

// NewLoader returns a RepoConfigLoader.
func NewLoader() gateway.RepoConfigLoader {
	return Loader{}
}

// Load reads .anneal.yml, falling back to defaults (a missing file is fine).
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
