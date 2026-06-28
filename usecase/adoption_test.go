package usecase_test

import (
	"context"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/infrastructure/notify"
	"github.com/ZONO33LHD/anneal/infrastructure/persistence"
	"github.com/ZONO33LHD/anneal/usecase"
)

func newAdoptionUsecase(t *testing.T) (usecase.AdoptionUsecase, *persistence.DB) {
	t.Helper()
	db := persistence.NewDB("")
	evals := persistence.NewEvaluationRepository(db)
	improvements := persistence.NewImprovementRepository(db)
	uc := usecase.NewAdoptionUsecase(evals, improvements, notify.NewConsole(), webhookNoopLogger{}, 3, 3.0)
	return uc, db
}

func putFinalEval(t *testing.T, db *persistence.DB, key, version string, score float64) {
	t.Helper()
	repo := persistence.NewEvaluationRepository(db)
	if err := repo.Put(model.AgentEvaluation{
		UpdateKey:    key,
		AgentVersion: version,
		TotalScore:   score,
		ScoreStatus:  model.ScoreFinal,
		Seq:          model.NextSeq(),
	}); err != nil {
		t.Fatalf("put eval: %v", err)
	}
}

func putImprovement(t *testing.T, db *persistence.DB, imp model.AgentImprovement) {
	t.Helper()
	if err := persistence.NewImprovementRepository(db).PutImprovement(imp); err != nil {
		t.Fatalf("put improvement: %v", err)
	}
}

func onlyImprovement(t *testing.T, db *persistence.DB) model.AgentImprovement {
	t.Helper()
	imps, err := persistence.NewImprovementRepository(db).ListImprovements()
	if err != nil {
		t.Fatalf("list improvements: %v", err)
	}
	if len(imps) != 1 {
		t.Fatalf("improvements=%d, want 1", len(imps))
	}
	return imps[0]
}

func TestAdoptionPromotesCandidateToCanary(t *testing.T) {
	uc, db := newAdoptionUsecase(t)
	// 現行版 prompt_v1 のベースライン実績。
	putFinalEval(t, db, "a", "prompt_v1", 80)
	putFinalEval(t, db, "b", "prompt_v1", 70)
	putImprovement(t, db, model.AgentImprovement{
		ImprovementID:    "imp1",
		Seq:              model.NextSeq(),
		PreviousVersion:  "prompt_v1",
		CandidateVersion: "prompt_v2",
		Status:           model.ImprovementCandidate,
	})

	if err := uc.Evaluate(context.Background()); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	got := onlyImprovement(t, db)
	if got.Status != model.ImprovementCanary {
		t.Fatalf("status=%s, want canary", got.Status)
	}
	if got.BaselineScore != 75 {
		t.Errorf("baseline=%v, want 75", got.BaselineScore)
	}
}

func TestAdoptionAdoptsWhenCanaryBeatsBaseline(t *testing.T) {
	uc, db := newAdoptionUsecase(t)
	putImprovement(t, db, model.AgentImprovement{
		ImprovementID:    "imp1",
		Seq:              model.NextSeq(),
		PreviousVersion:  "prompt_v1",
		CandidateVersion: "prompt_v2",
		Status:           model.ImprovementCanary,
		BaselineScore:    75,
	})
	// 候補版 prompt_v2 の確定スコアがベースラインを上回る（minSample=3 を満たす）。
	putFinalEval(t, db, "c", "prompt_v2", 90)
	putFinalEval(t, db, "d", "prompt_v2", 85)
	putFinalEval(t, db, "e", "prompt_v2", 80)

	if err := uc.Evaluate(context.Background()); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	got := onlyImprovement(t, db)
	if got.Status != model.ImprovementAdopted {
		t.Fatalf("status=%s, want adopted", got.Status)
	}
	if got.AdoptedAt == "" {
		t.Error("AdoptedAt should be set")
	}
}

func TestAdoptionRollsBackWhenCanaryRegresses(t *testing.T) {
	uc, db := newAdoptionUsecase(t)
	putImprovement(t, db, model.AgentImprovement{
		ImprovementID:    "imp1",
		Seq:              model.NextSeq(),
		PreviousVersion:  "prompt_v1",
		CandidateVersion: "prompt_v2",
		Status:           model.ImprovementCanary,
		BaselineScore:    80,
	})
	// 候補版が baseline-margin を超えて劣化。
	putFinalEval(t, db, "c", "prompt_v2", 60)
	putFinalEval(t, db, "d", "prompt_v2", 65)
	putFinalEval(t, db, "e", "prompt_v2", 62)

	if err := uc.Evaluate(context.Background()); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	got := onlyImprovement(t, db)
	if got.Status != model.ImprovementRolledBack {
		t.Fatalf("status=%s, want rolled_back", got.Status)
	}
	if got.RolledBackAt == "" {
		t.Error("RolledBackAt should be set")
	}
}

func TestAdoptionHoldsWhenSamplesInsufficient(t *testing.T) {
	uc, db := newAdoptionUsecase(t)
	putImprovement(t, db, model.AgentImprovement{
		ImprovementID:    "imp1",
		Seq:              model.NextSeq(),
		CandidateVersion: "prompt_v2",
		Status:           model.ImprovementCanary,
		BaselineScore:    80,
	})
	putFinalEval(t, db, "c", "prompt_v2", 95) // 1 件だけ（minSample=3 未満）

	if err := uc.Evaluate(context.Background()); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	got := onlyImprovement(t, db)
	if got.Status != model.ImprovementCanary {
		t.Fatalf("status=%s, want canary (hold)", got.Status)
	}
}
