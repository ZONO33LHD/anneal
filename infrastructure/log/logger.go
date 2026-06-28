// Package log は Logger ポートを log/slog で実装する。ローカルでは読みやすい text、
// 本番（Cloud Run 等）では severity 付き JSON を出力でき、Error はスタックトレースを
// 添えて記録する。
package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime/debug"

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
	l *slog.Logger
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
	ho := &slog.HandlerOptions{Level: level, ReplaceAttr: replaceAttr}
	var h slog.Handler
	if opts.JSON {
		h = slog.NewJSONHandler(w, ho)
	} else {
		h = slog.NewTextHandler(w, ho)
	}
	return &slogLogger{l: slog.New(h)}
}

func (s *slogLogger) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }
func (s *slogLogger) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s *slogLogger) Warn(msg string, args ...any)  { s.l.Warn(msg, args...) }

func (s *slogLogger) Step(msg string, args ...any) {
	s.l.Log(context.Background(), levelStep, msg, args...)
}

// Error はエラーとスタックトレースを添えて記録する。stack は呼び出し時点の
// goroutine スタックで、障害発生箇所の追跡に使える。
func (s *slogLogger) Error(msg string, err error, args ...any) {
	args = append(args, slog.Any("err", err), slog.String("stack", string(debug.Stack())))
	s.l.Error(msg, args...)
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
