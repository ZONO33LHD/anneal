package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ZONO33LHD/anneal/domain/gateway"
)

// Gemini calls the Generative Language REST API directly (no SDK dependency),
// defaulting to the cheapest model tier.
type Gemini struct {
	apiKey string
	model  string
	http   *http.Client
}

// NewGemini builds a Gemini client. model defaults to gemini-2.5-flash-lite.
func NewGemini(apiKey, model string) gateway.LLM {
	if model == "" {
		model = "gemini-2.5-flash-lite"
	}
	return &Gemini{apiKey: apiKey, model: model, http: &http.Client{Timeout: 30 * time.Second}}
}

func (g *Gemini) Name() string {
	return "gemini"
}
func (g *Gemini) Model() string {
	return g.model
}

func (g *Gemini) Generate(ctx context.Context, req gateway.LLMRequest) (string, error) {
	prompt := req.Prompt
	if req.System != "" {
		prompt = req.System + "\n\n" + req.Prompt
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	body, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{{"parts": []map[string]any{{"text": prompt}}}},
		"generationConfig": map[string]any{
			"temperature":     req.Temperature,
			"maxOutputTokens": maxTokens,
		},
	})
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := g.http.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini: status %d", resp.StatusCode)
	}
	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Candidates) > 0 && len(out.Candidates[0].Content.Parts) > 0 {
		return out.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", nil
}
