package service

import "github.com/ZONO33LHD/anneal/domain/model"

// DeriveRisk maps update kind + usage breadth to a risk level. The rule-based
// decision (not the LLM) is the source of truth for risk.
func DeriveRisk(updateType model.UpdateType, sites int, breaking bool) model.RiskLevel {
	if breaking || updateType == model.Major {
		return model.RiskHigh
	}
	if updateType == model.Minor && sites > 10 {
		return model.RiskHigh
	}
	if updateType == model.Minor && sites > 0 {
		return model.RiskMedium
	}
	if sites > 20 {
		return model.RiskMedium
	}
	return model.RiskLow
}

// DeriveConfidence estimates how trustworthy the automated analysis is: high for
// small patches, low for majors and widely-used minors.
func DeriveConfidence(updateType model.UpdateType, sites int) float64 {
	switch updateType {
	case model.Patch:
		if sites > 0 {
			return 0.9
		}
		return 0.95
	case model.Minor:
		if sites > 10 {
			return 0.55
		}
		return 0.8
	default:
		return 0.4 // major
	}
}
