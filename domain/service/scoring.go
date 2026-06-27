package service

import (
	"math"

	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/policy"
)

// ScoreComponents は各個別スコア（0..100）を保持する。
type ScoreComponents struct {
	CI, Review, Risk, PRQuality, FixAccuracy, Merge, Regression float64
}

func clamp(n float64) float64 {
	return math.Max(0, math.Min(100, math.Round(n)))
}

// ComputeComponents は現在のレコード状態から各スコアを導出する。
func ComputeComponents(u model.DependencyUpdate) ScoreComponents {
	return ScoreComponents{
		CI:          ciScore(u),
		Review:      reviewScore(u),
		Risk:        riskScore(u),
		PRQuality:   prQualityScore(u),
		FixAccuracy: fixAccuracyScore(u),
		Merge:       mergeScore(u),
		Regression:  regressionScore(u),
	}
}

// ComputeTotal は重み付けの合計を返す。Regression は別途追跡される。
func ComputeTotal(c ScoreComponents) float64 {
	w := policy.ScoreWeights
	total := c.CI*w.CI + c.Review*w.Review + c.Risk*w.Risk +
		c.PRQuality*w.PRQuality + c.FixAccuracy*w.FixAccuracy + c.Merge*w.Merge
	return math.Round(total*10) / 10
}

var finalStatuses = map[model.State]bool{
	model.StateMerged: true, model.StateMonitoringRegression: true,
	model.StateDone: true, model.StateRegressed: true, model.StateClosed: true,
}

// BuildEvaluation は評価を構築または更新する。スコアは時間をかけて到着する
// （CI -> review -> merge -> regression）。本関数は各コンポーネントを再計算し、
// 結果が判明した時点で partial -> final へと昇格させる。
func BuildEvaluation(u model.DependencyUpdate, prev *model.AgentEvaluation) model.AgentEvaluation {
	c := ComputeComponents(u)
	status := model.ScorePartial
	if finalStatuses[u.Status] {
		status = model.ScoreFinal
	}
	runID := model.NewID("run")
	createdAt := model.NowString()
	if prev != nil {
		runID = prev.RunID
		createdAt = prev.CreatedAt
	}
	return model.AgentEvaluation{
		RunID:             runID,
		UpdateKey:         u.UpdateKey,
		AgentVersion:      u.AgentVersion,
		PullRequestNumber: u.PullRequestNumber,
		CISuccessScore:    c.CI,
		ReviewBurdenScore: c.Review,
		RiskPredictScore:  c.Risk,
		PRQualityScore:    c.PRQuality,
		FixAccuracyScore:  c.FixAccuracy,
		MergeOutcomeScore: c.Merge,
		RegressionScore:   c.Regression,
		TotalScore:        ComputeTotal(c),
		ScoreStatus:       status,
		Seq:               model.NextSeq(),
		CreatedAt:         createdAt,
		UpdatedAt:         model.NowString(),
	}
}

func ciScore(u model.DependencyUpdate) float64 {
	if u.CI == nil {
		return 50
	}
	switch u.CI.Status {
	case "passed":
		if u.CI.Attempts > 1 {
			return 70 // 修正後の成功は減点される
		}
		return 100
	case "failed":
		return 20
	default:
		return 50
	}
}

func reviewScore(u model.DependencyUpdate) float64 {
	if u.ReviewCommentCount == nil {
		return 70
	}
	return clamp(100 - float64(*u.ReviewCommentCount)*15)
}

func riskScore(u model.DependencyUpdate) float64 {
	if u.Impact == nil {
		return 60
	}
	passed := u.CI != nil && u.CI.Status == "passed"
	cleanPass := passed && (u.CI == nil || u.CI.Attempts <= 1)
	switch u.Impact.RiskLevel {
	case model.RiskLow:
		if cleanPass {
			return 100
		}
		return 40
	case model.RiskHigh:
		if passed {
			return 70
		}
		return 85
	default:
		return 75
	}
}

func prQualityScore(u model.DependencyUpdate) float64 {
	score := 60.0
	if u.Impact != nil {
		score += 15
		if u.Impact.Summary != "" {
			score += 10
		}
	}
	if u.PullRequestURL != "" {
		score += 10
	}
	if u.CVE != nil {
		score += 5
	}
	return clamp(score)
}

func fixAccuracyScore(u model.DependencyUpdate) float64 {
	attempts := 0
	if u.CI != nil {
		attempts = u.CI.Attempts
	}
	if attempts == 0 {
		return 100
	}
	if u.CI != nil && u.CI.Status == "passed" {
		return 80
	}
	return 30
}

func mergeScore(u model.DependencyUpdate) float64 {
	switch u.Status {
	case model.StateMerged, model.StateMonitoringRegression, model.StateDone:
		return 100
	case model.StateChangesRequested:
		return 50
	case model.StateClosed, model.StateRegressed:
		return 0
	default:
		return 50
	}
}

func regressionScore(u model.DependencyUpdate) float64 {
	switch u.Status {
	case model.StateRegressed:
		return 0
	case model.StateDone:
		return 100
	default:
		return 90
	}
}
