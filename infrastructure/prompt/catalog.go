// Package prompt はプロンプトカタログを提供する。version → プロンプトの対応を
// 一箇所に集約し、採用版に応じた差し替え（T10）の土台とする。
package prompt

import (
	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/repository"
)

// v1 は既定版 prompt_v1 のプロンプト。文言は engine 内のハードコード実装を逐語で
// 再現する（挙動不変・golden test で差分ゼロを担保）。Base は fmt の書式文字列。
var v1 = map[gateway.PromptKey]string{
	gateway.PromptImpact: "[impact] Summarize the impact of bumping %s from %s to %s (%s, %d usage sites, risk=%s).",
}

// catalog はプロンプトカタログの実装。Base は常に prompt_v1 の文言（現時点で版管理
// 対象は判断プロンプト1本のため版で不変）。Appendix は採用版の改善（ProposedChange）
// を後置する追加指示で、improvements から解決する。improvements が nil の場合は
// Appendix を解決せず静的カタログとして振る舞う（テスト・モック向け）。
type catalog struct {
	improvements repository.ImprovementRepository
}

// NewCatalog は静的カタログを返す（Appendix は常に空）。
func NewCatalog() gateway.PromptProvider { return catalog{} }

// NewRepoCatalog は改善リポジトリを参照するカタログを返す。採用/試用版（canary 以降）
// でタグ付けされた版については、その版を生成した改善の ProposedChange を Appendix に
// 載せる。
func NewRepoCatalog(improvements repository.ImprovementRepository) gateway.PromptProvider {
	return catalog{improvements: improvements}
}

// For は指定版・箇所のプロンプトを返す。Base は prompt_v1 の書式文字列。version が
// 既定版（または不明）なら Appendix は空。それ以外は、その CandidateVersion を持つ
// prompt 改善の ProposedChange を Appendix にする。
//
// リポジトリ読み込みに失敗した場合は Appendix を空にフォールバックする（既定版と
// 同じ挙動＝パイプラインを止めない）。ここは版に応じた「追加指示」の解決であり、
// 失敗しても Base で判断は継続できる。
func (c catalog) For(version string, key gateway.PromptKey) gateway.PromptTemplate {
	base := v1[key]
	if c.improvements == nil || version == "" || version == model.CurrentAgentVersion {
		return gateway.PromptTemplate{Base: base}
	}
	imps, err := c.improvements.ListImprovements()
	if err != nil {
		return gateway.PromptTemplate{Base: base}
	}
	for _, imp := range imps {
		// version を生成した prompt 改善を照合する。status は問わない（当該版で作られた
		// レコードのプロンプトを後から忠実に再現するため。F-056 の監査整合）。
		if imp.CandidateVersion == version && imp.Target == "prompt" {
			return gateway.PromptTemplate{Base: base, Appendix: imp.ProposedChange}
		}
	}
	return gateway.PromptTemplate{Base: base}
}
