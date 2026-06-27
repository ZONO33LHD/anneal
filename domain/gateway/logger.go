package gateway

// Logger は横断的なロギングのポートである。Step はライフサイクルの遷移ごとに 1 行
// 出力するために使われ、デモ中に非同期フローを観測できるようにする。
type Logger interface {
	Step(msg string)
	Info(msg string)
	Warn(msg string)
	Debug(msg string)
}
