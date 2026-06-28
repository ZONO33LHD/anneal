// Package service の adoption は、改善候補（canary）の A/B 採用・巻き戻し判断に
// 使う純粋ロジックを提供する。永続化や外部 I/O は持たない。
package service

import "github.com/ZONO33LHD/anneal/domain/model"

// VersionStats は特定のエージェント版に紐づく確定スコアの集計である。
type VersionStats struct {
	Version string
	Average float64
	Count   int
}

// ScoreByVersion は指定した版の「確定（final）」評価だけを対象に平均スコアを集計する。
// partial は確定前で揺れるため除外する。
func ScoreByVersion(evals []model.AgentEvaluation, version string) VersionStats {
	var sum float64
	var n int
	for _, e := range evals {
		if e.ScoreStatus != model.ScoreFinal {
			continue
		}
		if e.AgentVersion != version {
			continue
		}
		sum += e.TotalScore
		n++
	}
	stats := VersionStats{Version: version, Count: n}
	if n > 0 {
		stats.Average = sum / float64(n)
	}
	return stats
}

// CanaryVerdict は canary 版に対する判断結果である。
type CanaryVerdict string

const (
	// CanaryHold は標本不足や判断保留（baseline 近傍）を表す。
	CanaryHold CanaryVerdict = "hold"
	// CanaryAdopt は baseline 以上で採用すべきことを表す。
	CanaryAdopt CanaryVerdict = "adopt"
	// CanaryRollback は baseline を margin 超で下回り、巻き戻すべきことを表す。
	CanaryRollback CanaryVerdict = "rollback"
)

// EvaluateCanary は canary 版の集計とベースライン平均を比較して判断を返す。
//   - 標本が minSample 未満なら CanaryHold（まだ判断しない）。
//   - canary 平均が baseline 以上なら CanaryAdopt。
//   - canary 平均が baseline-margin を下回るなら CanaryRollback（明確な劣化）。
//   - その中間（baseline 未満だが margin 以内）は CanaryHold（様子見）。
func EvaluateCanary(canary VersionStats, baseline float64, minSample int, margin float64) CanaryVerdict {
	if canary.Count < minSample {
		return CanaryHold
	}
	if canary.Average >= baseline {
		return CanaryAdopt
	}
	if canary.Average < baseline-margin {
		return CanaryRollback
	}
	return CanaryHold
}
