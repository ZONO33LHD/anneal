package testutil

import (
	"github.com/ZONO33LHD/anneal/domain/config"
	"github.com/ZONO33LHD/anneal/registry"
)

// NewRegistry はテスト用に、すべてモックでインメモリな registry を構築します。
func NewRegistry() *registry.Registry {
	return registry.New(&config.Config{
		StorePath:         "", // インメモリ
		ImproveWindow:     5,
		LowScoreThreshold: 90,
		ForceMock:         true,
	})
}
