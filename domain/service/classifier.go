// Package service は I/O を持たない純粋なドメインロジックを保持する。すなわち分類、
// 承認判断、リスク導出、CI 失敗ルール、スコアリングである。
package service

import "github.com/ZONO33LHD/anneal/domain/model"

// ClassifyPriority はトリアージの緊急度を割り当てる:
//
//	critical: 悪用可能な CVE / 公開済みのエクスプロイト
//	high:     その他のアドバイザリ
//	medium:   通常の patch / minor
//	low:      dev 専用、または破壊的変更の懸念がある major
func ClassifyPriority(updateType model.UpdateType, isDev bool, cve *model.CVEInfo) model.Priority {
	if cve != nil {
		if cve.Severity == "critical" || cve.ExploitAvailable {
			return model.PriorityCritical
		}
		return model.PriorityHigh
	}
	if isDev {
		return model.PriorityLow
	}
	if updateType == model.Major {
		return model.PriorityLow // 先送り。破壊的変更のリスクがある
	}
	return model.PriorityMedium
}
