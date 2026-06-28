package log

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestErrorIncludesStackTrace(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{JSON: true, Writer: &buf})
	l.Error("boom", errors.New("disk full"), "update_key", "acme/x")

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
	l.Debug("d")
	l.Info("i")
	l.Step("s")
	l.Warn("w")
	out := buf.String()
	for _, want := range []string{`"severity":"DEBUG"`, `"severity":"INFO"`, `"severity":"NOTICE"`, `"severity":"WARNING"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
