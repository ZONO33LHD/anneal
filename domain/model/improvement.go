package model

// ImprovementStatus は改善候補のライフサイクルである。
// candidate（生成）→ approved（人間承認）→ canary（実運用で試用中）→
// adopted（採用確定）または rolled_back（劣化により巻き戻し）と遷移する。
// approved を挟むのは、LLM 生成文を本番の判断プロンプトに載せる前に人間の承認を
// 必須にするため（生成文が実プロンプトに載り始めるのは canary から）。
type ImprovementStatus string

const (
	ImprovementCandidate  ImprovementStatus = "candidate"
	ImprovementApproved   ImprovementStatus = "approved"
	ImprovementCanary     ImprovementStatus = "canary"
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
// A/B 採用では候補版を canary として試用し、確定スコアでベースライン版と比較して
// 採用（adopted）または巻き戻し（rolled_back）を判断する。
type AgentImprovement struct {
	ImprovementID    string            `json:"improvement_id"`
	Seq              int64             `json:"seq"`
	Trigger          string            `json:"trigger"`
	Target           string            `json:"target"` // prompt|tool|rule|weights
	PreviousVersion  string            `json:"previous_version"`
	CandidateVersion string            `json:"candidate_version"`
	Hypothesis       string            `json:"hypothesis"`
	ProposedChange   string            `json:"proposed_change"`
	Status           ImprovementStatus `json:"status"`
	BaselineScore    float64           `json:"baseline_score,omitempty"`
	CandidateScore   float64           `json:"candidate_score,omitempty"`
	CreatedAt        string            `json:"created_at"`
	CanaryStartedAt  string            `json:"canary_started_at,omitempty"`
	AdoptedAt        string            `json:"adopted_at,omitempty"`
	RolledBackAt     string            `json:"rolled_back_at,omitempty"`
}

// ActiveAgentVersion は現在アクティブなエージェント版を返す。試用中（canary）または
// 採用済み（adopted）で、かつ巻き戻されていない最新の改善版があればその候補版を、
// 無ければコンパイル時の既定版（CurrentAgentVersion）を返す。Seq の大きい順に
// 評価し、最初に見つかった有効な版を採用する。
func ActiveAgentVersion(improvements []AgentImprovement) string {
	latest := AgentImprovement{Seq: -1}
	found := false
	for _, imp := range improvements {
		if imp.Status != ImprovementCanary && imp.Status != ImprovementAdopted {
			continue
		}
		if imp.Seq > latest.Seq {
			latest = imp
			found = true
		}
	}
	if !found {
		return CurrentAgentVersion
	}
	return latest.CandidateVersion
}
