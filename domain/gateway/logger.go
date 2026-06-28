package gateway

import "context"

// Logger は横断的なロギングのポート。slog 互換の可変長 args（key, value, ... の並び）を
// 受け取り、構造化ログを出せる。ctx を渡すことで相関 ID（trace_id）などを各行へ
// 自動付与できる。Error はエラーとスタックトレースを併せて記録する。
// Step はライフサイクルの遷移ごとに 1 行出力するために使う（非同期フローの観測用）。
type Logger interface {
	Debug(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
	Warn(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, err error, args ...any)
	Step(ctx context.Context, msg string, args ...any)
}
