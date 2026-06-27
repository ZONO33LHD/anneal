// Package metadata implements the MetadataSource port (offline mock + OSV/registry).
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

// Offline dataset mirroring fixtures/sample-repo. Keyed by "ecosystem:name".
var dataset = map[string]entry{
	"npm:lodash": {
		latest: "4.17.21",
		advisories: []model.CVEInfo{{
			ID: "CVE-2021-23337", Severity: "high", AffectedRange: "<4.17.21",
			PatchedVersion: "4.17.21", ExploitAvailable: true,
			Summary: "Command injection via template in lodash.",
		}},
	},
	"npm:axios":      {latest: "1.7.0"},
	"npm:express":    {latest: "4.19.2"},
	"npm:chalk":      {latest: "5.3.0"}, // major: ESM-only, breaking
	"npm:typescript": {latest: "5.6.2"},
	"go:github.com/gin-gonic/gin": {latest: "v1.9.1"},
	"go:golang.org/x/crypto": {
		latest: "v0.17.0",
		advisories: []model.CVEInfo{{
			ID: "CVE-2023-48795", Severity: "moderate", AffectedRange: "<v0.17.0",
			PatchedVersion: "v0.17.0",
			Summary:        "Terrapin attack on SSH transport (golang.org/x/crypto/ssh).",
		}},
	},
}

// Mock is an offline metadata source for deterministic demos.
type Mock struct{}

// NewMock returns a mock metadata source.
func NewMock() gateway.MetadataSource { return Mock{} }

func (Mock) Name() string { return "mock" }

func (Mock) LatestVersion(_ context.Context, eco model.Ecosystem, name, _ string) (string, error) {
	return dataset[string(eco)+":"+name].latest, nil
}

func (Mock) Advisories(_ context.Context, eco model.Ecosystem, name, _ string) ([]model.CVEInfo, error) {
	return dataset[string(eco)+":"+name].advisories, nil
}
