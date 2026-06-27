package model

// ImprovementStatus は改善候補のライフサイクルである。このビルドは候補生成までを
// カバーする。A/B 採用は今後の課題である。
type ImprovementStatus string

const (
	ImprovementCandidate  ImprovementStatus = "candidate"
	ImprovementAdopted    ImprovementStatus = "adopted"
	ImprovementRolledBack ImprovementStatus = "rolled_back"
)

// FailureCase は学習材料として保持される低スコアの更新である。
type FailureCase struct {
	UpdateKey    string         `json:"update_key"`
	AgentVersion string         `json:"agent_version"`
	TotalScore   float64        `json:"total_score"`
	Reason       string         `json:"reason"`
	Snapshot     map[string]any `json:"snapshot"`
	CreatedAt    string         `json:"created_at"`
}

// AgentImprovement は Annealing Loop から生成された改善候補である。
type AgentImprovement struct {
	ImprovementID    string            `json:"improvement_id"`
	Trigger          string            `json:"trigger"`
	Target           string            `json:"target"` // prompt|tool|rule|weights
	PreviousVersion  string            `json:"previous_version"`
	CandidateVersion string            `json:"candidate_version"`
	Hypothesis       string            `json:"hypothesis"`
	ProposedChange   string            `json:"proposed_change"`
	Status           ImprovementStatus `json:"status"`
	CreatedAt        string            `json:"created_at"`
}
