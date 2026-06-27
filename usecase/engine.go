package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/repository"
	"github.com/ZONO33LHD/anneal/domain/service"
)

// EngineUsecase advances dependency-update records through the lifecycle. The
// single discipline is: read a record, move it one step, write it back.
type EngineUsecase interface {
	Dispatch(ctx context.Context, rec model.DependencyUpdate) (bool, model.DependencyUpdate, error)
	Tick(ctx context.Context) (int, error)
	Drive(ctx context.Context, maxRounds int) error
	Reconcile(ctx context.Context) (int, error)
}

type engine struct {
	updates      repository.UpdateRepository
	evals        repository.EvaluationRepository
	improvements repository.ImprovementRepository
	llm          gateway.LLM
	git          gateway.Git
	notifier     gateway.Notifier
	ecosystems   gateway.EcosystemProvider
	scanner      gateway.SourceScanner
	log          gateway.Logger

	lowScoreThreshold float64
	// simulate auto-advances human gates and CI (mock mode) so the full lifecycle
	// runs locally; in real mode those boundaries wait for webhooks/reconcile.
	simulate bool
}

// NewEngineUsecase wires the engine.
func NewEngineUsecase(
	updates repository.UpdateRepository,
	evals repository.EvaluationRepository,
	improvements repository.ImprovementRepository,
	llm gateway.LLM,
	git gateway.Git,
	notifier gateway.Notifier,
	ecosystems gateway.EcosystemProvider,
	scanner gateway.SourceScanner,
	log gateway.Logger,
	lowScoreThreshold float64,
	simulate bool,
) EngineUsecase {
	return &engine{updates, evals, improvements, llm, git, notifier, ecosystems, scanner, log, lowScoreThreshold, simulate}
}

var scoreStates = map[model.State]bool{
	model.StateCIPassed: true, model.StateCIFailed: true, model.StateAwaitingReview: true,
	model.StateChangesRequested: true, model.StateMerged: true,
	model.StateMonitoringRegression: true, model.StateDone: true,
	model.StateRegressed: true, model.StateClosed: true,
}

// Dispatch advances a single record exactly one meaningful step, persists it, and
// recomputes its evaluation. On error the record is moved to the error state.
func (e *engine) Dispatch(ctx context.Context, rec model.DependencyUpdate) (bool, model.DependencyUpdate, error) {
	next, moved, err := e.advance(ctx, rec)
	if err != nil {
		errored := rec.ToError(err.Error())
		e.log.Warn("dispatch failed; moving to error: " + err.Error())
		_ = e.updates.Put(errored)
		return true, errored, nil
	}
	if !moved {
		return false, rec, nil
	}
	if err := e.updates.Put(next); err != nil {
		return false, rec, err
	}
	e.maybeScore(next)
	return true, next, nil
}

// Tick advances every active record one step.
func (e *engine) Tick(ctx context.Context) (int, error) {
	active, err := e.updates.ListActive()
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, rec := range active {
		moved, _, err := e.Dispatch(ctx, rec)
		if err != nil {
			return changed, err
		}
		if moved {
			changed++
		}
	}
	return changed, nil
}

// Drive runs the lifecycle to a standstill (used by demo).
func (e *engine) Drive(ctx context.Context, maxRounds int) error {
	for range maxRounds {
		changed, err := e.Tick(ctx)
		if err != nil {
			return err
		}
		if changed == 0 {
			return nil
		}
	}
	e.log.Warn("drive: hit max rounds; some records may still be active")
	return nil
}

// Reconcile is the catch-up loop for records left behind by missed events. In
// local mode the store is the source of truth, so it is a single tick.
func (e *engine) Reconcile(ctx context.Context) (int, error) {
	changed, err := e.Tick(ctx)
	if err != nil {
		return changed, err
	}
	e.log.Info(fmt.Sprintf("reconcile complete advanced=%d", changed))
	return changed, nil
}

