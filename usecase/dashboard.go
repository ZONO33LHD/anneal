package usecase

import (
	"context"
	"sort"

	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/repository"
)

// DashboardUsecase は可視化用の集計ビューを組み立てる。表示は delivery 層に任せ、
// ここでは数値の集計だけを行う。
type DashboardUsecase interface {
	Snapshot(ctx context.Context) (DashboardView, error)
}

// DashboardView はダッシュボード 1 画面ぶんの読み取り専用データである。
type DashboardView struct {
	ActiveVersion  string
	TotalUpdates   int
	StateCounts    []StateCount
	RecentScores   []ScoreRow
	Improvements   []ImprovementRow
	AverageScore   float64
	HasFinalScores bool
}

// StateCount は状態ごとの件数である（件数の多い順に並べる）。
type StateCount struct {
	State string
	Count int
}

// ScoreRow は確定/暫定スコア 1 件の表示用行である。
type ScoreRow struct {
	Package string
	Version string
	Status  string
	Total   float64
}

// ImprovementRow は改善候補 1 件の表示用行である。
type ImprovementRow struct {
	Version   string
	Status    string
	Baseline  float64
	Candidate float64
	Trigger   string
}

type dashboardUsecase struct {
	updates      repository.UpdateRepository
	evals        repository.EvaluationRepository
	improvements repository.ImprovementRepository
}

// NewDashboardUsecase はダッシュボード集計を組み立てる。
func NewDashboardUsecase(
	updates repository.UpdateRepository,
	evals repository.EvaluationRepository,
	improvements repository.ImprovementRepository,
) DashboardUsecase {
	return &dashboardUsecase{
		updates:      updates,
		evals:        evals,
		improvements: improvements,
	}
}

const dashboardRecentLimit = 20

// Snapshot は現在の状態分布・スコア・改善履歴をまとめて返す。
func (d *dashboardUsecase) Snapshot(_ context.Context) (DashboardView, error) {
	updates, err := d.updates.List()
	if err != nil {
		return DashboardView{}, err
	}
	evals, err := d.evals.List()
	if err != nil {
		return DashboardView{}, err
	}
	improvements, err := d.improvements.ListImprovements()
	if err != nil {
		return DashboardView{}, err
	}

	view := DashboardView{
		ActiveVersion: model.ActiveAgentVersion(improvements),
		TotalUpdates:  len(updates),
		StateCounts:   countStates(updates),
		RecentScores:  recentScores(evals),
		Improvements:  improvementRows(improvements),
	}
	view.AverageScore, view.HasFinalScores = averageFinalScore(evals)
	return view, nil
}

// countStates は状態ごとの件数を多い順に並べて返す。
func countStates(updates []model.DependencyUpdate) []StateCount {
	counts := map[string]int{}
	for _, u := range updates {
		counts[string(u.Status)]++
	}
	out := make([]StateCount, 0, len(counts))
	for state, n := range counts {
		out = append(out, StateCount{State: state, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].State < out[j].State
	})
	return out
}

// recentScores は新しい順（Seq 降順）に最大 dashboardRecentLimit 件のスコア行を返す。
func recentScores(evals []model.AgentEvaluation) []ScoreRow {
	sorted := append([]model.AgentEvaluation(nil), evals...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Seq > sorted[j].Seq })
	if len(sorted) > dashboardRecentLimit {
		sorted = sorted[:dashboardRecentLimit]
	}
	rows := make([]ScoreRow, 0, len(sorted))
	for _, e := range sorted {
		rows = append(rows, ScoreRow{
			Package: packageOfKey(e.UpdateKey),
			Version: e.AgentVersion,
			Status:  string(e.ScoreStatus),
			Total:   e.TotalScore,
		})
	}
	return rows
}

// averageFinalScore は確定スコアの平均を返す。確定スコアが無ければ ok=false。
func averageFinalScore(evals []model.AgentEvaluation) (avg float64, ok bool) {
	var sum float64
	var n int
	for _, e := range evals {
		if e.ScoreStatus != model.ScoreFinal {
			continue
		}
		sum += e.TotalScore
		n++
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}

// improvementRows は改善候補を新しい順（Seq 降順）に表示用行へ変換する。
func improvementRows(improvements []model.AgentImprovement) []ImprovementRow {
	sorted := append([]model.AgentImprovement(nil), improvements...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Seq > sorted[j].Seq })
	rows := make([]ImprovementRow, 0, len(sorted))
	for _, imp := range sorted {
		rows = append(rows, ImprovementRow{
			Version:   imp.CandidateVersion,
			Status:    string(imp.Status),
			Baseline:  imp.BaselineScore,
			Candidate: imp.CandidateScore,
			Trigger:   imp.Trigger,
		})
	}
	return rows
}

// packageOfKey は "repo::package::version" から package セグメントを取り出す。
func packageOfKey(key string) string {
	first := -1
	for i := 0; i+1 < len(key); i++ {
		if key[i] == ':' && key[i+1] == ':' {
			if first == -1 {
				first = i + 2
				i++
				continue
			}
			return key[first:i]
		}
	}
	return key
}
