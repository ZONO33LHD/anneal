package ecosystem

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
)

// GoMod は go.mod の依存関係を扱う。
type GoMod struct{}

func (GoMod) ID() model.Ecosystem {
	return model.EcosystemGo
}

func (GoMod) Detect(ctx context.Context, src gateway.ManifestSource) (bool, error) {
	return src.Exists(ctx, "go.mod")
}

// requireLine は "  module/path v1.2.3  // indirect" にマッチする。
var requireLine = regexp.MustCompile(`^\s*([^\s]+)\s+(v\d[^\s]*)(\s*//\s*indirect)?\s*$`)

func (GoMod) Scan(ctx context.Context, src gateway.ManifestSource) ([]gateway.Dependency, error) {
	data, err := src.ReadFile(ctx, "go.mod")
	if err != nil {
		return nil, err
	}
	var out []gateway.Dependency
	inBlock := false
	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "require ("):
			inBlock = true
			continue
		case inBlock && trimmed == ")":
			inBlock = false
			continue
		}
		target := line
		if strings.HasPrefix(trimmed, "require ") && !strings.HasPrefix(trimmed, "require (") {
			target = strings.TrimPrefix(trimmed, "require ")
		} else if !inBlock {
			continue
		}
		if m := requireLine.FindStringSubmatch(target); m != nil {
			out = append(out, gateway.Dependency{
				Name:           m[1],
				CurrentVersion: m[2],
				IsDev:          m[3] != "", // indirect は dev 相当として扱う
				Ecosystem:      model.EcosystemGo,
			})
		}
	}
	return out, nil
}

// bumpGoMod は go.mod の内容に対し、name の require 行のバージョンを target に
// 書き換えた新しい内容と、置換が起きたかを返す（純粋関数・ディスク非依存）。
func bumpGoMod(data []byte, name, target string) ([]byte, bool) {
	if !strings.HasPrefix(target, "v") {
		target = "v" + target
	}
	updated := false
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		single := strings.TrimPrefix(strings.TrimSpace(line), "require ")
		if m := requireLine.FindStringSubmatch(line); m != nil && m[1] == name {
			lines[i] = strings.Replace(line, m[2], target, 1)
			updated = true
		} else if strings.HasPrefix(strings.TrimSpace(line), "require ") {
			if sm := requireLine.FindStringSubmatch(single); sm != nil && sm[1] == name {
				lines[i] = strings.Replace(line, sm[2], target, 1)
				updated = true
			}
		}
	}
	return []byte(strings.Join(lines, "\n")), updated
}

func (GoMod) ApplyUpdate(repoPath, name, target string) ([]string, error) {
	modPath := filepath.Join(repoPath, "go.mod")
	data, err := os.ReadFile(modPath)
	if err != nil {
		return nil, err
	}
	out, updated := bumpGoMod(data, name, target)
	var changed []string
	if updated {
		if err := os.WriteFile(modPath, out, 0o644); err != nil {
			return nil, err
		}
		changed = append(changed, "go.mod")
		if fileExists(filepath.Join(repoPath, "go.sum")) {
			changed = append(changed, "go.sum")
		}
	}
	return changed, nil
}

// ApplyUpdateContent は ManifestSource から go.mod を読み、メモリ上でバージョンを
// 更新して「相対パス→更新後の内容」を返す（ローカル checkout 不要）。
// go.sum は再生成にツールチェインが要るためここでは扱わない（go.mod のみ更新し、
// go.sum の整合は CI/自己修復に委ねる）。
func (GoMod) ApplyUpdateContent(ctx context.Context, src gateway.ManifestSource, name, target string) (map[string][]byte, error) {
	data, err := src.ReadFile(ctx, "go.mod")
	if err != nil {
		return nil, err
	}
	out, updated := bumpGoMod(data, name, target)
	if !updated {
		return map[string][]byte{}, nil
	}
	return map[string][]byte{"go.mod": out}, nil
}
