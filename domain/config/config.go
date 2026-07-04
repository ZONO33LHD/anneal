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
	StorePath                 string
	StoreBackend              string
	FirestoreProjectID        string
	FirestoreCollectionPrefix string
	ImproveWindow             int
	LowScoreThreshold         float64
	HTTPAddr                  string
	InternalToken             string
	InternalOIDCAudience      string
	InternalOIDCEmail         string
	ScanTargets               []ScanTarget

	// シークレット。空値はその機能について「モックを使う」ことを意味する。
	GeminiAPIKey        string
	GeminiModel         string
	GitHubToken         string
	GitHubWebhookSecret string
	SlackWebhookURL     string

	// ForceMock はすべてのプロバイダをモック/オフライン形態に強制する（デモで使用）。
	ForceMock bool
	// Verbose はデバッグログを有効にする。
	Verbose bool
	// LogJSON は true で severity 付き JSON ログ（Cloud Run 等）、false で text を出す。
	LogJSON bool
}

// ScanTarget は定期 scan endpoint が実行する対象 repository を表す。
type ScanTarget struct {
	Path       string
	Repository string
}

// Load は環境（および任意の .env ファイル）から設定を読み込む。
func Load() (*Config, error) {
	loadDotEnv(".env")
	return &Config{
		StorePath:                 envOr("ANNEAL_STORE_PATH", ".anneal/store.json"),
		StoreBackend:              envOr("ANNEAL_STORE_BACKEND", StoreBackendJSON),
		FirestoreProjectID:        firestoreProjectID(),
		FirestoreCollectionPrefix: os.Getenv("ANNEAL_FIRESTORE_PREFIX"),
		ImproveWindow:             envInt("ANNEAL_IMPROVE_WINDOW", policy.DefaultImproveWindow),
		LowScoreThreshold:         envFloat("ANNEAL_LOW_SCORE_THRESHOLD", policy.DefaultLowScoreThreshold),
		HTTPAddr:                  envOr("ANNEAL_HTTP_ADDR", defaultHTTPAddr()),
		InternalToken:             os.Getenv("ANNEAL_INTERNAL_TOKEN"),
		InternalOIDCAudience:      os.Getenv("ANNEAL_INTERNAL_OIDC_AUDIENCE"),
		InternalOIDCEmail:         os.Getenv("ANNEAL_INTERNAL_OIDC_EMAIL"),
		ScanTargets:               parseScanTargets(os.Getenv("ANNEAL_SCAN_TARGETS")),
		GeminiAPIKey:              os.Getenv("GEMINI_API_KEY"),
		GeminiModel:               envOr("ANNEAL_LLM_MODEL", "gemini-2.5-flash-lite"),
		GitHubToken:               os.Getenv("GITHUB_TOKEN"),
		GitHubWebhookSecret:       os.Getenv("GITHUB_WEBHOOK_SECRET"),
		SlackWebhookURL:           os.Getenv("SLACK_WEBHOOK_URL"),
		Verbose:                   os.Getenv("ANNEAL_DEBUG") == "1",
		LogJSON:                   os.Getenv("ANNEAL_LOG_FORMAT") == "json",
	}, nil
}

const (
	// StoreBackendJSON はローカル JSON ファイルを使う既定の永続化 backend である。
	StoreBackendJSON = "json"
	// StoreBackendFirestore は Firestore を使う永続化 backend である。
	StoreBackendFirestore = "firestore"
)

// defaultHTTPAddr は HTTP リッスンアドレスの既定値を返す。Cloud Run などは待受ポートを
// PORT 環境変数で注入するため、ANNEAL_HTTP_ADDR が未指定なら PORT を尊重する。
// どちらも無ければ :8080 にフォールバックする。
func defaultHTTPAddr() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return ":8080"
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

func firestoreProjectID() string {
	if v := os.Getenv("ANNEAL_FIRESTORE_PROJECT"); v != "" {
		return v
	}
	return os.Getenv("GOOGLE_CLOUD_PROJECT")
}

func parseScanTargets(raw string) []ScanTarget {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	targets := make([]ScanTarget, 0)
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		path, repo, hasRepo := strings.Cut(item, "=")
		path = strings.TrimSpace(path)
		repo = strings.TrimSpace(repo)
		if path == "" {
			continue
		}
		target := ScanTarget{Path: path}
		if hasRepo {
			target.Repository = repo
		}
		targets = append(targets, target)
	}
	return targets
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
