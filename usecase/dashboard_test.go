package usecase_test

import (
	"context"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/infrastructure/persistence"
	"github.com/ZONO33LHD/anneal/usecase"
)

func TestDashboardSnapshotAggregates(t *testing.T) {
	db := persistence.NewDB("")
	updates := persistence.NewUpdateRepository(db)
	evals := persistence.NewEvaluationRepository(db)
	improvements := persistence.NewImprovementRepository(db)

	mustPut := func(status model.State, key string) {
		if err := updates.Put(model.DependencyUpdate{UpdateKey: key, Status: status}); err != nil {
			t.Fatalf("put update: %v", err)
		}
	}
	mustPut(model.StateMerged, "r::a::1")
	mustPut(model.StateMerged, "r::b::1")
	mustPut(model.StateCIRunning, "r::c::1")

	if err := evals.Put(model.AgentEvaluation{
		UpdateKey: "r::a::1", AgentVersion: "prompt_v1", TotalScore: 80,
		ScoreStatus: model.ScoreFinal, Seq: model.NextSeq(),
	}); err != nil {
		t.Fatalf("put eval: %v", err)
	}
	if err := evals.Put(model.AgentEvaluation{
		UpdateKey: "r::b::1", AgentVersion: "prompt_v1", TotalScore: 60,
		ScoreStatus: model.ScoreFinal, Seq: model.NextSeq(),
	}); err != nil {
		t.Fatalf("put eval: %v", err)
	}
	if err := improvements.PutImprovement(model.AgentImprovement{
		ImprovementID: "imp1", Seq: model.NextSeq(),
		CandidateVersion: "prompt_v2", Status: model.ImprovementCanary, BaselineScore: 70,
	}); err != nil {
		t.Fatalf("put improvement: %v", err)
	}

	uc := usecase.NewDashboardUsecase(updates, evals, improvements)
	view, err := uc.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	if view.TotalUpdates != 3 {
		t.Errorf("TotalUpdates=%d, want 3", view.TotalUpdates)
	}
	// canary が有効なのでアクティブ版は候補版になる。
	if view.ActiveVersion != "prompt_v2" {
		t.Errorf("ActiveVersion=%q, want prompt_v2", view.ActiveVersion)
	}
	if !view.HasFinalScores || view.AverageScore != 70 {
		t.Errorf("average=%v has=%v, want 70/true", view.AverageScore, view.HasFinalScores)
	}
	// 状態分布は件数の多い順。merged(2) が先頭。
	if len(view.StateCounts) == 0 || view.StateCounts[0].State != "merged" || view.StateCounts[0].Count != 2 {
		t.Errorf("top state=%+v, want merged/2", view.StateCounts)
	}
	if len(view.RecentScores) != 2 {
		t.Errorf("RecentScores=%d, want 2", len(view.RecentScores))
	}
	if len(view.Improvements) != 1 || view.Improvements[0].Version != "prompt_v2" {
		t.Errorf("Improvements=%+v, want 1 (prompt_v2)", view.Improvements)
	}
}

func TestDashboardSnapshotEmpty(t *testing.T) {
	db := persistence.NewDB("")
	uc := usecase.NewDashboardUsecase(
		persistence.NewUpdateRepository(db),
		persistence.NewEvaluationRepository(db),
		persistence.NewImprovementRepository(db),
	)
	view, err := uc.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if view.TotalUpdates != 0 || view.HasFinalScores {
		t.Errorf("empty snapshot unexpected: %+v", view)
	}
	if view.ActiveVersion != model.CurrentAgentVersion {
		t.Errorf("ActiveVersion=%q, want %q", view.ActiveVersion, model.CurrentAgentVersion)
	}
}
