package usecase

import (
	"context"
	"fmt"
	"sort"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/repository"
	"github.com/ZONO33LHD/anneal/domain/service"
)

// AdoptionUsecase は改善候補（candidate）の A/B 採用を進める。1 回の Evaluate で
// 「candidate を canary に昇格」または「canary を確定スコアで評価して採用/巻き戻し」の
// いずれか 1 ステップを進める。状態機械と同じく「1 呼び出し 1 ステップ」の規律に従う。
type AdoptionUsecase interface {
	Evaluate(ctx context.Context) error
}

type adoptionUsecase struct {
	evals        repository.EvaluationRepository
	improvements repository.ImprovementRepository
	notifier     gateway.Notifier
	log          gateway.Logger
	minSample    int
	margin       float64
}

// NewAdoptionUsecase は A/B 採用ユースケースを組み立てる。
func NewAdoptionUsecase(
	evals repository.EvaluationRepository,
	improvements repository.ImprovementRepository,
	notifier gateway.Notifier,
	log gateway.Logger,
	minSample int,
	margin float64,
) AdoptionUsecase {
	return &adoptionUsecase{
		evals:        evals,
		improvements: improvements,
		notifier:     notifier,
		log:          log,
		minSample:    minSample,
		margin:       margin,
	}
}

// Evaluate は採用ループを 1 ステップ進める。
func (a *adoptionUsecase) Evaluate(ctx context.Context) error {
	improvements, err := a.improvements.ListImprovements()
	if err != nil {
		return err
	}
	evals, err := a.evals.List()
	if err != nil {
		return err
	}

	// canary が動いていればその評価を優先する（候補の昇格より採用判断を先に進める）。
	if canary, ok := latestByStatus(improvements, model.ImprovementCanary); ok {
		return a.judgeCanary(ctx, canary, evals)
	}
	// canary が無ければ、最新の candidate を canary に昇格する。
	if candidate, ok := latestByStatus(improvements, model.ImprovementCandidate); ok {
		return a.promoteToCanary(ctx, candidate, improvements, evals)
	}
	return nil
}

// promoteToCanary は candidate を canary にし、昇格時点の現行版平均をベースラインとして
// 記録する。これ以降の新規スキャンは候補版でタグ付けされる（ActiveAgentVersion 経由）。
func (a *adoptionUsecase) promoteToCanary(
	ctx context.Context,
	candidate model.AgentImprovement,
	improvements []model.AgentImprovement,
	evals []model.AgentEvaluation,
) error {
	baselineVersion := model.ActiveAgentVersion(improvements)
	baseline := service.ScoreByVersion(evals, baselineVersion)

	next := candidate
	next.Status = model.ImprovementCanary
	next.BaselineScore = baseline.Average
	next.CanaryStartedAt = model.NowString()
	if err := a.improvements.PutImprovement(next); err != nil {
		return err
	}
	a.log.Step(ctx, "adoption: candidate promoted to canary",
		"improvement_id", next.ImprovementID,
		"candidate_version", next.CandidateVersion,
		"baseline_version", baselineVersion,
		"baseline_score", baseline.Average,
	)
	return nil
}

// judgeCanary は canary 版の確定スコアをベースラインと比較し、採用/巻き戻し/様子見を決める。
func (a *adoptionUsecase) judgeCanary(
	ctx context.Context,
	canary model.AgentImprovement,
	evals []model.AgentEvaluation,
) error {
	stats := service.ScoreByVersion(evals, canary.CandidateVersion)
	verdict := service.EvaluateCanary(stats, canary.BaselineScore, a.minSample, a.margin)
	a.log.Info(ctx, "adoption: canary evaluated",
		"improvement_id", canary.ImprovementID,
		"candidate_version", canary.CandidateVersion,
		"candidate_score", stats.Average,
		"candidate_samples", stats.Count,
		"baseline_score", canary.BaselineScore,
		"verdict", string(verdict),
	)

	switch verdict {
	case service.CanaryAdopt:
		return a.finalize(ctx, canary, stats.Average, model.ImprovementAdopted)
	case service.CanaryRollback:
		return a.finalize(ctx, canary, stats.Average, model.ImprovementRolledBack)
	default:
		return nil
	}
}

// finalize は canary を adopted か rolled_back に確定させ、結果を通知する。
func (a *adoptionUsecase) finalize(
	ctx context.Context,
	canary model.AgentImprovement,
	candidateScore float64,
	status model.ImprovementStatus,
) error {
	next := canary
	next.Status = status
	next.CandidateScore = candidateScore
	if status == model.ImprovementAdopted {
		next.AdoptedAt = model.NowString()
	} else {
		next.RolledBackAt = model.NowString()
	}
	if err := a.improvements.PutImprovement(next); err != nil {
		return err
	}

	title := "✅ Annealing: 改善版を採用しました"
	level := gateway.NotifySuccess
	if status == model.ImprovementRolledBack {
		title = "↩️ Annealing: 改善版を巻き戻しました"
		level = gateway.NotifyPriority
	}
	notifyOrLog(ctx, a.notifier, a.log, gateway.NotifyMessage{
		Level: level,
		Title: title,
		Body: fmt.Sprintf("version=%s baseline=%.1f candidate=%.1f",
			next.CandidateVersion, next.BaselineScore, candidateScore),
	})
	a.log.Step(ctx, "adoption: canary finalized",
		"improvement_id", next.ImprovementID,
		"status", string(status),
	)
	return nil
}

// latestByStatus は指定 status の改善のうち Seq が最大のものを返す。
func latestByStatus(improvements []model.AgentImprovement, status model.ImprovementStatus) (model.AgentImprovement, bool) {
	matched := make([]model.AgentImprovement, 0, len(improvements))
	for _, imp := range improvements {
		if imp.Status == status {
			matched = append(matched, imp)
		}
	}
	if len(matched) == 0 {
		return model.AgentImprovement{}, false
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].Seq > matched[j].Seq })
	return matched[0], true
}
