// Package prompt は版で引けるプロンプトカタログ（gateway.PromptProvider の実装）を提供する。
// 依存方向: registry -> infrastructure/prompt -> domain (gateway/model)。
package prompt

import (
	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
)

// catalog は「版 -> (種別 -> テンプレート)」の静的カタログ。A（現段階）では既定版
// prompt_v1 のみを保持し、Appendix はすべて空（現行のハードコード文言と 1 文字も
// 変えない）。ループが自動生成した版など未知の版・種別は既定版の該当テンプレートへ
// フォールバックする（設計方針 A1）。
type catalog struct {
	versions map[string]map[string]gateway.PromptTemplate
}

// NewCatalog は既定カタログを組み込んだ PromptProvider を返す。
func NewCatalog() gateway.PromptProvider {
	return &catalog{versions: defaultCatalog()}
}

// defaultCatalog は prompt_v1 の各テンプレートを保持する。
// impact テンプレートは usecase/engine_analysis.go の現行 [impact] 文言の逐語コピーであり、
// catalog_test.go の golden test が両者の一致（挙動不変）を担保する。
func defaultCatalog() map[string]map[string]gateway.PromptTemplate {
	return map[string]map[string]gateway.PromptTemplate{
		model.CurrentAgentVersion: {
			gateway.PromptKeyImpact: {
				Base: "[impact] Summarize the impact of bumping %s from %s to %s (%s, %d usage sites, risk=%s).",
			},
		},
	}
}

// For は版・種別からテンプレート文字列を解決する。未知の版・種別は既定版へ
// フォールバックし、それでも見つからなければ空文字を返す。
func (c *catalog) For(version, key string) string {
	if v, ok := c.versions[version]; ok {
		if t, ok := v[key]; ok {
			return t.Rendered()
		}
	}
	if base, ok := c.versions[model.CurrentAgentVersion]; ok {
		if t, ok := base[key]; ok {
			return t.Rendered()
		}
	}
	return ""
}
