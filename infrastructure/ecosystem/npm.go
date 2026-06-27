// Package ecosystem implements manifest reading/writing and source scanning for
// the supported package ecosystems.
package ecosystem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
)

// NPM handles package.json dependencies.
type NPM struct{}

func (NPM) ID() model.Ecosystem {
	return model.EcosystemNPM
}

func (NPM) Detect(repoPath string) bool {
	return fileExists(filepath.Join(repoPath, "package.json"))
}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func (NPM) Scan(repoPath string) ([]gateway.Dependency, error) {
	data, err := os.ReadFile(filepath.Join(repoPath, "package.json"))
	if err != nil {
		return nil, err
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	var out []gateway.Dependency
	add := func(deps map[string]string, dev bool) {
		names := make([]string, 0, len(deps))
		for n := range deps {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			out = append(out, gateway.Dependency{Name: n, CurrentVersion: deps[n], IsDev: dev, Ecosystem: model.EcosystemNPM})
		}
	}
	add(pkg.Dependencies, false)
	add(pkg.DevDependencies, true)
	return out, nil
}

// ApplyUpdate rewrites the version via a targeted replace so the file's
// formatting and key order are preserved.
func (NPM) ApplyUpdate(repoPath, name, target string) ([]string, error) {
	pkgPath := filepath.Join(repoPath, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`("` + regexp.QuoteMeta(name) + `"\s*:\s*")([\^~]?)[^"]*(")`)
	updated := re.ReplaceAll(data, []byte("${1}${2}"+target+"${3}"))
	var changed []string
	if string(updated) != string(data) {
		if err := os.WriteFile(pkgPath, updated, 0o644); err != nil {
			return nil, err
		}
		changed = append(changed, "package.json")
	}
	if fileExists(filepath.Join(repoPath, "package-lock.json")) {
		// In real mode this is where `npm install` would regenerate the lockfile.
		changed = append(changed, "package-lock.json")
	}
	return changed, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
