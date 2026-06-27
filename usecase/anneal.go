package usecase

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/repository"
)

// AnnealUsecase runs the self-improvement (Annealing) loop.
type AnnealUsecase interface {
	MaybeAnneal(ctx context.Context) error
}

type annealUsecase struct {
	evals        repository.EvaluationRepository
	improvements repository.ImprovementRepository
	llm          gateway.LLM
	notifier     gateway.Notifier
	log          gateway.Logger
	window       int
	threshold    float64
}

// NewAnnealUsecase wires the Annealing loop.
func NewAnnealUsecase(
	evals repository.EvaluationRepository,
	improvements repository.ImprovementRepository,
	llm gateway.LLM,
	notifier gateway.Notifier,
	log gateway.Logger,
	window int,
	threshold float64,
) AnnealUsecase {
	return &annealUsecase{evals, improvements, llm, notifier, log, window, threshold}
}

// MaybeAnneal fires the loop when the rolling average of the last `window`
// finalized scores drops below threshold.
func (a *annealUsecase) MaybeAnneal(ctx context.Context) error {
	avg, sampled, err := a.rollingAverage()
	if err != nil {
		return err
	}
	if sampled == 0 {
		return nil
	}
	a.log.Info(fmt.Sprintf("score check average=%.1f sampled=%d threshold=%.0f", avg, sampled, a.threshold))
	if avg >= a.threshold {
		return nil
	}
	return a.run(ctx, avg)
}

func (a *annealUsecase) rollingAverage() (avg float64, sampled int, err error) {
	evals, err := a.evals.List()
	if err != nil {
		return 0, 0, err
	}
	var finals []model.AgentEvaluation
	for _, e := range evals {
		if e.ScoreStatus == model.ScoreFinal {
			finals = append(finals, e)
		}
	}
	sort.Slice(finals, func(i, j int) bool { return finals[i].Seq < finals[j].Seq })
	if len(finals) > a.window {
		finals = finals[len(finals)-a.window:]
	}
	if len(finals) == 0 {
		return 0, 0, nil
	}
	var sum float64
	for _, e := range finals {
		sum += e.TotalScore
	}
	avg = sum / float64(len(finals))
	return float64(int(avg*10)) / 10, len(finals), nil
}

// run reads failure cases and generates an improvement candidate via the LLM.
// Adoption via A/B is out of scope for this build.
func (a *annealUsecase) run(ctx context.Context, average float64) error {
	failures, err := a.improvements.ListFailures()
	if err != nil {
		return err
	}
	if len(failures) == 0 {
		a.log.Info("annealing: no failure cases to learn from")
		return nil
	}

	recent := failures
	if len(recent) > 10 {
		recent = recent[len(recent)-10:]
	}
	var digest strings.Builder
	for _, f := range recent {
		fmt.Fprintf(&digest, "- %s score=%.1f\n", f.UpdateKey, f.TotalScore)
	}

	hypothesis, _ := a.llm.Generate(ctx, gateway.LLMRequest{
		Prompt: "[hypothesis] Given these low-scoring dependency-update cases, propose the single most likely systemic weakness in the agent's judgement:\n" + digest.String(),
		Temperature: 0.4,
	})
	proposed, _ := a.llm.Generate(ctx, gateway.LLMRequest{
		Prompt:      "[prompt-improvement] Based on this hypothesis, propose one concrete change to the agent's risk/decision prompt to fix it:\nHypothesis: " + hypothesis,
		Temperature: 0.4,
	})

	imp := model.AgentImprovement{
		ImprovementID:    model.NewID("imp"),
		Trigger:          fmt.Sprintf("rolling avg %.1f below threshold", average),
		Target:           "prompt",
		PreviousVersion:  model.CurrentAgentVersion,
		CandidateVersion: nextVersion(model.CurrentAgentVersion),
		Hypothesis:       hypothesis,
		ProposedChange:   proposed,
		Status:           model.ImprovementCandidate,
		CreatedAt:        model.NowString(),
	}
	if err := a.improvements.PutImprovement(imp); err != nil {
		return err
	}
	_ = a.notifier.Notify(ctx, gateway.NotifyMessage{
		Level: gateway.NotifySuccess,
		Title: "🔥 Annealing Loop produced an improvement candidate",
		Body: fmt.Sprintf("Trigger: %s\nHypothesis: %s\nProposed (%s): %s",
			imp.Trigger, hypothesis, imp.CandidateVersion, proposed),
	})
	a.log.Step("annealing: candidate generated " + imp.ImprovementID)
	return nil
}

var versionTail = regexp.MustCompile(`^(.*?)(\d+)$`)

func nextVersion(current string) string {
	m := versionTail.FindStringSubmatch(current)
	if m == nil {
		return current + "_v2"
	}
	n, _ := strconv.Atoi(m[2])
	return m[1] + strconv.Itoa(n+1)
}
