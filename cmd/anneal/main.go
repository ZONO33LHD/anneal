// Command anneal は自己改善型の依存関係更新エージェントの CLI です。
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
	applog "github.com/ZONO33LHD/anneal/infrastructure/log"
	"github.com/ZONO33LHD/anneal/registry"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		// 最終的な失敗を severity=ERROR ＋スタックトレース付きで構造化記録する。
		logFatal(err)
		os.Exit(1)
	}
}

func logFatal(err error) {
	opts := applog.Options{}
	if cfg, cerr := config.Load(); cerr == nil {
		opts.JSON, opts.Verbose = cfg.LogJSON, cfg.Verbose
	}
	applog.New(opts).Error("command failed", err)
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
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	repo := fs.String("r", "", "logical repository name (owner/repo)")
	if err := fs.Parse(args); err != nil {
		return err
	}
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

// runDemo は fixture を隔離された作業ディレクトリにコピーし、モックのパイプライン全体を
// 実行します: detection → PR → CI self-heal → scoring → Annealing Loop。ループが確実に
// 発火するように、引き上げた low-score しきい値を使用します。
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
		LowScoreThreshold: 90, // Annealing Loop が処理対象を確実に持てるようにする
		GeminiModel:       "gemini-2.5-flash-lite",
		ForceMock:         true,
		Verbose:           true,
		LogJSON:           os.Getenv("ANNEAL_LOG_FORMAT") == "json",
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

// packageOf は "repo::package::version" から package のセグメントを抽出します。
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
