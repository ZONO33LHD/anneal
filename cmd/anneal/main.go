// Command anneal is the CLI for the self-improving dependency update agent.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/ZONO33LHD/anneal/domain/config"
	"github.com/ZONO33LHD/anneal/domain/policy"
	"github.com/ZONO33LHD/anneal/registry"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "✖ "+err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}
	switch args[0] {
	case "scan":
		return cmdScan(args[1:])
	case "tick":
		return cmdTick()
	case "reconcile":
		return cmdReconcile()
	case "improve":
		return cmdImprove()
	case "status":
		return cmdStatus()
	case "demo":
		return runDemo()
	case "-h", "--help", "help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func usage() {
	fmt.Print(`anneal — self-improving dependency & security update agent

Usage:
  anneal scan <repoPath> [-r owner/repo]   Scan a repository and create detected records
  anneal tick                              Advance every active record one step
  anneal reconcile                         Catch up records left behind by missed events
  anneal improve                           Run the Annealing Loop score check
  anneal status                            Print a summary of records and scores
  anneal demo                              Run the full lifecycle on the bundled fixture (all-mock)
`)
}

func newRegistry() (*registry.Registry, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return registry.New(cfg), nil
}

func cmdScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	repo := fs.String("r", "", "logical repository name (owner/repo)")
	_ = fs.Parse(args)
	if fs.NArg() < 1 {
		return fmt.Errorf("scan requires <repoPath>")
	}
	reg, err := newRegistry()
	if err != nil {
		return err
	}
	res, err := reg.Scan.Run(context.Background(), fs.Arg(0), *repo)
	if err != nil {
		return err
	}
	fmt.Printf("ℹ scan complete created=%d skipped=%d\n", len(res.Created), res.Skipped)
	return nil
}

func cmdTick() error {
	reg, err := newRegistry()
	if err != nil {
		return err
	}
	ctx := context.Background()
	changed, err := reg.Engine.Tick(ctx)
	if err != nil {
		return err
	}
	if err := reg.Anneal.MaybeAnneal(ctx); err != nil {
		return err
	}
	fmt.Printf("ℹ tick complete advanced=%d\n", changed)
	return nil
}

func cmdReconcile() error {
	reg, err := newRegistry()
	if err != nil {
		return err
	}
	ctx := context.Background()
	if _, err := reg.Engine.Reconcile(ctx); err != nil {
		return err
	}
	return reg.Anneal.MaybeAnneal(ctx)
}

func cmdImprove() error {
	reg, err := newRegistry()
	if err != nil {
		return err
	}
	return reg.Anneal.MaybeAnneal(context.Background())
}

func cmdStatus() error {
	reg, err := newRegistry()
	if err != nil {
		return err
	}
	updates, err := reg.Updates.List()
	if err != nil {
		return err
	}
	evals, _ := reg.Evaluations.List()
	improvements, _ := reg.Improvements.ListImprovements()
	fmt.Printf("ℹ status updates=%d evaluations=%d improvements=%d\n", len(updates), len(evals), len(improvements))
	score := map[string]string{}
	for _, e := range evals {
		score[e.UpdateKey] = fmt.Sprintf("%s %.1f", e.ScoreStatus, e.TotalScore)
	}
	for _, u := range updates {
		line := fmt.Sprintf("  %-22s %s@%s", u.Status, u.PackageName, u.TargetVersion)
		if s, ok := score[u.UpdateKey]; ok {
			line += " — " + s
		}
		fmt.Println(line)
	}
	return nil
}

// runDemo copies the fixture to an isolated working dir and runs the whole mock
// pipeline: detection → PR → CI self-heal → scoring → Annealing Loop. It uses a
// raised low-score threshold so the loop reliably fires.
func runDemo() error {
	work, _ := filepath.Abs(".anneal/work/sample-repo")
	storePath, _ := filepath.Abs(".anneal/demo-store.json")
	_ = os.RemoveAll(work)
	_ = os.Remove(storePath)
	if err := copyDir("fixtures/sample-repo", work); err != nil {
		return err
	}

	cfg := &config.Config{
		StorePath:         storePath,
		ImproveWindow:     policy.DefaultImproveWindow,
		LowScoreThreshold: 90, // ensure the Annealing Loop has material
		GeminiModel:       "gemini-2.5-flash-lite",
		ForceMock:         true,
		Verbose:           true,
	}
	reg := registry.New(cfg)
	ctx := context.Background()

	fmt.Println("\n=== 1) Detect ===")
	res, err := reg.Scan.Run(ctx, work, "acme/sample-repo")
	if err != nil {
		return err
	}
	fmt.Printf("ℹ detected candidates=%d\n", len(res.Created))

	fmt.Println("\n=== 2) Drive lifecycle (analyze → PR → CI → self-heal → merge) ===")
	if err := reg.Engine.Drive(ctx, policy.MaxDriveRounds); err != nil {
		return err
	}

	fmt.Println("\n=== 3) Scores ===")
	evals, _ := reg.Evaluations.List()
	sort.Slice(evals, func(i, j int) bool { return evals[i].TotalScore > evals[j].TotalScore })
	for _, e := range evals {
		fmt.Printf("  %-30s %s total=%.1f\n", packageOf(e.UpdateKey), e.ScoreStatus, e.TotalScore)
	}

	fmt.Println("\n=== 4) 🔥 Annealing Loop ===")
	if err := reg.Anneal.MaybeAnneal(ctx); err != nil {
		return err
	}

	fmt.Println("\n=== Done. Store: " + storePath + " ===")
	return nil
}

// packageOf extracts the package segment from "repo::package::version".
func packageOf(key string) string {
	first := -1
	for i := 0; i+1 < len(key); i++ {
		if key[i] == ':' && key[i+1] == ':' {
			if first == -1 {
				first = i + 2
				i++
				continue
			}
			return key[first:i]
		}
	}
	return key
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
