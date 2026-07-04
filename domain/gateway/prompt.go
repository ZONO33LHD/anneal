package gateway

// プロンプト種別のキー。版管理の対象は「判断（リスク/影響分析）プロンプト」1 本のみ
// （engine_analysis の [impact]）。改善ループの ProposedChange が狙う対象と一致する。
// 出力系（ci-summary / pr-body）やメタプロンプトは版管理しない。
const (
	PromptKeyImpact = "impact"
)

// PromptTemplate は 1 つのプロンプトを「ベース + 追記」で表す（設計方針 A''）。
// A（現段階）ではすべての版で Appendix は空。C で Appendix に採用版の ProposedChange を
// 詰めることで、ポート契約 For を変えずに自律改善へ拡張できる。
type PromptTemplate struct {
	Base     string
	Appendix string
}

// Rendered は Base と Appendix を結合した最終テンプレート文字列を返す。
// 呼び出し側はこの文字列に fmt.Sprintf で引数を埋める。
func (t PromptTemplate) Rendered() string {
	if t.Appendix == "" {
		return t.Base
	}
	return t.Base + "\n" + t.Appendix
}

// PromptProvider は「エージェント版 + プロンプト種別」から、その版のプロンプト
// テンプレート文字列を解決する。未知の版・種別は既定版（model.CurrentAgentVersion =
// prompt_v1）へフォールバックする（設計方針 A1）ため、常に空でない文字列を返せる限り
// パイプラインを止めない。
type PromptProvider interface {
	For(version, key string) string
}
