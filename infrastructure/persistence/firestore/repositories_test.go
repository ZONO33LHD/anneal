package firestore_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ZONO33LHD/anneal/domain/model"
	firestorep "github.com/ZONO33LHD/anneal/infrastructure/persistence/firestore"
	"github.com/ZONO33LHD/anneal/internal/testutil"
)

func TestRepositoriesRoundTripWithEmulator(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := context.Background()
	client, err := firestorep.NewClient(ctx, "anneal-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	prefix := "test_" + strings.ReplaceAll(t.Name(), "/", "_") + "_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	opt := firestorep.WithCollectionPrefix(prefix)
	updates := firestorep.NewUpdateRepositoryWithOptions(client, opt)
	evals := firestorep.NewEvaluationRepositoryWithOptions(client, opt)
	improvements := firestorep.NewImprovementRepositoryWithOptions(client, opt)

	active := testutil.MakeUpdate()
	active.Status = model.StatePRCreated
	if err := updates.Put(active); err != nil {
		t.Fatal(err)
	}
	got, err := updates.Get(active.UpdateKey)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.UpdateKey != active.UpdateKey {
		t.Fatalf("Get() = %+v, want update", got)
	}
	got, err = updates.GetActive(active.UpdateKey)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("GetActive() should return active update")
	}

	done := testutil.MakeUpdate()
	done.PackageName = "lodash"
	done.TargetVersion = "5.0.0"
	done.UpdateKey = model.UpdateKey(done.Repository, done.PackageName, done.TargetVersion)
	done.Status = model.StateDone
	if err := updates.Put(done); err != nil {
		t.Fatal(err)
	}
	got, err = updates.GetActive(done.UpdateKey)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("GetActive(done) = %+v, want nil", got)
	}
	missing, err := updates.Get("missing/update::axios::9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	if missing != nil {
		t.Fatalf("Get(missing) = %+v, want nil", missing)
	}
	all, err := updates.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("List() len = %d, want 2", len(all))
	}
	activeList, err := updates.ListActive()
	if err != nil {
		t.Fatal(err)
	}
	if len(activeList) != 1 || activeList[0].UpdateKey != active.UpdateKey {
		t.Fatalf("ListActive() = %+v, want active update only", activeList)
	}

	eval := model.AgentEvaluation{
		RunID:        model.NewID("run"),
		UpdateKey:    active.UpdateKey,
		AgentVersion: model.CurrentAgentVersion,
		TotalScore:   88,
		ScoreStatus:  model.ScoreFinal,
		Seq:          model.NextSeq(),
		CreatedAt:    model.NowString(),
		UpdatedAt:    model.NowString(),
	}
	if err := evals.Put(eval); err != nil {
		t.Fatal(err)
	}
	gotEval, err := evals.Get(active.UpdateKey)
	if err != nil {
		t.Fatal(err)
	}
	if gotEval == nil || gotEval.RunID != eval.RunID {
		t.Fatalf("Get eval = %+v, want %+v", gotEval, eval)
	}
	evalList, err := evals.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(evalList) != 1 {
		t.Fatalf("List eval len = %d, want 1", len(evalList))
	}
	missingEval, err := evals.Get("missing/update::axios::9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	if missingEval != nil {
		t.Fatalf("Get missing eval = %+v, want nil", missingEval)
	}

	failure := model.FailureCase{
		UpdateKey:    active.UpdateKey,
		AgentVersion: model.CurrentAgentVersion,
		TotalScore:   42,
		Reason:       "low score",
		Snapshot:     map[string]any{"package": active.PackageName},
		CreatedAt:    model.NowString(),
	}
	if err := improvements.PutFailure(failure); err != nil {
		t.Fatal(err)
	}
	failures, err := improvements.ListFailures()
	if err != nil {
		t.Fatal(err)
	}
	if len(failures) != 1 || failures[0].UpdateKey != failure.UpdateKey {
		t.Fatalf("ListFailures() = %+v, want failure", failures)
	}

	improvement := model.AgentImprovement{
		ImprovementID:    model.NewID("imp"),
		Trigger:          "low score",
		Target:           "prompt",
		PreviousVersion:  model.CurrentAgentVersion,
		CandidateVersion: "agent-test",
		Hypothesis:       "better scoring",
		ProposedChange:   "tighten prompt",
		Status:           model.ImprovementCandidate,
		CreatedAt:        model.NowString(),
	}
	if err := improvements.PutImprovement(improvement); err != nil {
		t.Fatal(err)
	}
	improvementList, err := improvements.ListImprovements()
	if err != nil {
		t.Fatal(err)
	}
	if len(improvementList) != 1 || improvementList[0].ImprovementID != improvement.ImprovementID {
		t.Fatalf("ListImprovements() = %+v, want improvement", improvementList)
	}
}
