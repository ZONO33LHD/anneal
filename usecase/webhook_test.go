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

func TestAlertUpdateKey(t *testing.T) {
	key, ok := usecase.AlertUpdateKey(usecase.WebhookEvent{
		Repository: "Acme/Repo",
		Signal:     usecase.WebhookSignalAlertDetected,
		Alert: &usecase.WebhookAlert{
			PackageName:   "AXIOS",
			TargetVersion: "1.7.0",
		},
	})
	if !ok {
		t.Fatal("ok=false, want true")
	}
	if key != "acme/repo::axios::1.7.0" {
		t.Fatalf("key=%q", key)
	}
}

func TestAlertUpdateKeyRejectsIncompleteEvent(t *testing.T) {
	_, ok := usecase.AlertUpdateKey(usecase.WebhookEvent{
		Repository: "acme/repo",
		Signal:     usecase.WebhookSignalAlertDetected,
		Alert:      &usecase.WebhookAlert{PackageName: "axios"},
	})
	if ok {
		t.Fatal("ok=true, want false")
	}
}

func TestAlertCVEInfo(t *testing.T) {
	cve := usecase.AlertCVEInfo(usecase.WebhookAlert{
		TargetVersion:    "1.7.0",
		VulnerableRange:  "<1.7.0",
		AdvisoryID:       "GHSA-1234",
		AdvisorySeverity: "HIGH",
		AdvisorySummary:  "summary",
	})
	if cve.ID != "GHSA-1234" || cve.Severity != "high" || cve.PatchedVersion != "1.7.0" {
		t.Fatalf("cve=%#v", cve)
	}
}

func TestWebhookUsecaseCreatesDetectedRecordFromAlert(t *testing.T) {
	updates := &fakeUpdateRepository{}
	engine := &fakeEngineUsecase{}
	uc := usecase.NewWebhookUsecase(updates, engine, webhookNoopLogger{})

	moved, err := uc.Handle(context.Background(), usecase.WebhookEvent{
		Repository: "acme/demo",
		Signal:     usecase.WebhookSignalAlertDetected,
		Alert: &usecase.WebhookAlert{
			PackageName:      "lodash",
			Ecosystem:        model.EcosystemNPM,
			CurrentVersion:   "1.2.0",
			TargetVersion:    "1.2.3",
			VulnerableRange:  "< 1.2.3",
			AdvisoryID:       "CVE-2026-1234",
			AdvisorySeverity: "high",
			AdvisorySummary:  "demo vulnerability",
		},
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !moved {
		t.Fatal("moved=false, want true")
	}
	if len(engine.dispatched) != 0 {
		t.Fatalf("dispatch calls=%d, want 0", len(engine.dispatched))
	}
	if len(updates.puts) != 1 {
		t.Fatalf("Put calls=%d, want 1", len(updates.puts))
	}
	got := updates.puts[0]
	if got.UpdateKey != "acme/demo::lodash::1.2.3" || got.Status != model.StateDetected {
		t.Fatalf("put record=%#v", got)
	}
	if got.Priority != model.PriorityHigh || got.RiskLevel != model.RiskHigh {
		t.Fatalf("priority=%s risk=%s", got.Priority, got.RiskLevel)
	}
	if got.CVE == nil || got.CVE.ID != "CVE-2026-1234" || got.CVE.PatchedVersion != "1.2.3" {
		t.Fatalf("cve=%#v", got.CVE)
	}
}

func TestWebhookUsecaseDoesNotDuplicateActiveAlertRecord(t *testing.T) {
	existing := testutil.MakeUpdate()
	existing.UpdateKey = "acme/demo::lodash::1.2.3"
	updates := &fakeUpdateRepository{activeByKey: map[string]*model.DependencyUpdate{
		existing.UpdateKey: &existing,
	}}
	uc := usecase.NewWebhookUsecase(updates, &fakeEngineUsecase{}, webhookNoopLogger{})

	moved, err := uc.Handle(context.Background(), usecase.WebhookEvent{
		Repository: "acme/demo",
		Signal:     usecase.WebhookSignalAlertDetected,
		Alert: &usecase.WebhookAlert{
			PackageName:   "lodash",
			TargetVersion: "1.2.3",
		},
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if moved {
		t.Fatal("moved=true, want false")
	}
	if len(updates.puts) != 0 {
		t.Fatalf("Put calls=%d, want 0", len(updates.puts))
	}
}

type fakeUpdateRepository struct {
	active      []model.DependencyUpdate
	activeByKey map[string]*model.DependencyUpdate
	puts        []model.DependencyUpdate
}

func (r *fakeUpdateRepository) Get(string) (*model.DependencyUpdate, error) {
	return nil, nil
}

func (r *fakeUpdateRepository) GetActive(key string) (*model.DependencyUpdate, error) {
	if r.activeByKey != nil {
		return r.activeByKey[key], nil
	}
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
