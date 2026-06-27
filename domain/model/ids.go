package model

import (
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var idCounter atomic.Int64

// NewID returns a readable, process-unique id like "run_lm0k3f-7".
func NewID(prefix string) string {
	n := idCounter.Add(1)
	stamp := strconv.FormatInt(time.Now().UnixNano(), 36)
	return prefix + "_" + stamp + "-" + strconv.FormatInt(n, 10)
}

var seqCounter atomic.Int64

// NextSeq returns a monotonic sequence number for stable ordering of events
// within a process (used to pick the most recent evaluations reliably even when
// wall-clock timestamps collide).
func NextSeq() int64 {
	return seqCounter.Add(1)
}

// NowString is the current time as an RFC3339 string.
func NowString() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

var slugInvalid = regexp.MustCompile(`[^a-z0-9._/-]+`)

// Slug produces a stable, lowercased token used inside update keys.
func Slug(s string) string {
	return strings.Trim(slugInvalid.ReplaceAllString(strings.ToLower(s), "-"), "-")
}
