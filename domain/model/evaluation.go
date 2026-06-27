package model

// ScoreStatus tracks whether an evaluation is finalized. Scores arrive at
// different times (CI -> review -> merge -> regression), so a record starts
// partial and is promoted to final once the merge outcome is known.
type ScoreStatus string

const (
	ScorePartial ScoreStatus = "partial"
	ScoreFinal   ScoreStatus = "final"
)

// AgentEvaluation is the scorecard for one dependency update; components are 0..100.
type AgentEvaluation struct {
	RunID             string      `json:"run_id"`
	UpdateKey         string      `json:"update_key"`
	AgentVersion      string      `json:"agent_version"`
	PullRequestNumber int         `json:"pull_request_number,omitempty"`
	CISuccessScore    float64     `json:"ci_success_score"`
	ReviewBurdenScore float64     `json:"review_burden_score"`
	RiskPredictScore  float64     `json:"risk_prediction_score"`
	PRQualityScore    float64     `json:"pr_quality_score"`
	FixAccuracyScore  float64     `json:"fix_accuracy_score"`
	MergeOutcomeScore float64     `json:"merge_outcome_score"`
	RegressionScore   float64     `json:"regression_score"`
	TotalScore        float64     `json:"total_score"`
	ScoreStatus       ScoreStatus `json:"score_status"`
	Seq               int64       `json:"seq"`
	CreatedAt         string      `json:"created_at"`
	UpdatedAt         string      `json:"updated_at"`
}
