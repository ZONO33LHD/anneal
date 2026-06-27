package model

import "fmt"

// Ecosystem identifies a package ecosystem.
type Ecosystem string

const (
	EcosystemNPM Ecosystem = "npm"
	EcosystemGo  Ecosystem = "go"
)

// RiskLevel is the predicted blast radius of an update.
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// Priority is the triage urgency of an update.
type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityMedium   Priority = "medium"
	PriorityLow      Priority = "low"
)

// CIFailureCategory classifies why a CI run failed.
type CIFailureCategory string

const (
	FailDependencyConflict CIFailureCategory = "Dependency Conflict"
	FailAPIBreaking        CIFailureCategory = "API Breaking Change"
	FailTestUpdate         CIFailureCategory = "Test Update Required"
	FailEnvironment        CIFailureCategory = "Environment Issue"
	FailUnknown            CIFailureCategory = "Unknown"
)

// CVEInfo describes an advisory attached to a security-driven update.
type CVEInfo struct {
	ID               string `json:"id"`
	Severity         string `json:"severity"` // critical|high|moderate|low
	AffectedRange    string `json:"affected_range"`
	PatchedVersion   string `json:"patched_version"`
	ExploitAvailable bool   `json:"exploit_available,omitempty"`
	Summary          string `json:"summary,omitempty"`
}

// ImpactAnalysis is the output of the impact-analysis stage.
type ImpactAnalysis struct {
	UsageSites        []string  `json:"usage_sites"`
	HasBreakingChange bool      `json:"has_breaking_change"`
	AffectedFiles     []string  `json:"affected_files"`
	RiskLevel         RiskLevel `json:"risk_level"`
	Summary           string    `json:"summary"`
	Confidence        float64   `json:"confidence"` // 0..1
}

// CIResult is the latest CI outcome for the pull request.
type CIResult struct {
	Status          string            `json:"status"` // running|passed|failed
	FailureCategory CIFailureCategory `json:"failure_category,omitempty"`
	Fixable         bool              `json:"fixable,omitempty"`
	Attempts        int               `json:"attempts"`
	LogSummary      string            `json:"log_summary,omitempty"`
}

// TransitionLog is one entry in the immutable audit history of a record.
type TransitionLog struct {
	From   State  `json:"from"`
	To     State  `json:"to"`
	Reason string `json:"reason"`
	At     string `json:"at"`
}

// DependencyUpdate is the central aggregate. UpdateKey makes a given update
// unique and is the basis for idempotency and duplicate prevention.
type DependencyUpdate struct {
	UpdateKey          string          `json:"update_key"`
	Repository         string          `json:"repository"`
	RepoPath           string          `json:"repo_path,omitempty"`
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

// IsSecurityDriven reports whether this update carries an advisory.
func (u *DependencyUpdate) IsSecurityDriven() bool {
	return u.CVE != nil
}

// UpdateKey builds the canonical unique key: repository + package + target.
func UpdateKey(repository, packageName, targetVersion string) string {
	return Slug(repository) + "::" + Slug(packageName) + "::" + Slug(targetVersion)
}

// InvalidTransitionError is returned when a move is not allowed by the graph.
type InvalidTransitionError struct{ From, To State }

func (e InvalidTransitionError) Error() string {
	return fmt.Sprintf("invalid transition: %s -> %s", e.From, e.To)
}

func (u DependencyUpdate) clone() DependencyUpdate {
	cp := u
	cp.History = append([]TransitionLog(nil), u.History...)
	return cp
}

// Transition moves the record one step along the lifecycle, returning a NEW
// record (immutable update) with the move appended to its audit history. A
// same-state call is an idempotent no-op so redelivered events are safe.
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

// ToError moves the record to the error state from anywhere; error is retry-safe.
func (u DependencyUpdate) ToError(reason string) DependencyUpdate {
	at := NowString()
	next := u.clone()
	next.History = append(next.History, TransitionLog{From: u.Status, To: StateError, Reason: reason, At: at})
	next.Status = StateError
	next.UpdatedAt = at
	return next
}
