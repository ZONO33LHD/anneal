package model

// ImprovementStatus is the lifecycle of an improvement candidate. This build
// covers up to candidate generation; A/B adoption is future work.
type ImprovementStatus string

const (
	ImprovementCandidate  ImprovementStatus = "candidate"
	ImprovementAdopted    ImprovementStatus = "adopted"
	ImprovementRolledBack ImprovementStatus = "rolled_back"
)

// FailureCase is a low-scoring update kept as learning material.
type FailureCase struct {
	UpdateKey    string         `json:"update_key"`
	AgentVersion string         `json:"agent_version"`
	TotalScore   float64        `json:"total_score"`
	Reason       string         `json:"reason"`
	Snapshot     map[string]any `json:"snapshot"`
	CreatedAt    string         `json:"created_at"`
}

// AgentImprovement is a generated improvement candidate from the Annealing Loop.
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
