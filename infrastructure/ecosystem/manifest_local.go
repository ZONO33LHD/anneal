package ecosystem

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

// LocalFS はローカルファイルシステム上の repo ルートを読む ManifestSource。
// 現行の `os.ReadFile(filepath.Join(repoPath, ...))` と同じ挙動を保つ（開発・CLI・
// demo 向け）。
type LocalFS struct {
	root string
}

// NewLocalFS は指定ルート配下を読む ManifestSource を返す。
func NewLocalFS(root string) gateway.ManifestSource {
	return LocalFS{root: root}
}

func (l LocalFS) Exists(_ context.Context, relPath string) (bool, error) {
	_, err := os.Stat(filepath.Join(l.root, relPath))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (l LocalFS) ReadFile(_ context.Context, relPath string) ([]byte, error) {
	return os.ReadFile(filepath.Join(l.root, relPath))
}
