package service

import "github.com/ZONO33LHD/anneal/domain/model"

// DeriveRisk は更新の種類と利用の広がりをリスクレベルへマッピングする。リスクの
// 真の拠り所は（LLM ではなく）ルールベースの判断である。
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

// DeriveConfidence は自動解析がどれだけ信頼できるかを見積もる。小さな patch では
// 高く、major や広く使われている minor では低くなる。
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
