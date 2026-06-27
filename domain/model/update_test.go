package model

import (
	"errors"
	"testing"
)

func TestUpdateKey(t *testing.T) {
	a := UpdateKey("acme/repo", "axios", "1.7.0")
	if a != UpdateKey("acme/repo", "axios", "1.7.0") {
		t.Error("update key should be stable")
	}
	if UpdateKey("acme/repo", "axios", "1.7.0") == UpdateKey("acme/repo", "axios", "1.8.0") {
		t.Error("update key should differ by target version")
	}
	if got := UpdateKey("Acme/Repo", "AXIOS", "1.7.0"); got != "acme/repo::axios::1.7.0" {
		t.Errorf("normalization failed: %q", got)
	}
}

func TestStateHelpers(t *testing.T) {
	if !IsTerminal(StateDone) || IsTerminal(StateAnalyzing) {
		t.Error("terminal classification wrong")
	}
	if !CanTransition(StateMerged, StateMonitoringRegression) || CanTransition(StateDone, StateAnalyzing) {
		t.Error("transition table wrong")
	}
}

func TestTransition(t *testing.T) {
	rec := DependencyUpdate{Status: StateDetected, History: []TransitionLog{}}
	next, err := rec.Transition(StateAnalyzing, "start")
	if err != nil {
		t.Fatal(err)
	}
	if next.Status != StateAnalyzing || len(next.History) != 1 {
		t.Errorf("unexpected: %s history=%d", next.Status, len(next.History))
	}
	if rec.Status != StateDetected {
		t.Error("original mutated")
	}

	_, err = rec.Transition(StateMerged, "nope")
	var ite InvalidTransitionError
	if !errors.As(err, &ite) {
		t.Errorf("expected InvalidTransitionError, got %v", err)
	}

	same, _ := rec.Transition(StateDetected, "noop")
	if len(same.History) != 0 {
		t.Error("same-state should not add history")
	}

	if rec.ToError("boom").Status != StateError {
		t.Error("ToError should set error state")
	}
}
