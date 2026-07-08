// Package ecosystem は、サポート対象のパッケージエコシステムについて、マニフェストの
// 読み書きとソーススキャンを実装する。
package ecosystem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
)

// NPM は package.json の依存関係を扱う。
type NPM struct{}

func (NPM) ID() model.Ecosystem {
	return model.EcosystemNPM
}

func (NPM) Detect(ctx context.Context, src gateway.ManifestSource) (bool, error) {
	return src.Exists(ctx, "package.json")
}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func (NPM) Scan(ctx context.Context, src gateway.ManifestSource) ([]gateway.Dependency, error) {
	data, err := src.ReadFile(ctx, "package.json")
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

// ApplyUpdate は対象を絞った置換でバージョンを書き換えるため、ファイルの
// フォーマットとキーの順序が保たれる。
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
		// 置換が JSON を壊していないことを書き込み前に検証する（同名キーが他箇所に
		// あった場合などの破損を防ぐ安全弁）。
		if !json.Valid(updated) {
			return nil, fmt.Errorf("npm: updating %s would produce invalid package.json", name)
		}
		if err := os.WriteFile(pkgPath, updated, 0o644); err != nil {
			return nil, err
		}
		changed = append(changed, "package.json")
	}
	if fileExists(filepath.Join(repoPath, "package-lock.json")) {
		// 実モードでは、ここで `npm install` がロックファイルを再生成する。
		changed = append(changed, "package-lock.json")
	}
	return changed, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
