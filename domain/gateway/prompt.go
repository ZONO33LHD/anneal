package gateway

// PromptKey は版管理されるプロンプトの呼び出し箇所を識別する。
type PromptKey string

const (
	// PromptImpact は影響範囲分析（リスク/判断）のプロンプト。改善ループの
	// ProposedChange が狙う唯一の版管理対象（dig で確定）。
	PromptImpact PromptKey = "impact"
)

// PromptTemplate は版で解決されたプロンプト。Base は fmt の書式文字列、Appendix は
// 整形の「後」に連結される追加指示である。生成文（ProposedChange）を Appendix に
// 載せても書式指定子（%）として誤解釈されないよう、両者を分けて保持する。
//
// A フェーズ（本 PR）では Appendix は常に空で、Base は現行のハードコード文言を
// 逐語再現する。C フェーズで採用版の ProposedChange が Appendix に入る。
type PromptTemplate struct {
	Base     string
	Appendix string
}

// PromptProvider はエージェント版と呼び出し箇所から使用するプロンプトを解決する。
// 未知の版は既定版（prompt_v1）にフォールバックし、パイプラインを止めない。
type PromptProvider interface {
	For(version string, key PromptKey) PromptTemplate
}
