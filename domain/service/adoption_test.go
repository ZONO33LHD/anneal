package service

import (
	"testing"

	"github.com/ZONO33LHD/anneal/domain/model"
)

func finalEval(version string, score float64) model.AgentEvaluation {
	return model.AgentEvaluation{
		AgentVersion: version,
		TotalScore:   score,
		ScoreStatus:  model.ScoreFinal,
	}
}

func TestScoreByVersionAveragesOnlyFinalForVersion(t *testing.T) {
	evals := []model.AgentEvaluation{
		finalEval("prompt_v1", 80),
		finalEval("prompt_v1", 60),
		finalEval("prompt_v2", 100),
		{AgentVersion: "prompt_v1", TotalScore: 0, ScoreStatus: model.ScorePartial},
	}
	got := ScoreByVersion(evals, "prompt_v1")
	if got.Count != 2 {
		t.Fatalf("final v1 件数は 2 を期待したが %d", got.Count)
	}
	if got.Average != 70 {
		t.Errorf("平均は 70 を期待したが %v", got.Average)
	}
}

func TestScoreByVersionNoSamples(t *testing.T) {
	got := ScoreByVersion(nil, "prompt_v9")
	if got.Count != 0 || got.Average != 0 {
		t.Errorf("標本なしは Count=0 Average=0 を期待したが %+v", got)
	}
}

func TestEvaluateCanary(t *testing.T) {
	cases := []struct {
		name     string
		canary   VersionStats
		baseline float64
		want     CanaryVerdict
	}{
		{"標本不足は hold", VersionStats{Count: 2, Average: 100}, 70, CanaryHold},
		{"baseline 以上は adopt", VersionStats{Count: 3, Average: 75}, 70, CanaryAdopt},
		{"baseline 同点は adopt", VersionStats{Count: 5, Average: 70}, 70, CanaryAdopt},
		{"margin 超の劣化は rollback", VersionStats{Count: 4, Average: 60}, 70, CanaryRollback},
		{"margin 以内の低下は hold", VersionStats{Count: 4, Average: 68}, 70, CanaryHold},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateCanary(tc.canary, tc.baseline, 3, 3.0)
			if got != tc.want {
				t.Errorf("%s: %q を期待したが %q", tc.name, tc.want, got)
			}
		})
	}
}
