package model

import "strings"

// RepoConfig is the per-repository policy committed as .anneal.yml. Repo settings
// override the organisation defaults represented by DefaultRepoConfig.
type RepoConfig struct {
	BaseBranch           string   `yaml:"baseBranch"`
	Ignore               []string `yaml:"ignore"`
	AutoPRTypes          []string `yaml:"autoPrTypes"`
	RegressionWindowDays int      `yaml:"regressionWindowDays"`
}

// DefaultRepoConfig returns the organisation default configuration.
func DefaultRepoConfig() RepoConfig {
	return RepoConfig{
		BaseBranch:           "main",
		Ignore:               []string{},
		AutoPRTypes:          []string{"patch", "minor"},
		RegressionWindowDays: 7,
	}
}

// IsIgnored reports whether the package should be skipped per repo config.
func (c RepoConfig) IsIgnored(packageName string) bool {
	for _, p := range c.Ignore {
		prefix := strings.TrimSuffix(p, "*")
		if packageName == p || strings.HasPrefix(packageName, prefix) {
			return true
		}
	}
	return false
}
