package prompt

import (
	"testing"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
)

// impactGolden は usecase/engine_analysis.go の現行 [impact] プロンプト文言の逐語コピー。
// 4b で呼び出し側をカタログ経由へ差し替えても LLM への入力が 1 文字も変わらない
// （= 既存の全レコードのスコア基準が不変）ことを、この golden で守る。
// engine_analysis.go の文言を変えたらここも同時に更新する意図的な二重管理。
const impactGolden = "[impact] Summarize the impact of bumping %s from %s to %s (%s, %d usage sites, risk=%s)."

// prompt_v1 の判断プロンプトが現行のハードコード文言と完全一致すること。ここがズレると
// 差し替え後に全 PR のスコア基準が静かに変わってしまうため、挙動不変の要である。
func TestCatalog_PromptV1ImpactIsVerbatim(t *testing.T) {
	c := NewCatalog()
	if got := c.For(model.CurrentAgentVersion, gateway.PromptKeyImpact); got != impactGolden {
		t.Fatalf("prompt_v1 impact template drifted from source literal:\n got=%q\nwant=%q", got, impactGolden)
	}
}

// ループが自動生成した未知の版（例: prompt_v2）は、A ではカタログに実体を持たないため
// 既定版 prompt_v1 の文言へフォールバックする（設計方針 A1）。これにより採用版が
// あってもパイプラインは止まらず、挙動は既定のまま保たれる。
func TestCatalog_UnknownVersionFallsBackToV1(t *testing.T) {
	c := NewCatalog()
	if got := c.For("prompt_v2", gateway.PromptKeyImpact); got != impactGolden {
		t.Fatalf("unknown version should fall back to prompt_v1 impact, got=%q", got)
	}
}

// 未知の種別キーは（フォールバックしても実体が無いため）空文字を返す。呼び出し側が
// 空を「LLM 補強なし」として扱える契約を明示する。
func TestCatalog_UnknownKeyReturnsEmpty(t *testing.T) {
	c := NewCatalog()
	if got := c.For(model.CurrentAgentVersion, "no-such-key"); got != "" {
		t.Fatalf("unknown key should return empty string, got=%q", got)
	}
}
