// Package git implements the git-host port (mock + GitHub).
package git

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

// Mock is a scripted git provider for local demos. PR numbers come from a
// counter; CI outcome is deterministic: when ShouldFailFirst is set, attempt 0
// fails (fixable) and later attempts pass — reproducing the
// "CI fails -> self-heal -> passes" story.
type Mock struct {
	prCounter atomic.Int64
}

// NewMock returns a mock git provider.
func NewMock() gateway.Git {
	return &Mock{}
}

func (*Mock) Name() string {
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
			Passed:     false,
			LogSummary: fmt.Sprintf("npm test failed: 2 tests broke after the bump (%s).", branch),
		}, nil
	}
	return gateway.CICheck{Passed: true}, nil
}
