package usecase_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/internal/testutil"
	"github.com/ZONO33LHD/anneal/registry"
)

func setup(t *testing.T) (*registry.Registry, string) {
	t.Helper()
	reg := testutil.NewRegistry()
	work := testutil.CopyFixture(t, filepath.Join(".."))
	return reg, work
}

func find(t *testing.T, reg *registry.Registry, pkg string) model.DependencyUpdate {
	t.Helper()
	updates, _ := reg.Updates.List()
	for _, u := range updates {
		if u.PackageName == pkg {
			return u
		}
	}
	t.Fatalf("update for %s not found", pkg)
	return model.DependencyUpdate{}
}

func hasTransition(rec model.DependencyUpdate, to model.State) bool {
	for _, h := range rec.History {
		if h.To == to {
			return true
		}
	}
	return false
}

func mustScan(t *testing.T, ctx context.Context, reg *registry.Registry, work string) {
	t.Helper()
	if _, err := reg.Scan.Run(ctx, work, "acme/sample-repo"); err != nil {
		t.Fatalf("scan: %v", err)
	}
}

func mustDrive(t *testing.T, ctx context.Context, reg *registry.Registry) {
	t.Helper()
	if err := reg.Engine.Drive(ctx, 100); err != nil {
		t.Fatalf("drive: %v", err)
	}
}

func TestLifecycleDetect(t *testing.T) {
	reg, work := setup(t)
	res, err := reg.Scan.Run(context.Background(), work, "acme/sample-repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) < 6 {
		t.Errorf("expected >=6 candidates, got %d", len(res.Created))
	}
}

func TestLifecycleIdempotent(t *testing.T) {
	reg, work := setup(t)
	ctx := context.Background()
	if _, err := reg.Scan.Run(ctx, work, "acme/sample-repo"); err != nil {
		t.Fatal(err)
	}
	second, _ := reg.Scan.Run(ctx, work, "acme/sample-repo")
	if len(second.Created) != 0 {
		t.Errorf("re-scan created %d duplicates", len(second.Created))
	}
}

func TestLifecycleSecurityPatchToDone(t *testing.T) {
	reg, work := setup(t)
	ctx := context.Background()
	mustScan(t, ctx, reg, work)
	if err := reg.Engine.Drive(ctx, 100); err != nil {
		t.Fatal(err)
	}
	lodash := find(t, reg, "lodash")
	if lodash.Status != model.StateDone {
		t.Errorf("lodash status=%s", lodash.Status)
	}
	eval, _ := reg.Evaluations.Get(lodash.UpdateKey)
	if eval == nil || eval.ScoreStatus != model.ScoreFinal || eval.TotalScore <= 90 {
		t.Errorf("lodash eval=%+v", eval)
	}
}

func TestLifecycleMajorNeedsApproval(t *testing.T) {
	reg, work := setup(t)
	ctx := context.Background()
	mustScan(t, ctx, reg, work)
	mustDrive(t, ctx, reg)
	if !hasTransition(find(t, reg, "chalk"), model.StateAwaitingApproval) {
		t.Error("chalk (major) should pass through awaiting_approval")
	}
}

func TestLifecycleSelfHeal(t *testing.T) {
	reg, work := setup(t)
	ctx := context.Background()
	mustScan(t, ctx, reg, work)
	mustDrive(t, ctx, reg)
	axios := find(t, reg, "axios")
	if !hasTransition(axios, model.StateCIFailed) ||
		!hasTransition(axios, model.StateFixing) ||
		!hasTransition(axios, model.StateCIPassed) {
		t.Errorf("axios did not self-heal: %+v", axios.History)
	}
}

func TestLifecycleAnnealingFires(t *testing.T) {
	reg, work := setup(t)
	ctx := context.Background()
	mustScan(t, ctx, reg, work)
	mustDrive(t, ctx, reg)
	if err := reg.Anneal.MaybeAnneal(ctx); err != nil {
		t.Fatal(err)
	}
	imps, _ := reg.Improvements.ListImprovements()
	if len(imps) < 1 {
		t.Error("expected at least one improvement candidate")
	}
}
