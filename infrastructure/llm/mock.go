// Package llm は LLM のポートを実装する（モック + Gemini）。
package llm

import (
	"context"
	"regexp"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

// Mock は決定論的な LLM。プロンプト先頭のタグ（例: "[pr-body]"）を認識し、
// もっともらしくタスクに即したテキストを返す。これによりデモを再現可能に保ち、
// テストをネットワークや鍵に依存させずに済む。
type Mock struct{}

// NewMock はモックの LLM を返す。
func NewMock() gateway.LLM {
	return Mock{}
}

func (Mock) ProviderName() string {
	return "mock"
}
func (Mock) Model() string {
	return "mock-flash-lite"
}

var tagRe = regexp.MustCompile(`^\[([a-z-]+)\]`)

func (Mock) Generate(_ context.Context, req gateway.LLMRequest) (string, error) {
	tag := ""
	if m := tagRe.FindStringSubmatch(req.Prompt); m != nil {
		tag = m[1]
	}
	switch tag {
	case "pr-body":
		return "This update keeps the dependency current and pulls in upstream fixes. Risk is contained to the documented usage sites; tests cover the affected paths.", nil
	case "ci-summary":
		return "The build failed because the bumped package changed an exported signature; the call site needs a small adjustment. This looks mechanically fixable.", nil
	case "hypothesis":
		return "Risk for minor bumps of widely-imported packages is being underestimated, leading to surprise CI failures. The classifier should weight import breadth more heavily.", nil
	case "prompt-improvement":
		return "Add an explicit instruction: \"When >10 import sites exist, raise risk one level and require a usage-diff before auto-PR.\" This should reduce false LOW-risk calls.", nil
	default:
		return "OK", nil
	}
}
