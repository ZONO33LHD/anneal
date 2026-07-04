package model

import "fmt"

// Ecosystem はパッケージのエコシステムを識別する。
type Ecosystem string

const (
	EcosystemNPM  Ecosystem = "npm"
	EcosystemGo   Ecosystem = "go"
	EcosystemPyPI Ecosystem = "pypi"
)

// RiskLevel は更新の予測される影響範囲（blast radius）である。
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// Priority は更新のトリアージ上の緊急度である。
type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityMedium   Priority = "medium"
	PriorityLow      Priority = "low"
)

// CIFailureCategory は CI 実行が失敗した理由を分類する。
type CIFailureCategory string

const (
	FailDependencyConflict CIFailureCategory = "Dependency Conflict"
	FailAPIBreaking        CIFailureCategory = "API Breaking Change"
	FailTestUpdate         CIFailureCategory = "Test Update Required"
	FailEnvironment        CIFailureCategory = "Environment Issue"
	FailUnknown            CIFailureCategory = "Unknown"
)

// CVEInfo はセキュリティ起因の更新に紐づくアドバイザリを記述する。
type CVEInfo struct {
	ID               string `json:"id"`
	Severity         string `json:"severity"` // critical|high|moderate|low
	AffectedRange    string `json:"affected_range"`
	PatchedVersion   string `json:"patched_version"`
	ExploitAvailable bool   `json:"exploit_available,omitempty"`
	Summary          string `json:"summary,omitempty"`
}

// ImpactAnalysis は影響分析ステージの出力である。
type ImpactAnalysis struct {
	UsageSites        []string  `json:"usage_sites"`
	HasBreakingChange bool      `json:"has_breaking_change"`
	AffectedFiles     []string  `json:"affected_files"`
	RiskLevel         RiskLevel `json:"risk_level"`
	Summary           string    `json:"summary"`
	Confidence        float64   `json:"confidence"` // 0..1
}

// CIResult はプルリクエストに対する最新の CI 結果である。
type CIResult struct {
	Status          string            `json:"status"` // running|passed|failed
	FailureCategory CIFailureCategory `json:"failure_category,omitempty"`
	Fixable         bool              `json:"fixable,omitempty"`
	Attempts        int               `json:"attempts"`
	LogSummary      string            `json:"log_summary,omitempty"`
}

// TransitionLog はレコードのイミュータブルな監査履歴における 1 エントリである。
type TransitionLog struct {
	From   State  `json:"from"`
	To     State  `json:"to"`
	Reason string `json:"reason"`
	At     string `json:"at"`
}

// DependencyUpdate は中心となる集約（aggregate）である。UpdateKey は個々の更新を
// 一意にし、冪等性と重複防止の基礎となる。
type DependencyUpdate struct {
	UpdateKey          string          `json:"update_key"`
	Repository         string          `json:"repository"`
	RepoPath           string          `json:"repo_path,omitempty"`
	BaseBranch         string          `json:"base_branch,omitempty"`
	Ecosystem          Ecosystem       `json:"ecosystem"`
	PackageName        string          `json:"package_name"`
	CurrentVersion     string          `json:"current_version"`
	TargetVersion      string          `json:"target_version"`
	UpdateType         UpdateType      `json:"update_type"`
	IsDevDependency    bool            `json:"is_dev_dependency"`
	Priority           Priority        `json:"priority"`
	RiskLevel          RiskLevel       `json:"risk_level"`
	Status             State           `json:"status"`
	AgentVersion       string          `json:"agent_version"`
	CVE                *CVEInfo        `json:"cve,omitempty"`
	Impact             *ImpactAnalysis `json:"impact,omitempty"`
	CI                 *CIResult       `json:"ci,omitempty"`
	PullRequestURL     string          `json:"pull_request_url,omitempty"`
	PullRequestNumber  int             `json:"pull_request_number,omitempty"`
	Branch             string          `json:"branch,omitempty"`
	ReviewCommentCount *int            `json:"review_comment_count,omitempty"`
	History            []TransitionLog `json:"history"`
	CreatedAt          string          `json:"created_at"`
	UpdatedAt          string          `json:"updated_at"`
}

// IsSecurityDriven はこの更新がアドバイザリを伴うかどうかを返す。
func (u *DependencyUpdate) IsSecurityDriven() bool {
	return u.CVE != nil
}

// UpdateKey は正規の一意キーを生成する: repository + package + target。
func UpdateKey(repository, packageName, targetVersion string) string {
	return Slug(repository) + "::" + Slug(packageName) + "::" + Slug(targetVersion)
}

// InvalidTransitionError はグラフ上で許可されていない遷移が行われたときに返される。
type InvalidTransitionError struct{ From, To State }

func (e InvalidTransitionError) Error() string {
	return fmt.Sprintf("invalid transition: %s -> %s", e.From, e.To)
}

func (u DependencyUpdate) clone() DependencyUpdate {
	cp := u
	cp.History = append([]TransitionLog(nil), u.History...)
	return cp
}

// Transition はレコードをライフサイクルに沿って 1 ステップ進め、その遷移を監査履歴に
// 追加した新しいレコード（イミュータブルな更新）を返す。同一状態への呼び出しは冪等な
// no-op であり、再配信されたイベントでも安全である。
func (u DependencyUpdate) Transition(to State, reason string) (DependencyUpdate, error) {
	if u.Status == to {
		next := u.clone()
		next.UpdatedAt = NowString()
		return next, nil
	}
	if !CanTransition(u.Status, to) {
		return u, InvalidTransitionError{From: u.Status, To: to}
	}
	at := NowString()
	next := u.clone()
	next.History = append(next.History, TransitionLog{From: u.Status, To: to, Reason: reason, At: at})
	next.Status = to
	next.UpdatedAt = at
	return next, nil
}

// ToError はどの状態からでもレコードをエラー状態へ移す。エラーはリトライ安全である。
func (u DependencyUpdate) ToError(reason string) DependencyUpdate {
	at := NowString()
	next := u.clone()
	next.History = append(next.History, TransitionLog{From: u.Status, To: StateError, Reason: reason, At: at})
	next.Status = StateError
	next.UpdatedAt = at
	return next
}
