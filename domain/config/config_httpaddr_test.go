package config

import (
	"testing"
)

func TestDefaultHTTPAddrUsesPort(t *testing.T) {
	t.Setenv("PORT", "9090")
	if got := defaultHTTPAddr(); got != ":9090" {
		t.Errorf("PORT が設定されている場合は :9090 を期待したが %q だった", got)
	}
}

func TestDefaultHTTPAddrFallsBackTo8080(t *testing.T) {
	// PORT を空にしてフォールバックを検証する。
	t.Setenv("PORT", "")
	if got := defaultHTTPAddr(); got != ":8080" {
		t.Errorf("PORT 未設定では :8080 を期待したが %q だった", got)
	}
}
