package httpinterface

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/config"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/usecase"
)

func TestInternalTaskHandlerRunsTick(t *testing.T) {
	engine := &fakeInternalTaskEngine{tickChanged: 2}
	anneal := &fakeInternalTaskAnneal{}
	adoption := &fakeInternalTaskAdoption{}
	handler := NewInternalTaskHandler(InternalTaskOptions{
		Token:    "internal-secret",
		Engine:   engine,
		Anneal:   anneal,
		Adoption: adoption,
		Logger:   noopLogger{},
	})
	req := internalTaskRequest("/internal/tick", "internal-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if engine.tickCalls != 1 || anneal.calls != 1 || adoption.calls != 1 {
		t.Fatalf("calls engine=%d anneal=%d adoption=%d", engine.tickCalls, anneal.calls, adoption.calls)
	}
	if !strings.Contains(rec.Body.String(), "advanced=2") {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestInternalTaskHandlerRunsScanTargets(t *testing.T) {
	scan := &fakeInternalTaskScan{results: []usecase.ScanResult{
		{Created: []model.DependencyUpdate{{UpdateKey: "one"}}, Skipped: 1},
		{Skipped: 3},
	}}
	handler := NewInternalTaskHandler(InternalTaskOptions{
		Token: "internal-secret",
		ScanTargets: []config.ScanTarget{
			{Path: "/repo-a", Repository: "acme/repo-a"},
			{Path: "/repo-b"},
		},
		Scan:   scan,
		Logger: noopLogger{},
	})
	req := internalTaskRequest("/internal/scan", "internal-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if scan.calls != 2 {
		t.Fatalf("scan calls=%d, want 2", scan.calls)
	}
	if scan.args[0] != "/repo-a=acme/repo-a" || scan.args[1] != "/repo-b=" {
		t.Fatalf("scan args=%v", scan.args)
	}
	if !strings.Contains(rec.Body.String(), "targets=2 created=1 skipped=4") {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestInternalTaskHandlerRejectsInvalidToken(t *testing.T) {
	engine := &fakeInternalTaskEngine{}
	handler := NewInternalTaskHandler(InternalTaskOptions{
		Token:  "internal-secret",
		Engine: engine,
		Logger: noopLogger{},
	})
	req := internalTaskRequest("/internal/tick", "wrong")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rec.Code)
	}
	if engine.tickCalls != 0 {
		t.Fatal("tick should not run")
	}
}

func TestInternalTaskHandlerAcceptsAuthenticator(t *testing.T) {
	engine := &fakeInternalTaskEngine{tickChanged: 1}
	handler := NewInternalTaskHandler(InternalTaskOptions{
		Authenticator: fakeInternalTaskAuthenticator{ok: true},
		Engine:        engine,
		Anneal:        &fakeInternalTaskAnneal{},
		Adoption:      &fakeInternalTaskAdoption{},
		Logger:        noopLogger{},
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/tick", nil)
	req.Header.Set("Authorization", "Bearer scheduler-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if engine.tickCalls != 1 {
		t.Fatalf("tick calls=%d, want 1", engine.tickCalls)
	}
}

func internalTaskRequest(path, token string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, nil)
	req.Header.Set(internalTokenHeader, token)
	return req
}

type fakeInternalTaskScan struct {
	calls   int
	args    []string
	results []usecase.ScanResult
}

func (f *fakeInternalTaskScan) Run(ctx context.Context, repoArg, repository string) (usecase.ScanResult, error) {
	f.calls++
	f.args = append(f.args, repoArg+"="+repository)
	if len(f.results) < f.calls {
		return usecase.ScanResult{}, nil
	}
	return f.results[f.calls-1], nil
}

type fakeInternalTaskEngine struct {
	tickCalls   int
	tickChanged int
}

func (f *fakeInternalTaskEngine) Dispatch(ctx context.Context, rec model.DependencyUpdate) (bool, model.DependencyUpdate, error) {
	return false, rec, nil
}

func (f *fakeInternalTaskEngine) Tick(ctx context.Context) (int, error) {
	f.tickCalls++
	return f.tickChanged, nil
}

func (f *fakeInternalTaskEngine) Drive(ctx context.Context, maxRounds int) error {
	return nil
}

func (f *fakeInternalTaskEngine) Reconcile(ctx context.Context) (int, error) {
	return 0, nil
}

type fakeInternalTaskAnneal struct {
	calls int
}

func (f *fakeInternalTaskAnneal) MaybeAnneal(ctx context.Context) error {
	f.calls++
	return nil
}

type fakeInternalTaskAdoption struct {
	calls int
}

func (f *fakeInternalTaskAdoption) Evaluate(ctx context.Context) error {
	f.calls++
	return nil
}

type fakeInternalTaskAuthenticator struct {
	ok bool
}

func (f fakeInternalTaskAuthenticator) AuthenticateInternalTask(ctx context.Context, authorization string) error {
	if f.ok {
		return nil
	}
	return context.Canceled
}
