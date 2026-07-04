package config_test

import (
	"testing"

	"github.com/ZONO33LHD/anneal/domain/config"
)

func TestLoadStoreBackendDefaultsToJSON(t *testing.T) {
	t.Setenv("ANNEAL_STORE_BACKEND", "")
	t.Setenv("ANNEAL_FIRESTORE_PROJECT", "")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StoreBackend != config.StoreBackendJSON {
		t.Fatalf("backend = %q, want %q", cfg.StoreBackend, config.StoreBackendJSON)
	}
	if cfg.FirestoreProjectID != "" {
		t.Fatalf("firestore project = %q, want empty", cfg.FirestoreProjectID)
	}
}

func TestLoadFirestoreProjectFallback(t *testing.T) {
	t.Setenv("ANNEAL_STORE_BACKEND", config.StoreBackendFirestore)
	t.Setenv("ANNEAL_FIRESTORE_PROJECT", "")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "fallback-project")
	t.Setenv("ANNEAL_FIRESTORE_PREFIX", "test_prefix")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StoreBackend != config.StoreBackendFirestore {
		t.Fatalf("backend = %q, want %q", cfg.StoreBackend, config.StoreBackendFirestore)
	}
	if cfg.FirestoreProjectID != "fallback-project" {
		t.Fatalf("firestore project = %q, want fallback-project", cfg.FirestoreProjectID)
	}
	if cfg.FirestoreCollectionPrefix != "test_prefix" {
		t.Fatalf("prefix = %q, want test_prefix", cfg.FirestoreCollectionPrefix)
	}
}

func TestLoadFirestoreProjectOverride(t *testing.T) {
	t.Setenv("ANNEAL_STORE_BACKEND", config.StoreBackendFirestore)
	t.Setenv("ANNEAL_FIRESTORE_PROJECT", "explicit-project")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "fallback-project")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FirestoreProjectID != "explicit-project" {
		t.Fatalf("firestore project = %q, want explicit-project", cfg.FirestoreProjectID)
	}
}

func TestLoadInternalTaskConfig(t *testing.T) {
	t.Setenv("ANNEAL_INTERNAL_TOKEN", "internal-secret")
	t.Setenv("ANNEAL_INTERNAL_OIDC_AUDIENCE", "https://anneal.example.run.app")
	t.Setenv("ANNEAL_INTERNAL_OIDC_EMAIL", "scheduler@example.iam.gserviceaccount.com")
	t.Setenv("ANNEAL_SCAN_TARGETS", "/work/repo-a=acme/repo-a, /work/repo-b")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InternalToken != "internal-secret" {
		t.Fatalf("internal token = %q, want configured value", cfg.InternalToken)
	}
	if cfg.InternalOIDCAudience != "https://anneal.example.run.app" {
		t.Fatalf("oidc audience = %q", cfg.InternalOIDCAudience)
	}
	if cfg.InternalOIDCEmail != "scheduler@example.iam.gserviceaccount.com" {
		t.Fatalf("oidc email = %q", cfg.InternalOIDCEmail)
	}
	if len(cfg.ScanTargets) != 2 {
		t.Fatalf("scan targets len = %d, want 2", len(cfg.ScanTargets))
	}
	if cfg.ScanTargets[0].Path != "/work/repo-a" || cfg.ScanTargets[0].Repository != "acme/repo-a" {
		t.Fatalf("target[0] = %#v", cfg.ScanTargets[0])
	}
	if cfg.ScanTargets[1].Path != "/work/repo-b" || cfg.ScanTargets[1].Repository != "" {
		t.Fatalf("target[1] = %#v", cfg.ScanTargets[1])
	}
}
