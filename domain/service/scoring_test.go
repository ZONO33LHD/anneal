package service_test

import (
	"testing"

	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/policy"
	"github.com/ZONO33LHD/anneal/domain/service"
	"github.com/ZONO33LHD/anneal/internal/testutil"
)

func TestCleanPassScoresHigh(t *testing.T) {
	u := testutil.MakeUpdate()
	u.Status = model.StateMerged
	u.CI = &model.CIResult{Status: "passed", Attempts: 0}
	zero := 0
	u.ReviewCommentCount = &zero
	u.Impact = &model.ImpactAnalysis{RiskLevel: model.RiskLow, Confidence: 0.9, Summary: "s", UsageSites: []string{"a"}, AffectedFiles: []string{"a"}}
	c := service.ComputeComponents(u)
	if c.CI != 100 || c.Merge != 100 {
		t.Errorf("ci=%v merge=%v", c.CI, c.Merge)
	}
	if service.ComputeTotal(c) < 90 {
		t.Error("expected >90")
	}
}

func TestFailedCIScoresLow(t *testing.T) {
	u := testutil.MakeUpdate()
	u.Status = model.StateClosed
	u.CI = &model.CIResult{Status: "failed", Attempts: 0}
	c := service.ComputeComponents(u)
	if c.CI != 20 || c.Merge != 0 || service.ComputeTotal(c) >= 70 {
		t.Errorf("ci=%v merge=%v total=%v", c.CI, c.Merge, service.ComputeTotal(c))
	}
}

func TestWeightsSumToOne(t *testing.T) {
	w := policy.ScoreWeights
	if sum := w.CI + w.Review + w.Risk + w.PRQuality + w.FixAccuracy + w.Merge; sum < 0.999 || sum > 1.001 {
		t.Errorf("weights sum=%v", sum)
	}
}

func TestPartialThenFinal(t *testing.T) {
	u := testutil.MakeUpdate()
	u.Status = model.StateCIPassed
	partial := service.BuildEvaluation(u, nil)
	if partial.ScoreStatus != model.ScorePartial {
		t.Error("expected partial")
	}
	u.Status = model.StateMerged
	final := service.BuildEvaluation(u, &partial)
	if final.ScoreStatus != model.ScoreFinal || final.RunID != partial.RunID {
		t.Error("expected final with preserved run_id")
	}
}
