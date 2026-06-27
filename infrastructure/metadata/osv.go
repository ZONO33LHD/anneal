package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
)

// OSV は標準の HTTP クライアントのみを使用する。最新バージョンは npm レジストリ /
// Go module proxy から、アドバイザリは OSV.dev から取得する。ネットワーク障害時は
// 「更新なし / アドバイザリなし」へグレースフルに縮退する。
type OSV struct {
	http *http.Client
}

// NewOSV はライブのメタデータソースを返す。
func NewOSV() gateway.MetadataSource {
	return &OSV{http: &http.Client{Timeout: 15 * time.Second}}
}

func (OSV) Name() string {
	return "osv+registry"
}

func (o *OSV) LatestVersion(ctx context.Context, eco model.Ecosystem, name, _ string) (string, error) {
	if eco == model.EcosystemNPM {
		var out struct {
			Version string `json:"version"`
		}
		if err := o.getJSON(ctx, "https://registry.npmjs.org/"+url.PathEscape(name)+"/latest", &out); err != nil {
			return "", nil
		}
		return out.Version, nil
	}
	var out struct {
		Version string `json:"Version"`
	}
	if err := o.getJSON(ctx, "https://proxy.golang.org/"+url.PathEscape(name)+"/@latest", &out); err != nil {
		return "", nil
	}
	return out.Version, nil
}

func (o *OSV) Advisories(ctx context.Context, eco model.Ecosystem, name, current string) ([]model.CVEInfo, error) {
	ecoName := "npm"
	if eco == model.EcosystemGo {
		ecoName = "Go"
	}
	body, _ := json.Marshal(map[string]any{
		"version": model.CleanVersion(current),
		"package": map[string]string{"name": name, "ecosystem": ecoName},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.osv.dev/v1/query", bytes.NewReader(body))
	if err != nil {
		return nil, nil
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.http.Do(req)
	if err != nil {
		return nil, nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}
	var out struct {
		Vulns []struct {
			ID               string   `json:"id"`
			Summary          string   `json:"summary"`
			Aliases          []string `json:"aliases"`
			DatabaseSpecific struct {
				Severity string `json:"severity"`
			} `json:"database_specific"`
			Affected []struct {
				Ranges []struct {
					Events []struct {
						Fixed string `json:"fixed"`
					} `json:"events"`
				} `json:"ranges"`
			} `json:"affected"`
		} `json:"vulns"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, nil
	}
	var cves []model.CVEInfo
	for _, v := range out.Vulns {
		id := v.ID
		for _, a := range v.Aliases {
			if strings.HasPrefix(a, "CVE-") {
				id = a
				break
			}
		}
		sev := strings.ToLower(v.DatabaseSpecific.Severity)
		if !slices.Contains([]string{"critical", "high", "moderate", "low"}, sev) {
			sev = "moderate"
		}
		fixed := "unknown"
		for _, af := range v.Affected {
			for _, r := range af.Ranges {
				for _, e := range r.Events {
					if e.Fixed != "" {
						fixed = e.Fixed
					}
				}
			}
		}
		cves = append(cves, model.CVEInfo{ID: id, Severity: sev, AffectedRange: "see advisory", PatchedVersion: fixed, Summary: v.Summary})
	}
	return cves, nil
}

func (o *OSV) getJSON(ctx context.Context, u string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := o.http.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
