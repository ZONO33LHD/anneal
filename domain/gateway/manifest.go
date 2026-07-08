package gateway

import "context"

// ManifestSource はマニフェスト等のリポジトリ内ファイルを読み取る抽象である。
// ローカル FS（開発・CLI・demo）と GitHub Contents API（Cloud Run・ローカル
// checkout が無い環境）を差し替えるための境界で、Ecosystem の scan 読み取りが
// `os.ReadFile(filepath.Join(repoPath, ...))` に直接依存するのを解消する。
//
// パスは常にリポジトリルートからの相対パス（例 "package.json" / "go.mod"）。
// 取得失敗は「ファイルなし」と取り違えないよう、Exists と ReadFile の双方で
// エラーを握りつぶさず返す（読み取り失敗を「更新なし」と誤認しないため）。
type ManifestSource interface {
	// Exists は相対パスのファイルが存在するかを返す。ファイル不在は (false, nil)、
	// ネットワーク等の取得失敗は (false, err) として区別する。
	Exists(ctx context.Context, relPath string) (bool, error)
	// ReadFile は相対パスのファイル内容を返す。
	ReadFile(ctx context.Context, relPath string) ([]byte, error)
}
