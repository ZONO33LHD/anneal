// Package testutil はテスト向けの共有フィクスチャを提供する。
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/model"
)

// MakeUpdate はテスト向けに妥当なデフォルト値を持つ DependencyUpdate を返す。
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

// CopyFixture は fixtures/sample-repo を一時ディレクトリへコピーし、applyUpdate が
// フィクスチャを変更しないようにする。relRoot はテストパッケージからモジュール
// ルートまでのパスである（例: 2 階層深いパッケージなら "../.."）。
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
