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
