package prompt

import (
	"testing"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
)

// prompt_v1 の impact プロンプトは、engine_analysis.go がこれまでハードコードしていた
// 文言と 1 文字も違ってはならない。ズレると 4b で呼び出し箇所を差し替えた瞬間に LLM 出力
// が変わり、全レコードのスコア基準が静かにずれる（roadmap 懸念 #7）。この golden test は
// 「移行しても観測挙動が変わらない」ことを 4a 単独で保証するためにある。
func TestCatalog_PromptV1Impact_MatchesEngineWording(t *testing.T) {
	const want = "[impact] Summarize the impact of bumping %s from %s to %s (%s, %d usage sites, risk=%s)."

	got := NewCatalog().For(model.CurrentAgentVersion, gateway.PromptImpact)
	if got.Base != want {
		t.Fatalf("prompt_v1 impact Base=%q\nwant %q", got.Base, want)
	}
	// A フェーズでは生成文を載せないので Appendix は必ず空。
	if got.Appendix != "" {
		t.Fatalf("prompt_v1 impact Appendix=%q want empty", got.Appendix)
	}
}

// 未知の版は既定版（prompt_v1）に解決され、パイプラインを止めない（A1 フォールバック）。
func TestCatalog_UnknownVersion_FallsBackToV1(t *testing.T) {
	base := NewCatalog()
	want := base.For(model.CurrentAgentVersion, gateway.PromptImpact)
	got := base.For("prompt_vX-does-not-exist", gateway.PromptImpact)
	if got != want {
		t.Fatalf("unknown version=%+v\nwant fallback to v1=%+v", got, want)
	}
}
