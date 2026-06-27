// Package log implements the Logger port with simple, icon-prefixed console
// output, so the asynchronous lifecycle is readable during demos.
package log

import (
	"fmt"
	"os"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

// Console is a minimal console logger.
type Console struct {
	verbose bool
}

// NewConsole returns a console logger. verbose enables Debug output.
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
