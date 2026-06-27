package model

import "slices"

// State は依存関係更新のライフサイクルにおける 1 段階である。このライフサイクルは
// システムの背骨であり、すべてのトリガはレコードを読み取り、このグラフに沿って 1 ステップ
// 進め、書き戻す。
type State string

const (
	StateDetected             State = "detected"
	StateAnalyzing            State = "analyzing"
	StateAwaitingApproval     State = "awaiting_approval"
	StatePRCreating           State = "pr_creating"
	StatePRCreated            State = "pr_created"
	StateCIRunning            State = "ci_running"
	StateCIPassed             State = "ci_passed"
	StateCIFailed             State = "ci_failed"
	StateFixing               State = "fixing"
	StateAwaitingReview       State = "awaiting_review"
	StateChangesRequested     State = "changes_requested"
	StateMerged               State = "merged"
	StateMonitoringRegression State = "monitoring_regression"
	StateDone                 State = "done"
	StateRegressed            State = "regressed"
	StateSuperseded           State = "superseded"
	StateOnHold               State = "on_hold"
	StateClosed               State = "closed"
	StateError                State = "error"
)

// Transitions は許可された遷移のグラフである。ここに列挙されていない遷移は拒否され、
// ライフサイクルが不正な状態へ逸脱するのを防ぐ。
var Transitions = map[State][]State{
	StateDetected:             {StateAnalyzing, StateSuperseded, StateError},
	StateAnalyzing:            {StateAwaitingApproval, StatePRCreating, StateError},
	StateAwaitingApproval:     {StatePRCreating, StateClosed},
	StatePRCreating:           {StatePRCreated, StateError},
	StatePRCreated:            {StateCIRunning, StateSuperseded},
	StateCIRunning:            {StateCIPassed, StateCIFailed},
	StateCIPassed:             {StateAwaitingReview},
	StateCIFailed:             {StateFixing, StateAwaitingReview},
	StateFixing:               {StateCIRunning, StateOnHold},
	StateAwaitingReview:       {StateChangesRequested, StateMerged, StateClosed},
	StateChangesRequested:     {StateFixing},
	StateMerged:               {StateMonitoringRegression},
	StateMonitoringRegression: {StateDone, StateRegressed},
	StateOnHold:               {StateAwaitingReview},
	StateRegressed:            {StateAwaitingReview},
	StateDone:                 {},
	StateClosed:               {},
	StateSuperseded:           {},
	// error は回復可能: リトライによってパイプラインに再投入される。
	StateError: {StateDetected, StateAnalyzing, StatePRCreating, StateClosed},
}

var terminalStates = map[State]bool{StateDone: true, StateClosed: true, StateSuperseded: true}

// IsTerminal は状態がこれ以上の処理を必要としないかどうかを返す。
func IsTerminal(s State) bool {
	return terminalStates[s]
}

// IsActive はレコードがまだ処理中かどうかを返す。重複防止はこれを使う: アクティブな
// レコードを持つ update_key に対しては新しいレコードを作成しない。error はリトライ
// 可能なのでアクティブとみなす。
func IsActive(s State) bool {
	return !IsTerminal(s)
}

// CanTransition は from->to が許可された遷移かどうかを返す。
func CanTransition(from, to State) bool {
	return slices.Contains(Transitions[from], to)
}
