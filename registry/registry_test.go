package registry_test

import (
	"strings"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/config"
	"github.com/ZONO33LHD/anneal/registry"
)

func TestNewUsesJSONBackendByDefault(t *testing.T) {
	reg, err := registry.New(&config.Config{
		StorePath:         "",
		ImproveWindow:     5,
		LowScoreThreshold: 90,
		StoreBackend:      config.StoreBackendJSON,
		ForceMock:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reg.Updates == nil || reg.Evaluations == nil || reg.Improvements == nil {
		t.Fatal("repositories should be wired")
	}
}

func TestNewFirestoreRequiresProjectID(t *testing.T) {
	_, err := registry.New(&config.Config{
		StoreBackend: config.StoreBackendFirestore,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "ANNEAL_FIRESTORE_PROJECT") {
		t.Fatalf("error = %q, want project id hint", err.Error())
	}
}

func TestNewForceMockKeepsJSONBackend(t *testing.T) {
	reg, err := registry.New(&config.Config{
		StorePath:         "",
		ImproveWindow:     5,
		LowScoreThreshold: 90,
		StoreBackend:      config.StoreBackendFirestore,
		ForceMock:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reg.Updates == nil {
		t.Fatal("updates repository should be wired")
	}
}

func TestNewRejectsUnknownBackend(t *testing.T) {
	_, err := registry.New(&config.Config{
		StoreBackend: "sqlite",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown store backend") {
		t.Fatalf("error = %q, want unknown backend hint", err.Error())
	}
}
