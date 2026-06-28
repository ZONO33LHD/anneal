// Package log は Logger ポートを log/slog で実装する。ローカルでは読みやすい text、
// 本番（Cloud Run 等）では severity 付き JSON を出力でき、各行に呼び出し元
// （source: file:line）を付ける。Error はエラーとスタックトレースを添えて記録する。
package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

// levelStep は Step 用の独自レベル（Info と Warn の中間）。severity では NOTICE に対応。
const levelStep = slog.LevelInfo + 2

// Options はロガーの構成。
type Options struct {
	Verbose bool      // true で Debug を有効化
	JSON    bool      // true で JSON（GCP 向け severity 付き）、false で text
	Writer  io.Writer // 出力先。nil なら os.Stdout（テストで差し替え可能）
}

type slogLogger struct {
	h slog.Handler
}

// New は slog ベースの Logger を構築する。
func New(opts Options) gateway.Logger {
	level := slog.LevelInfo
	if opts.Verbose {
		level = slog.LevelDebug
	}
	w := opts.Writer
	if w == nil {
		w = os.Stdout
	}
	ho := &slog.HandlerOptions{Level: level, AddSource: true, ReplaceAttr: replaceAttr}
	var h slog.Handler
	if opts.JSON {
		h = slog.NewJSONHandler(w, ho)
	} else {
		h = slog.NewTextHandler(w, ho)
	}
	return &slogLogger{h: h}
}

// log はハンドラを直接呼び、呼び出し元（このラッパーの 1 つ上）の PC を記録する。
// これにより source が logger.go ではなく実際の呼び出し箇所を指す。
func (s *slogLogger) log(level slog.Level, msg string, args ...any) {
	ctx := context.Background()
	if !s.h.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // 0: Callers, 1: log, 2: 公開メソッド, 3: 呼び出し元
	rec := slog.NewRecord(time.Now(), level, msg, pcs[0])
	rec.Add(args...)
	_ = s.h.Handle(ctx, rec)
}

func (s *slogLogger) Debug(msg string, args ...any) { s.log(slog.LevelDebug, msg, args...) }
func (s *slogLogger) Info(msg string, args ...any)  { s.log(slog.LevelInfo, msg, args...) }
func (s *slogLogger) Warn(msg string, args ...any)  { s.log(slog.LevelWarn, msg, args...) }
func (s *slogLogger) Step(msg string, args ...any)  { s.log(levelStep, msg, args...) }

// Error はエラーとスタックトレースを添えて記録する。stack は呼び出し時点の
// goroutine スタックで、障害発生箇所の追跡に使える。
func (s *slogLogger) Error(msg string, err error, args ...any) {
	args = append(args, slog.Any("err", err), slog.String("stack", string(debug.Stack())))
	s.log(slog.LevelError, msg, args...)
}

// replaceAttr は level を GCP Cloud Logging が解釈する severity に変換する。
func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		return slog.String("severity", severityOf(a.Value.Any().(slog.Level)))
	}
	return a
}

func severityOf(l slog.Level) string {
	switch {
	case l <= slog.LevelDebug:
		return "DEBUG"
	case l < levelStep:
		return "INFO"
	case l < slog.LevelWarn:
		return "NOTICE"
	case l < slog.LevelError:
		return "WARNING"
	default:
		return "ERROR"
	}
}
