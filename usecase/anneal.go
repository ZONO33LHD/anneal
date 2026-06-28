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

// AnnealUsecase は自己改善（Annealing）ループを実行する。
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

// NewAnnealUsecase は Annealing ループを組み立てる。
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

// MaybeAnneal は直近 `window` 件の確定スコアの移動平均が threshold を
// 下回ったときにループを起動する。
func (a *annealUsecase) MaybeAnneal(ctx context.Context) error {
	avg, sampled, err := a.rollingAverage()
	if err != nil {
		return err
	}
	if sampled == 0 {
		return nil
	}
	a.log.Info(ctx, "score check", "average", avg, "sampled", sampled, "threshold", a.threshold)
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

// run は失敗ケースを読み取り、LLM を介して改善候補を生成する。
// A/B による採用はこのビルドの対象外。
func (a *annealUsecase) run(ctx context.Context, average float64) error {
	failures, err := a.improvements.ListFailures()
	if err != nil {
		return err
	}
	if len(failures) == 0 {
		a.log.Info(ctx, "annealing: no failure cases to learn from")
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
		Prompt:      "[hypothesis] Given these low-scoring dependency-update cases, propose the single most likely systemic weakness in the agent's judgement:\n" + digest.String(),
		Temperature: 0.4,
	})
	proposed, _ := a.llm.Generate(ctx, gateway.LLMRequest{
		Prompt:      "[prompt-improvement] Based on this hypothesis, propose one concrete change to the agent's risk/decision prompt to fix it:\nHypothesis: " + hypothesis,
		Temperature: 0.4,
	})

	// 候補は「いま有効な版」を基準に作る。採用済み/試用中の改善があればその版から
	// 派生させ、版番号が連続するようにする。
	existing, _ := a.improvements.ListImprovements()
	activeVersion := model.ActiveAgentVersion(existing)
	imp := model.AgentImprovement{
		ImprovementID:    model.NewID("imp"),
		Seq:              model.NextSeq(),
		Trigger:          fmt.Sprintf("rolling avg %.1f below threshold", average),
		Target:           "prompt",
		PreviousVersion:  activeVersion,
		CandidateVersion: nextVersion(activeVersion),
		Hypothesis:       hypothesis,
		ProposedChange:   proposed,
		Status:           model.ImprovementCandidate,
		CreatedAt:        model.NowString(),
	}
	if err := a.improvements.PutImprovement(imp); err != nil {
		return err
	}
	notifyOrLog(ctx, a.notifier, a.log, gateway.NotifyMessage{
		Level: gateway.NotifySuccess,
		Title: "🔥 Annealing Loop produced an improvement candidate",
		Body: fmt.Sprintf("Trigger: %s\nHypothesis: %s\nProposed (%s): %s",
			imp.Trigger, hypothesis, imp.CandidateVersion, proposed),
	})
	a.log.Step(ctx, "annealing: candidate generated", "improvement_id", imp.ImprovementID)
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
