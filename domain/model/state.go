package model

import "slices"

// State is a stage in the dependency-update lifecycle. The lifecycle is the
// backbone of the system: every trigger reads a record, moves it one step along
// this graph, and writes it back.
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

// Transitions is the allowed-transition graph. A move not listed here is
// rejected, which keeps the lifecycle from drifting into invalid states.
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
	// error is recoverable: a retry re-enters the pipeline.
	StateError: {StateDetected, StateAnalyzing, StatePRCreating, StateClosed},
}

var terminalStates = map[State]bool{StateDone: true, StateClosed: true, StateSuperseded: true}

// IsTerminal reports whether a state needs no further processing.
func IsTerminal(s State) bool {
	return terminalStates[s]
}

// IsActive reports whether a record is still in flight. Duplicate prevention uses
// this: a new record is not created for an update_key with an active record.
// error counts as active because it is retryable.
func IsActive(s State) bool {
	return !IsTerminal(s)
}

// CanTransition reports whether from->to is an allowed move.
func CanTransition(from, to State) bool {
	return slices.Contains(Transitions[from], to)
}
