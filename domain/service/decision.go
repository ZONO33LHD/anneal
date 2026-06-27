package service

import (
	"fmt"
	"strings"

	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/policy"
)

// Decision is the outcome of the approval gate.
type Decision struct {
	Decision string // "auto" or "human"
	Reasons  []string
}

func isSensitive(name string) bool {
	lower := strings.ToLower(name)
	for _, k := range policy.SensitiveKeywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

// Decide returns whether a PR can proceed automatically or needs human approval.
// Low-quality model output is caught here and routed to a human (the safety net
// for running on the cheapest model tier).
func Decide(rec model.DependencyUpdate) Decision {
	if isSensitive(rec.PackageName) {
		return Decision{"human", []string{"sensitive package (auth/payment/security)"}}
	}

	if rec.CVE != nil {
		if rec.Impact != nil && rec.Impact.HasBreakingChange {
			return Decision{"human", []string{fmt.Sprintf("security fix for %s but breaking change", rec.CVE.ID)}}
		}
		return Decision{"auto", []string{fmt.Sprintf("security fix for %s, no breaking change", rec.CVE.ID)}}
	}

	if rec.UpdateType == model.Major {
		return Decision{"human", []string{"major version bump"}}
	}

	if rec.UpdateType == model.Patch || rec.IsDevDependency {
		return finalGate(rec, []string{"patch or dev-only dependency"})
	}

	if rec.UpdateType == model.Minor {
		if rec.Impact != nil && rec.Impact.RiskLevel == model.RiskHigh {
			return Decision{"human", []string{"minor bump with large/high-risk impact"}}
		}
		return finalGate(rec, []string{"minor bump with contained impact"})
	}

	return Decision{"human", []string{"default to human for safety"}}
}

// finalGate routes low AI confidence or repeated CI failures to a human.
func finalGate(rec model.DependencyUpdate, reasons []string) Decision {
	if rec.Impact != nil && rec.Impact.Confidence < 0.5 {
		return Decision{"human", append(reasons, fmt.Sprintf("low AI confidence (%.2f)", rec.Impact.Confidence))}
	}
	if rec.CI != nil && rec.CI.Attempts >= 2 {
		return Decision{"human", append(reasons, "CI failed multiple times")}
	}
	return Decision{"auto", reasons}
}
