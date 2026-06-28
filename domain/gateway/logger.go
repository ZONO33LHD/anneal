package gateway

// Logger は横断的なロギングのポート。slog 互換の可変長 args（key, value, ... の並び）を
// 受け取り、構造化ログを出せる。Error はエラーとスタックトレースを併せて記録する。
// Step はライフサイクルの遷移ごとに 1 行出力するために使う（非同期フローの観測用）。
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, err error, args ...any)
	Step(msg string, args ...any)
}