func (e *engine) advance(ctx context.Context, rec model.DependencyUpdate) (model.DependencyUpdate, bool, error) {
	switch rec.Status {
	case model.StateDetected:
		return wrap(e.analyzeStep(ctx, rec))

	case model.StateAnalyzing:
		d := service.Decide(rec)
		if d.Decision == "human" {
			_ = e.notifier.Notify(ctx, gateway.NotifyMessage{
				Level: gateway.NotifyApproval,
				Title: "Human approval required: " + rec.PackageName,
				Body:  strings.Join(d.Reasons, "; "),
			})
			return wrap(rec.Transition(model.StateAwaitingApproval, "human: "+strings.Join(d.Reasons, "; ")))
		}
		return wrap(rec.Transition(model.StatePRCreating, "auto: "+strings.Join(d.Reasons, "; ")))

	case model.StateAwaitingApproval:
		if !e.simulate {
			return rec, false, nil
		}
		return wrap(rec.Transition(model.StatePRCreating, "simulated approval"))

	case model.StatePRCreating:
		return wrap(e.createPRStep(ctx, rec))

	case model.StatePRCreated:
		next, err := rec.Transition(model.StateCIRunning, "CI started")
		if err == nil {
			next.CI = &model.CIResult{Status: "running", Attempts: 0}
		}
		return next, err == nil, err

	case model.StateCIRunning:
		return wrap(e.ciStep(ctx, rec))

	case model.StateCIPassed:
		return wrap(rec.Transition(model.StateAwaitingReview, "CI passed → review"))

	case model.StateCIFailed:
		return wrap(e.fixStep(ctx, rec))

	case model.StateFixing:
		return wrap(rec.Transition(model.StateCIRunning, "re-run CI after fix"))

	case model.StateChangesRequested:
		if !e.simulate {
			return rec, false, nil
		}
		return wrap(rec.Transition(model.StateFixing, "addressing review feedback"))

	case model.StateAwaitingReview:
		if !e.simulate {
			return rec, false, nil
		}
		if rec.CI != nil && rec.CI.Status == "failed" {
			next, err := rec.Transition(model.StateClosed, "simulated human rejection (CI red)")
			if err == nil {
				next.ReviewCommentCount = simulatedComments(rec)
			}
			return next, err == nil, err
		}
		next, err := rec.Transition(model.StateMerged, "simulated human merge")
		if err == nil {
			next.ReviewCommentCount = simulatedComments(rec)
		}
		return next, err == nil, err

	case model.StateMerged:
		_ = e.notifier.Notify(ctx, gateway.NotifyMessage{
			Level: gateway.NotifySuccess,
			Title: "Merged: " + rec.PackageName + " → " + rec.TargetVersion,
			Body:  "Entering regression monitoring window.",
			URL:   rec.PullRequestURL,
		})
		return wrap(rec.Transition(model.StateMonitoringRegression, "merged"))

	case model.StateMonitoringRegression:
		if !e.simulate {
			return rec, false, nil
		}
		return wrap(rec.Transition(model.StateDone, "no regression detected"))

	default:
		return rec, false, nil // terminal / on_hold / error
	}
}

// wrap adapts a (record, error) step into the advance signature.
func wrap(next model.DependencyUpdate, err error) (model.DependencyUpdate, bool, error) {
	if err != nil {
		return next, false, err
	}
	return next, true, nil
}

func simulatedComments(rec model.DependencyUpdate) *int {
	n := 0
	if rec.Impact != nil {
		switch rec.Impact.RiskLevel {
		case model.RiskHigh:
			n = 3
		case model.RiskMedium:
			n = 1
		}
	}
	return &n
}

func (e *engine) maybeScore(rec model.DependencyUpdate) {
	if !scoreStates[rec.Status] {
		return
	}
	prev, _ := e.evals.Get(rec.UpdateKey)
	eval := service.BuildEvaluation(rec, prev)
	_ = e.evals.Put(eval)
	if eval.ScoreStatus == model.ScoreFinal {
		e.recordIfLowScore(rec, eval)
	}
}

// recordIfLowScore persists a finalized below-threshold evaluation as a failure
// case (idempotent: at most one per update_key) so the Annealing Loop can learn.
func (e *engine) recordIfLowScore(rec model.DependencyUpdate, eval model.AgentEvaluation) {
	if eval.TotalScore >= e.lowScoreThreshold {
		return
	}
	existing, _ := e.improvements.ListFailures()
	for _, f := range existing {
		if f.UpdateKey == rec.UpdateKey {
			return
		}
	}
	snapshot := map[string]any{"update_type": rec.UpdateType, "risk_level": rec.RiskLevel, "ci": rec.CI}
	if rec.Impact != nil {
		snapshot["impact"] = map[string]any{"riskLevel": rec.Impact.RiskLevel, "sites": len(rec.Impact.UsageSites)}
	}
	_ = e.improvements.PutFailure(model.FailureCase{
		UpdateKey:    rec.UpdateKey,
		AgentVersion: rec.AgentVersion,
		TotalScore:   eval.TotalScore,
		Reason:       "score below threshold",
		Snapshot:     snapshot,
		CreatedAt:    model.NowString(),
	})
	e.log.Warn("recorded failure case " + rec.UpdateKey)
}
