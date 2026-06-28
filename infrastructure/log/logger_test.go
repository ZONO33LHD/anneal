package log

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/ctxkey"
)

func TestErrorIncludesStackTrace(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{JSON: true, Writer: &buf})
	l.Error(context.Background(), "boom", errors.New("disk full"), "update_key", "acme/x")

	out := buf.String()
	for _, want := range []string{`"severity":"ERROR"`, `"msg":"boom"`, `"err":"disk full"`, `"stack"`, `"update_key":"acme/x"`} {
		if !strings.Contains(out, want) {
			t.Errorf("error log missing %q in:\n%s", want, out)
		}
	}
	// スタックトレースにはこのテスト関数のフレームが含まれるはず。
	if !strings.Contains(out, "TestErrorIncludesStackTrace") {
		t.Errorf("stack trace should reference the caller frame:\n%s", out)
	}
	// source は logger.go ではなく呼び出し元（このテストファイル）を指すはず。
	if !strings.Contains(out, "logger_test.go") {
		t.Errorf("source should reference the caller file, not the wrapper:\n%s", out)
	}
}

func TestSeverityMapping(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Verbose: true, JSON: true, Writer: &buf})
	ctx := context.Background()
	l.Debug(ctx, "d")
	l.Info(ctx, "i")
	l.Step(ctx, "s")
	l.Warn(ctx, "w")
	out := buf.String()
	for _, want := range []string{`"severity":"DEBUG"`, `"severity":"INFO"`, `"severity":"NOTICE"`, `"severity":"WARNING"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestTraceIDPropagation(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{JSON: true, Writer: &buf})
	ctx := ctxkey.WithTraceID(context.Background(), "trace-abc123")
	l.Info(ctx, "hello")

	out := buf.String()
	if !strings.Contains(out, `"trace_id":"trace-abc123"`) {
		t.Errorf("trace_id from context should be in the log line:\n%s", out)
	}
	// trace ID の無い context では trace_id を付けない。
	buf.Reset()
	l.Info(context.Background(), "no-trace")
	if strings.Contains(buf.String(), "trace_id") {
		t.Errorf("trace_id should be absent without one in context:\n%s", buf.String())
	}
}
