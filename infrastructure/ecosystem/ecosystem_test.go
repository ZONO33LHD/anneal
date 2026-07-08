package ecosystem_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/infrastructure/ecosystem"
	"github.com/ZONO33LHD/anneal/internal/testutil"
)

func names(d []gateway.Dependency) []string {
	out := make([]string, len(d))
	for i, x := range d {
		out[i] = x.Name
	}
	return out
}

func TestNPMScanAndApply(t *testing.T) {
	work := testutil.CopyFixture(t, filepath.Join("..", ".."))
	deps, err := ecosystem.NPM{}.Scan(context.Background(), ecosystem.NewLocalFS(work))
	if err != nil {
		t.Fatal(err)
	}
	n := names(deps)
	if !slices.Contains(n, "lodash") || !slices.Contains(n, "axios") {
		t.Errorf("missing deps: %v", n)
	}
	for _, d := range deps {
		if d.Name == "typescript" && !d.IsDev {
			t.Error("typescript should be dev")
		}
	}
	changed, err := ecosystem.NPM{}.ApplyUpdate(work, "axios", "1.7.0")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(changed, "package.json") {
		t.Errorf("expected package.json changed: %v", changed)
	}
	data, _ := os.ReadFile(filepath.Join(work, "package.json"))
	if !strings.Contains(string(data), `"axios": "^1.7.0"`) {
		t.Errorf("axios not bumped:\n%s", data)
	}
}

func TestGoScanAndApply(t *testing.T) {
	work := testutil.CopyFixture(t, filepath.Join("..", ".."))
	deps, err := ecosystem.GoMod{}.Scan(context.Background(), ecosystem.NewLocalFS(work))
	if err != nil {
		t.Fatal(err)
	}
	n := names(deps)
	if !slices.Contains(n, "github.com/gin-gonic/gin") || !slices.Contains(n, "golang.org/x/crypto") {
		t.Errorf("missing go deps: %v", n)
	}
	changed, err := ecosystem.GoMod{}.ApplyUpdate(work, "github.com/gin-gonic/gin", "v1.9.1")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(changed, "go.mod") {
		t.Errorf("expected go.mod changed: %v", changed)
	}
	data, _ := os.ReadFile(filepath.Join(work, "go.mod"))
	if !strings.Contains(string(data), "gin v1.9.1") {
		t.Errorf("gin not bumped:\n%s", data)
	}
}

func TestProviderDetect(t *testing.T) {
	work := testutil.CopyFixture(t, filepath.Join("..", ".."))
	found, err := ecosystem.NewProvider().ForRepo(context.Background(), ecosystem.NewLocalFS(work))
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, e := range found {
		ids = append(ids, string(e.ID()))
	}
	slices.Sort(ids)
	if len(ids) != 2 || ids[0] != "go" || ids[1] != "npm" {
		t.Errorf("expected [go npm], got %v", ids)
	}
}

func TestScannerUsageSites(t *testing.T) {
	work := testutil.CopyFixture(t, filepath.Join("..", ".."))
	sites, err := ecosystem.NewScanner().UsageSites(work, "axios")
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) == 0 {
		t.Error("expected axios usage sites")
	}
}
