// Package log は Logger のポートを、アイコン接頭辞付きのシンプルなコンソール出力で
// 実装する。これによりデモ中に非同期のライフサイクルを読み取りやすくする。
package log

import (
	"fmt"
	"os"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

// Console は最小限のコンソールロガー。
type Console struct {
	verbose bool
}

// NewConsole はコンソールロガーを返す。verbose は Debug 出力を有効化する。
func NewConsole(verbose bool) gateway.Logger {
	return &Console{verbose: verbose}
}

func (Console) Step(msg string) {
	fmt.Println("→ " + msg)
}
func (Console) Info(msg string) {
	fmt.Println("ℹ " + msg)
}
func (Console) Warn(msg string) {
	fmt.Fprintln(os.Stderr, "⚠ "+msg)
}

func (l Console) Debug(msg string) {
	if l.verbose {
		fmt.Println("· " + msg)
	}
}
