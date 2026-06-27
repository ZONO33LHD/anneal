// Package config loads runtime configuration from the environment. It holds
// values only; the registry decides which providers to wire from them.
package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/ZONO33LHD/anneal/domain/policy"
)

// Config holds all runtime configuration.
type Config struct {
	StorePath         string
	ImproveWindow     int
	LowScoreThreshold float64

	// Secrets. An empty value means "use the mock for that capability".
	GeminiAPIKey    string
	GeminiModel     string
	GitHubToken     string
	SlackWebhookURL string

	// ForceMock forces every provider to its mock/offline form (used by demo).
	ForceMock bool
	// Verbose enables debug logging.
	Verbose bool
}

// Load reads configuration from the environment (and an optional .env file).
func Load() (*Config, error) {
	loadDotEnv(".env")
	return &Config{
		StorePath:         envOr("ANNEAL_STORE_PATH", ".anneal/store.json"),
		ImproveWindow:     envInt("ANNEAL_IMPROVE_WINDOW", policy.DefaultImproveWindow),
		LowScoreThreshold: envFloat("ANNEAL_LOW_SCORE_THRESHOLD", policy.DefaultLowScoreThreshold),
		GeminiAPIKey:      os.Getenv("GEMINI_API_KEY"),
		GeminiModel:       envOr("ANNEAL_LLM_MODEL", "gemini-2.5-flash-lite"),
		GitHubToken:       os.Getenv("GITHUB_TOKEN"),
		SlackWebhookURL:   os.Getenv("SLACK_WEBHOOK_URL"),
		Verbose:           os.Getenv("ANNEAL_DEBUG") == "1",
	}, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(key), 64); err == nil && v > 0 {
		return v
	}
	return def
}

// loadDotEnv loads KEY=VALUE lines from a .env file without overwriting existing
// environment variables. A missing file is fine.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if k, v = strings.TrimSpace(k), strings.TrimSpace(v); k != "" {
			if _, exists := os.LookupEnv(k); !exists {
				_ = os.Setenv(k, v)
			}
		}
	}
}
