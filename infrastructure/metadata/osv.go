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
// Go module proxy から、アドバイザリは OSV.dev から取得する。HTTP/JSON エラーは
// 呼び出し側で「更新なし/CVEなし」と区別できるよう、握りつぶさず error として返す。
type OSV struct {
	http *http.Client
}

// NewOSV はライブのメタデータソースを返す。
func NewOSV() gateway.MetadataSource {
	return &OSV{http: &http.Client{Timeout: 15 * time.Second}}
}

func (OSV) ProviderName() string {
	return "osv+registry"
}

func (o *OSV) LatestVersion(ctx context.Context, eco model.Ecosystem, name, _ string) (string, error) {
	if eco == model.EcosystemNPM {
		var out struct {
			Version string `json:"version"`
		}
		if err := o.getJSON(ctx, "https://registry.npmjs.org/"+npmPathEscape(name)+"/latest", &out); err != nil {
			return "", fmt.Errorf("npm latest %s: %w", name, err)
		}
		return out.Version, nil
	}
	var out struct {
		Version string `json:"Version"`
	}
	// Go module proxy はモジュールパスのスラッシュを保持し、大文字だけを !小文字 に
	// エスケープする（url.PathEscape では / が %2F になり壊れる）。
	if err := o.getJSON(ctx, "https://proxy.golang.org/"+escapeGoModulePath(name)+"/@latest", &out); err != nil {
		return "", fmt.Errorf("go proxy latest %s: %w", name, err)
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
		return nil, fmt.Errorf("osv request %s: %w", name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("osv query %s: %w", name, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osv query %s: status %d", name, resp.StatusCode)
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
		return nil, fmt.Errorf("osv decode %s: %w", name, err)
	}
	cves := make([]model.CVEInfo, 0, len(out.Vulns))
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

// escapeGoModulePath は Go module proxy のパスエスケープを行う。スラッシュは保持し、
// 大文字 X は "!x" に変換する（例: github.com/BurntSushi/toml → github.com/!burnt!sushi/toml）。
func escapeGoModulePath(path string) string {
	var b strings.Builder
	for _, r := range path {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// npmPathEscape はパッケージ名をエスケープする。スコープ付き (@scope/name) の
// スラッシュのみ %2F にし、@ は npm レジストリの仕様どおり残す。
func npmPathEscape(name string) string {
	if strings.HasPrefix(name, "@") && strings.Contains(name, "/") {
		return "@" + url.PathEscape(strings.TrimPrefix(name, "@"))
	}
	return url.PathEscape(name)
}
