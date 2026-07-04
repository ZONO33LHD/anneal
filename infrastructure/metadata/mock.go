// Package metadata は MetadataSource のポートを実装する（オフラインモック + OSV/レジストリ）。
package metadata

import (
	"context"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
)

type entry struct {
	latest     string
	advisories []model.CVEInfo
}

// fixtures/sample-repo を反映したオフラインのデータセット。キーは "ecosystem:name"。
var dataset = map[string]entry{
	"npm:lodash": {
		latest: "4.17.21",
		advisories: []model.CVEInfo{{
			ID: "CVE-2021-23337", Severity: "high", AffectedRange: "<4.17.21",
			PatchedVersion: "4.17.21", ExploitAvailable: true,
			Summary: "Command injection via template in lodash.",
		}},
	},
	"npm:axios":                   {latest: "1.7.0"},
	"npm:express":                 {latest: "4.19.2"},
	"npm:chalk":                   {latest: "5.3.0"}, // メジャー: ESM 専用、破壊的変更
	"npm:typescript":              {latest: "5.6.2"},
	"go:github.com/gin-gonic/gin": {latest: "v1.9.1"},
	"go:golang.org/x/crypto": {
		latest: "v0.17.0",
		advisories: []model.CVEInfo{{
			ID: "CVE-2023-48795", Severity: "moderate", AffectedRange: "<v0.17.0",
			PatchedVersion: "v0.17.0",
			Summary:        "Terrapin attack on SSH transport (golang.org/x/crypto/ssh).",
		}},
	},
	"pypi:requests": {latest: "2.32.3"},
}

// Mock は決定論的なデモ向けのオフラインメタデータソース。
type Mock struct{}

// NewMock はモックのメタデータソースを返す。
func NewMock() gateway.MetadataSource {
	return Mock{}
}

func (Mock) ProviderName() string {
	return "mock"
}

func (Mock) LatestVersion(_ context.Context, eco model.Ecosystem, name, _ string) (string, error) {
	return dataset[string(eco)+":"+name].latest, nil
}

func (Mock) Advisories(_ context.Context, eco model.Ecosystem, name, _ string) ([]model.CVEInfo, error) {
	return dataset[string(eco)+":"+name].advisories, nil
}
