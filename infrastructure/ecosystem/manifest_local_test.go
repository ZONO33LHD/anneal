package ecosystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// LocalFS は既存の os.ReadFile/os.Stat と同じ相対パス解決を保たねばならない。
// ここがズレると PR #2 で Ecosystem.Scan を ManifestSource 経由に切り替えた瞬間に
// scan の読み取り結果が変わってしまう（挙動不変の担保）。
func TestLocalFS_ReadsAndDetectsRelativePaths(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"x":1}`), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	src := NewLocalFS(root)
	ctx := context.Background()

	ok, err := src.Exists(ctx, "package.json")
	if err != nil || !ok {
		t.Fatalf("Exists(package.json)=%v,%v want true,nil", ok, err)
	}
	data, err := src.ReadFile(ctx, "package.json")
	if err != nil || string(data) != `{"x":1}` {
		t.Fatalf("ReadFile=%q,%v", data, err)
	}
}

// 不在ファイルは (false, nil)。取得失敗（err）と区別できることが、リモート実装で
// 「取得失敗を更新なしと誤認しない」ための契約になる。
func TestLocalFS_MissingFileIsNotAnError(t *testing.T) {
	src := NewLocalFS(t.TempDir())
	ctx := context.Background()

	ok, err := src.Exists(ctx, "go.mod")
	if err != nil {
		t.Fatalf("Exists err=%v want nil", err)
	}
	if ok {
		t.Fatal("Exists=true want false for missing file")
	}
	if _, err := src.ReadFile(ctx, "go.mod"); err == nil {
		t.Fatal("ReadFile of missing file should error")
	}
}
