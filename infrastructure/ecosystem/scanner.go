package ecosystem

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/policy"
)

var skipDirs = map[string]bool{
	"node_modules": true, ".git": true, "dist": true, "vendor": true, ".anneal": true,
}

var textExt = regexp.MustCompile(`\.(ts|tsx|js|jsx|mjs|cjs|go|json)$`)

// Scanner は、リポジトリのソースファイル全体でパッケージがどこでインポートされているかを探す。
type Scanner struct{}

// NewScanner は SourceScanner を返す。
func NewScanner() gateway.SourceScanner {
	return Scanner{}
}

// UsageSites は、packageName を参照しているファイルの相対パスを返す。
func (Scanner) UsageSites(repoPath, packageName string) []string {
	if repoPath == "" {
		return nil
	}
	pattern := usagePattern(packageName)
	var sites []string
	count := 0
	_ = filepath.WalkDir(repoPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if count >= policy.MaxScanFiles {
			return filepath.SkipAll
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !textExt.MatchString(d.Name()) {
			return nil
		}
		count++
		info, err := d.Info()
		if err != nil || info.Size() > 512*1024 {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		if pattern.Match(data) {
			rel, _ := filepath.Rel(repoPath, p)
			sites = append(sites, rel)
		}
		return nil
	})
	return sites
}

func usagePattern(name string) *regexp.Regexp {
	q := regexp.QuoteMeta(name)
	return regexp.MustCompile(`(from\s+['"]` + q + `|require\(['"]` + q + `|['"]` + q + `)`)
}
