package metadata

import "testing"

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
