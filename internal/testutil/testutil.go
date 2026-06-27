// Package testutil provides shared fixtures for tests.
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/config"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/registry"
)

// MakeUpdate returns a DependencyUpdate with sensible defaults for tests.
func MakeUpdate() model.DependencyUpdate {
	repo, pkg, target := "acme/demo", "axios", "1.7.0"
	return model.DependencyUpdate{
		UpdateKey:      model.UpdateKey(repo, pkg, target),
		Repository:     repo,
		Ecosystem:      model.EcosystemNPM,
		PackageName:    pkg,
		CurrentVersion: "1.6.2",
		TargetVersion:  target,
		UpdateType:     model.Minor,
		Priority:       model.PriorityMedium,
		RiskLevel:      model.RiskLow,
		Status:         model.StateDetected,
		AgentVersion:   model.CurrentAgentVersion,
		History:        []model.TransitionLog{},
		CreatedAt:      "2026-01-01T00:00:00Z",
		UpdatedAt:      "2026-01-01T00:00:00Z",
	}
}

// NewRegistry builds an all-mock, in-memory registry for tests.
func NewRegistry() *registry.Registry {
	return registry.New(&config.Config{
		StorePath:         "", // in-memory
		ImproveWindow:     5,
		LowScoreThreshold: 90,
		ForceMock:         true,
	})
}

// CopyFixture copies fixtures/sample-repo into a temp dir so applyUpdate never
// mutates the fixture. relRoot is the path from the test package to the module
// root (e.g. "../.." for a package two levels deep).
func CopyFixture(t *testing.T, relRoot string) string {
	t.Helper()
	src := filepath.Join(relRoot, "fixtures", "sample-repo")
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}
