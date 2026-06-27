// Package config は実行時設定を環境から読み込む。値を保持するだけで、
// それらからどのプロバイダを組み立てるかはレジストリが決める。
package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/ZONO33LHD/anneal/domain/policy"
)

// Config はすべての実行時設定を保持する。
type Config struct {
	StorePath         string
	ImproveWindow     int
	LowScoreThreshold float64

	// シークレット。空値はその機能について「モックを使う」ことを意味する。
	GeminiAPIKey    string
	GeminiModel     string
	GitHubToken     string
	SlackWebhookURL string

	// ForceMock はすべてのプロバイダをモック/オフライン形態に強制する（デモで使用）。
	ForceMock bool
	// Verbose はデバッグログを有効にする。
	Verbose bool
}

// Load は環境（および任意の .env ファイル）から設定を読み込む。
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

// loadDotEnv は .env ファイルから KEY=VALUE 行を読み込む。既存の環境変数は
// 上書きしない。ファイルが存在しなくても問題ない。
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
