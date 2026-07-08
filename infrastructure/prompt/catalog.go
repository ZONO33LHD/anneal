// Package prompt はコンパイル時に埋め込まれたプロンプトカタログを提供する。
// version → プロンプトの対応を一箇所に集約し、採用版に応じた差し替え（T10）の
// 土台とする。
package prompt

import "github.com/ZONO33LHD/anneal/domain/gateway"

// v1 は既定版 prompt_v1 のプロンプト。文言は engine 内のハードコード実装を逐語で
// 再現する（挙動不変・golden test で差分ゼロを担保）。Base は fmt の書式文字列。
var v1 = map[gateway.PromptKey]string{
	gateway.PromptImpact: "[impact] Summarize the impact of bumping %s from %s to %s (%s, %d usage sites, risk=%s).",
}

// catalog はコンパイル時カタログの実装。A フェーズでは版に関わらず prompt_v1 を返す
// （Appendix は空）。C フェーズで採用版の ProposedChange を Appendix に載せる分岐を
// ここに足す。
type catalog struct{}

// NewCatalog は埋め込みカタログを返す。
func NewCatalog() gateway.PromptProvider { return catalog{} }

// For は指定版・箇所のプロンプトを返す。A フェーズでは version を用いず prompt_v1 に
// 解決する（＝未知版フォールバックと同じ挙動）。
func (catalog) For(_ string, key gateway.PromptKey) gateway.PromptTemplate {
	return gateway.PromptTemplate{Base: v1[key]}
}
