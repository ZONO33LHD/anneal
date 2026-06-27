// Package service holds pure domain logic with no I/O: classification, the
// approval decision, risk derivation, CI-failure rules, and scoring.
package service

import "github.com/ZONO33LHD/anneal/domain/model"

// ClassifyPriority assigns triage urgency:
//
//	critical: exploitable CVE / exploit published
//	high:     any other advisory
//	medium:   ordinary patch / minor
//	low:      dev-only, or major with breaking-change concern
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
		return model.PriorityLow // defer; breaking-change risk
	}
	return model.PriorityMedium
}
