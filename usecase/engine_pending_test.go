package usecase_test

import (
	"context"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/internal/testutil"
	"github.com/ZONO33LHD/anneal/usecase"
)

type pendingGit struct{}

func (pendingGit) ProviderName() string {
	return "pending"
}

func (pendingGit) CreateBranchAndPR(context.Context, gateway.CreatePROptions) (gateway.PRRef, error) {
	return gateway.PRRef{}, nil
}

func (pendingGit) PushFix(context.Context, gateway.PushFixOptions) error {
	return nil
}

func (pendingGit) CheckCI(context.Context, gateway.CICheckOptions) (gateway.CICheck, error) {
	return gateway.CICheck{
		Complete:   false,
		Passed:     false,
		LogSummary: "checks not complete: test",
	}, nil
}

type noopLogger struct{}

func (noopLogger) Debug(context.Context, string, ...any) {
}

func (noopLogger) Info(context.Context, string, ...any) {
}

func (noopLogger) Warn(context.Context, string, ...any) {
}

func (noopLogger) Error(context.Context, string, error, ...any) {
}

func (noopLogger) Step(context.Context, string, ...any) {
}

func TestEngineWaitsWhenCIPending(t *testing.T) {
	rec := testutil.MakeUpdate()
	rec.Status = model.StateCIRunning
	rec.Branch = "anneal/npm/axios"
	rec.PullRequestNumber = 42
	rec.CI = &model.CIResult{Status: "running", Attempts: 0}

	engine := usecase.NewEngineUsecase(
		nil,
		nil,
		nil,
		nil,
		pendingGit{},
		nil,
		nil,
		nil,
		noopLogger{},
		90,
		false,
	)
	moved, out, err := engine.Dispatch(context.Background(), rec)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if moved {
		t.Fatal("pending CI should not advance lifecycle")
	}
	if out.Status != model.StateCIRunning {
		t.Fatalf("status=%s, want %s", out.Status, model.StateCIRunning)
	}
	if !sameHistory(rec.History, out.History) {
		t.Fatalf("history changed: before=%+v after=%+v", rec.History, out.History)
	}
}

func sameHistory(a, b []model.TransitionLog) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
