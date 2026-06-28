// Package git は git ホストのポートを実装する（モック + GitHub）。
package git

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

// Mock はローカルデモ向けにスクリプト化された git プロバイダー。PR 番号はカウンターから
// 採番される。CI の結果は決定論的で、ShouldFailFirst が設定されている場合は試行 0 が
// 失敗し（修正可能）、それ以降の試行は成功する。これにより
// 「CI が失敗 -> 自己修復 -> 成功」というストーリーを再現する。
type Mock struct {
	prCounter atomic.Int64
}

// NewMock はモックの git プロバイダーを返す。
func NewMock() gateway.Git {
	return &Mock{}
}

func (*Mock) ProviderName() string {
	return "mock"
}

func (m *Mock) CreateBranchAndPR(_ context.Context, opts gateway.CreatePROptions) (gateway.PRRef, error) {
	n := m.prCounter.Add(1) + 1000
	url := fmt.Sprintf("https://github.com/%s/pull/%d", opts.Repository, n)
	fmt.Printf("ℹ mock: opened PR branch=%s url=%s\n", opts.Branch, url)
	return gateway.PRRef{Number: int(n), URL: url}, nil
}

func (*Mock) PushFix(_ context.Context, opts gateway.PushFixOptions) error {
	fmt.Printf("ℹ mock: pushed fix commit pr=%d files=%v\n", opts.PRNumber, opts.ChangedFiles)
	return nil
}

func (*Mock) CheckCI(_ context.Context, opts gateway.CICheckOptions) (gateway.CICheck, error) {
	if opts.ShouldFailFirst && opts.Attempt == 0 {
		branch := strings.ReplaceAll(opts.Branch, "/", "-")
		return gateway.CICheck{
			Complete:   true,
			Passed:     false,
			LogSummary: fmt.Sprintf("npm test failed: 2 tests broke after the bump (%s).", branch),
		}, nil
	}
	return gateway.CICheck{Complete: true, Passed: true}, nil
}
