package metadata

import (
	"testing"

	"github.com/ZONO33LHD/anneal/domain/model"
)

func TestEscapeGoModulePath(t *testing.T) {
	cases := map[string]string{
		"golang.org/x/crypto":        "golang.org/x/crypto",          // 小文字・スラッシュは保持
		"github.com/gin-gonic/gin":   "github.com/gin-gonic/gin",     // 変化なし
		"github.com/BurntSushi/toml": "github.com/!burnt!sushi/toml", // 大文字は !小文字
	}
	for in, want := range cases {
		if got := escapeGoModulePath(in); got != want {
			t.Errorf("escapeGoModulePath(%q)=%q want %q", in, got, want)
		}
	}
}

func TestPyPILatestURL(t *testing.T) {
	got := pypiLatestURL("django rest framework")
	want := "https://pypi.org/pypi/django%20rest%20framework/json"
	if got != want {
		t.Fatalf("pypiLatestURL=%q want %q", got, want)
	}
}

func TestOSVEcosystemName(t *testing.T) {
	cases := map[model.Ecosystem]string{
		model.EcosystemNPM:  "npm",
		model.EcosystemGo:   "Go",
		model.EcosystemPyPI: "PyPI",
	}
	for eco, want := range cases {
		if got := osvEcosystemName(eco); got != want {
			t.Fatalf("osvEcosystemName(%q)=%q want %q", eco, got, want)
		}
	}
}
