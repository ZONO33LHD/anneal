// Package llm implements the LLM port (mock + Gemini).
package llm

import (
	"context"
	"regexp"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

// Mock is a deterministic LLM. It recognises a tag at the start of the prompt
// (e.g. "[pr-body]") and returns plausible, task-aware text, keeping demos
// reproducible and tests free of network/keys.
type Mock struct{}

// NewMock returns a mock LLM.
func NewMock() gateway.LLM {
	return Mock{}
}

func (Mock) Name() string {
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
