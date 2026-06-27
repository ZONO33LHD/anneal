package ecosystem

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
)

// GoMod handles go.mod dependencies.
type GoMod struct{}

func (GoMod) ID() model.Ecosystem {
	return model.EcosystemGo
}

func (GoMod) Detect(repoPath string) bool {
	return fileExists(filepath.Join(repoPath, "go.mod"))
}

// requireLine matches "  module/path v1.2.3  // indirect".
var requireLine = regexp.MustCompile(`^\s*([^\s]+)\s+(v\d[^\s]*)(\s*//\s*indirect)?\s*$`)

func (GoMod) Scan(repoPath string) ([]gateway.Dependency, error) {
	data, err := os.ReadFile(filepath.Join(repoPath, "go.mod"))
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
				IsDev:          m[3] != "", // indirect treated as dev-like
				Ecosystem:      model.EcosystemGo,
			})
		}
	}
	return out, nil
}

func (GoMod) ApplyUpdate(repoPath, name, target string) ([]string, error) {
	modPath := filepath.Join(repoPath, "go.mod")
	data, err := os.ReadFile(modPath)
	if err != nil {
		return nil, err
	}
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
	var changed []string
	if updated {
		if err := os.WriteFile(modPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			return nil, err
		}
		changed = append(changed, "go.mod")
		if fileExists(filepath.Join(repoPath, "go.sum")) {
			changed = append(changed, "go.sum")
		}
	}
	return changed, nil
}
