package model

import (
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var idCounter atomic.Int64

// NewID は "run_lm0k3f-7" のような可読でプロセス内一意な id を返す。
func NewID(prefix string) string {
	n := idCounter.Add(1)
	stamp := strconv.FormatInt(time.Now().UnixNano(), 36)
	return prefix + "_" + stamp + "-" + strconv.FormatInt(n, 10)
}

var seqCounter atomic.Int64

// NextSeq はプロセス内のイベントを安定して順序付けるための単調増加なシーケンス番号を
// 返す（壁時計のタイムスタンプが衝突しても最新の評価を確実に選び出すために使う）。
func NextSeq() int64 {
	return seqCounter.Add(1)
}

// NowString は現在時刻を RFC3339 文字列として返す。
func NowString() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

var slugInvalid = regexp.MustCompile(`[^a-z0-9._/-]+`)

// Slug は update キー内で使う、安定した小文字のトークンを生成する。
func Slug(s string) string {
	return strings.Trim(slugInvalid.ReplaceAllString(strings.ToLower(s), "-"), "-")
}
