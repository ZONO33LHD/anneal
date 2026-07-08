package prompt

import (
	"testing"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/infrastructure/persistence"
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

func newImprovementRepo(t *testing.T) *persistence.DB {
	t.Helper()
	return persistence.NewDB("")
}

// C-2 の核心: 採用版でタグ付けされたレコードのプロンプトは、その版を生成した改善の
// ProposedChange が Appendix として載る。これが効かないと A/B で版を切り替えても LLM
// への入力が変わらず、「使うほど賢くなる」ループが成立しない。
func TestRepoCatalog_ResolvesAppendixFromImprovement(t *testing.T) {
	db := newImprovementRepo(t)
	repo := persistence.NewImprovementRepository(db)
	const change = "Also weight import breadth more heavily for minor bumps."
	if err := repo.PutImprovement(model.AgentImprovement{
		ImprovementID:    "imp1",
		CandidateVersion: "prompt_v2",
		Target:           "prompt",
		ProposedChange:   change,
		Status:           model.ImprovementCanary,
	}); err != nil {
		t.Fatalf("put improvement: %v", err)
	}

	got := NewRepoCatalog(repo).For("prompt_v2", gateway.PromptImpact)
	if got.Appendix != change {
		t.Fatalf("Appendix=%q want %q", got.Appendix, change)
	}
	// Base は版に関わらず prompt_v1 の文言のまま。
	if got.Base != v1[gateway.PromptImpact] {
		t.Fatalf("Base=%q want v1 wording", got.Base)
	}
}

// 既定版はリポジトリを引かず Appendix 空（＝挙動不変）。
func TestRepoCatalog_DefaultVersion_NoAppendix(t *testing.T) {
	db := newImprovementRepo(t)
	repo := persistence.NewImprovementRepository(db)
	got := NewRepoCatalog(repo).For(model.CurrentAgentVersion, gateway.PromptImpact)
	if got.Appendix != "" {
		t.Fatalf("default version Appendix=%q want empty", got.Appendix)
	}
}
