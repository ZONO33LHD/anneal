package usecase_test

import (
	"context"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/internal/testutil"
	"github.com/ZONO33LHD/anneal/usecase"
)

func TestWebhookUsecaseDispatchesMatchingPullRequest(t *testing.T) {
	rec := testutil.MakeUpdate()
	rec.Status = model.StateCIRunning
	rec.PullRequestNumber = 17

	other := testutil.MakeUpdate()
	other.UpdateKey = "other"
	other.PullRequestNumber = 18

	updates := &fakeUpdateRepository{active: []model.DependencyUpdate{other, rec}}
	engine := &fakeEngineUsecase{moved: true}
	uc := usecase.NewWebhookUsecase(updates, engine, webhookNoopLogger{})

	moved, err := uc.Handle(context.Background(), usecase.WebhookEvent{
		Repository:        rec.Repository,
		PullRequestNumber: rec.PullRequestNumber,
		Signal:            usecase.WebhookSignalCIPassed,
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !moved {
		t.Fatal("moved=false, want true")
	}
	if len(engine.dispatched) != 1 {
		t.Fatalf("dispatch calls=%d, want 1", len(engine.dispatched))
	}
	got := engine.dispatched[0]
	if got.UpdateKey != rec.UpdateKey {
		t.Fatalf("dispatch update_key=%s, want %s", got.UpdateKey, rec.UpdateKey)
	}
	if got.Status != model.StateCIPassed {
		t.Fatalf("dispatch status=%s, want %s", got.Status, model.StateCIPassed)
	}
	if len(updates.puts) != 0 {
		t.Fatalf("unexpected Put calls=%d", len(updates.puts))
	}
}

func TestWebhookUsecaseMatchesBranchAndPersistsSignalWhenDispatchNoops(t *testing.T) {
	rec := testutil.MakeUpdate()
	rec.Status = model.StateAwaitingReview
	rec.PullRequestNumber = 0
	rec.Branch = "anneal/npm/axios"

	updates := &fakeUpdateRepository{active: []model.DependencyUpdate{rec}}
	engine := &fakeEngineUsecase{}
	uc := usecase.NewWebhookUsecase(updates, engine, webhookNoopLogger{})

	moved, err := uc.Handle(context.Background(), usecase.WebhookEvent{
		Repository: rec.Repository,
		Branch:     rec.Branch,
		Signal:     usecase.WebhookSignalReviewChangesRequested,
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !moved {
		t.Fatal("moved=false, want true")
	}
	if len(engine.dispatched) != 1 {
		t.Fatalf("dispatch calls=%d, want 1", len(engine.dispatched))
	}
	if engine.dispatched[0].Status != model.StateChangesRequested {
		t.Fatalf("dispatch status=%s, want %s", engine.dispatched[0].Status, model.StateChangesRequested)
	}
	if len(updates.puts) != 1 {
		t.Fatalf("Put calls=%d, want 1", len(updates.puts))
	}
	if updates.puts[0].Status != model.StateChangesRequested {
		t.Fatalf("put status=%s, want %s", updates.puts[0].Status, model.StateChangesRequested)
	}
}

func TestWebhookUsecaseNoopsWhenRecordNotFound(t *testing.T) {
	rec := testutil.MakeUpdate()
	rec.PullRequestNumber = 17

	updates := &fakeUpdateRepository{active: []model.DependencyUpdate{rec}}
	engine := &fakeEngineUsecase{}
	uc := usecase.NewWebhookUsecase(updates, engine, webhookNoopLogger{})

	moved, err := uc.Handle(context.Background(), usecase.WebhookEvent{
		Repository:        rec.Repository,
		PullRequestNumber: 999,
		Signal:            usecase.WebhookSignalPullRequestMerged,
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if moved {
		t.Fatal("moved=true, want false")
	}
	if len(engine.dispatched) != 0 {
		t.Fatalf("dispatch calls=%d, want 0", len(engine.dispatched))
	}
	if len(updates.puts) != 0 {
		t.Fatalf("Put calls=%d, want 0", len(updates.puts))
	}
}

type fakeUpdateRepository struct {
	active []model.DependencyUpdate
	puts   []model.DependencyUpdate
}

func (r *fakeUpdateRepository) Get(string) (*model.DependencyUpdate, error) {
	return nil, nil
}

func (r *fakeUpdateRepository) GetActive(string) (*model.DependencyUpdate, error) {
	return nil, nil
}

func (r *fakeUpdateRepository) Put(rec model.DependencyUpdate) error {
	r.puts = append(r.puts, rec)
	return nil
}

func (r *fakeUpdateRepository) List() ([]model.DependencyUpdate, error) {
	return r.active, nil
}

func (r *fakeUpdateRepository) ListActive() ([]model.DependencyUpdate, error) {
	return r.active, nil
}

type fakeEngineUsecase struct {
	moved      bool
	dispatched []model.DependencyUpdate
}

func (e *fakeEngineUsecase) Dispatch(
	_ context.Context,
	rec model.DependencyUpdate,
) (bool, model.DependencyUpdate, error) {
	e.dispatched = append(e.dispatched, rec)
	return e.moved, rec, nil
}

func (e *fakeEngineUsecase) Tick(context.Context) (int, error) {
	return 0, nil
}

func (e *fakeEngineUsecase) Drive(context.Context, int) error {
	return nil
}

func (e *fakeEngineUsecase) Reconcile(context.Context) (int, error) {
	return 0, nil
}

type webhookNoopLogger struct{}

func (webhookNoopLogger) Debug(context.Context, string, ...any) {
}

func (webhookNoopLogger) Info(context.Context, string, ...any) {
}

func (webhookNoopLogger) Warn(context.Context, string, ...any) {
}

func (webhookNoopLogger) Step(context.Context, string, ...any) {
}

func (webhookNoopLogger) Error(context.Context, string, error, ...any) {
}
