package testutil

import (
	"github.com/ZONO33LHD/anneal/domain/config"
	"github.com/ZONO33LHD/anneal/registry"
)

// NewRegistry builds an all-mock, in-memory registry for tests.
func NewRegistry() *registry.Registry {
	return registry.New(&config.Config{
		StorePath:         "", // in-memory
		ImproveWindow:     5,
		LowScoreThreshold: 90,
		ForceMock:         true,
	})
}
