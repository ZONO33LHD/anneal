package model

// ScoreStatus は評価が確定済みかどうかを追跡する。スコアは異なるタイミングで届くため
// （CI -> review -> merge -> regression）、レコードは partial で始まり、マージ結果が
// 判明した時点で final に昇格する。
type ScoreStatus string

const (
	ScorePartial ScoreStatus = "partial"
	ScoreFinal   ScoreStatus = "final"
)

// AgentEvaluation は 1 つの依存関係更新に対するスコアカードである。各構成要素は 0..100。
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
