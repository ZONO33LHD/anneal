// Package ctxkey は context に載せる相関キーを定義する。トレース ID を context で
// 伝播させ、ログ全体を 1 つの操作（CLI 実行・tick・将来の HTTP リクエスト）として
// 相関できるようにする。
package ctxkey

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type traceIDKeyType struct{}

var traceIDKey traceIDKeyType

// NewTraceID は 16 バイトのランダム ID を 32 桁の hex で返す（GCP Cloud Trace の
// trace id と同じ形式）。
func NewTraceID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// WithTraceID は context にトレース ID を載せる。
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

// TraceID は context からトレース ID を取り出す（無ければ空文字）。
func TraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}
	return ""
}

// EnsureTraceID は既存のトレース ID を返すか、無ければ新規生成して載せた context を返す。
// 既に上流（将来の HTTP ミドルウェア等）で設定済みならそれを尊重する。
func EnsureTraceID(ctx context.Context) (context.Context, string) {
	if id := TraceID(ctx); id != "" {
		return ctx, id
	}
	id := NewTraceID()
	return WithTraceID(ctx, id), id
}
