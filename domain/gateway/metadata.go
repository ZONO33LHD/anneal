package gateway

import (
	"context"

	"github.com/ZONO33LHD/anneal/domain/model"
)

// MetadataSource は「最新バージョンは何か」「アドバイザリは存在するか」に答える。
type MetadataSource interface {
	ProviderName() string
	LatestVersion(ctx context.Context, eco model.Ecosystem, name, current string) (string, error)
	Advisories(ctx context.Context, eco model.Ecosystem, name, current string) ([]model.CVEInfo, error)
}
