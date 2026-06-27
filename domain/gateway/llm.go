package gateway

import "context"

// LLMRequest は単一の生成呼び出しを表す。
type LLMRequest struct {
	System      string
	Prompt      string
	Temperature float64
	MaxTokens   int
}

// LLM はプロンプトからテキストを生成する。エージェントは文章生成や情報の付加に
// のみ利用するため、モック実装でもパイプラインは end-to-end で動作する。
type LLM interface {
	Name() string
	Model() string
	Generate(ctx context.Context, req LLMRequest) (string, error)
}
