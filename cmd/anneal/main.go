// Command anneal は自己改善型の依存関係更新エージェントの CLI です。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/ZONO33LHD/anneal/domain/config"
	"github.com/ZONO33LHD/anneal/domain/ctxkey"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/policy"
	"github.com/ZONO33LHD/anneal/infrastructure/auth"
	applog "github.com/ZONO33LHD/anneal/infrastructure/log"
	webhookhttp "github.com/ZONO33LHD/anneal/interface/http"
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
	applog.New(opts).Error(context.Background(), "command failed", err)
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
	case "serve":
		return cmdServe()
	case "improve":
		return cmdImprove()
	case "approve":
		return cmdApprove(args[1:])
	case "adopt":
		return cmdAdopt()
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
  anneal serve                             Listen for GitHub webhooks over HTTP
  anneal improve                           Run the Annealing Loop score check
  anneal approve <improvementID>           Approve a candidate so adopt can promote it to canary
  anneal adopt                             Advance A/B adoption (canary → adopt/rollback)
  anneal status                            Print a summary of records and scores
  anneal demo                              Run the full lifecycle on the bundled fixture (all-mock)
`)
}

// runContext は 1 コマンド実行ぶんを相関させるトレース ID を載せた context を返す。
func runContext() context.Context {
	ctx, _ := ctxkey.EnsureTraceID(context.Background())
	return ctx
}

func newRegistry() (*registry.Registry, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return registry.New(cfg)
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
	defer func() { _ = reg.Close() }()
	res, err := reg.Scan.Run(runContext(), fs.Arg(0), *repo)
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
	defer func() { _ = reg.Close() }()
	ctx := runContext()
	changed, err := reg.Engine.Tick(ctx)
	if err != nil {
		return err
	}
	if err := reg.Anneal.MaybeAnneal(ctx); err != nil {
		return err
	}
	// 改善候補があれば A/B 採用ループも 1 ステップ進める。
	if err := reg.Adoption.Evaluate(ctx); err != nil {
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
	defer func() { _ = reg.Close() }()
	ctx := runContext()
	if _, err := reg.Engine.Reconcile(ctx); err != nil {
		return err
	}
	return reg.Anneal.MaybeAnneal(ctx)
}

func cmdServe() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	reg, err := registry.New(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = reg.Close() }()
	mux := http.NewServeMux()
	mux.Handle("/webhooks/github", webhookhttp.NewGitHubWebhookHandler(webhookhttp.GitHubWebhookOptions{
		Secret:  cfg.GitHubWebhookSecret,
		Webhook: reg.Webhook,
		Logger:  reg.Logger,
	}))
	var internalAuth webhookhttp.InternalTaskAuthenticator
	if cfg.InternalOIDCAudience != "" {
		internalAuth = auth.NewOIDCValidator(cfg.InternalOIDCAudience, cfg.InternalOIDCEmail)
	}
	internalTasks := webhookhttp.NewInternalTaskHandler(webhookhttp.InternalTaskOptions{
		Token:         cfg.InternalToken,
		Authenticator: internalAuth,
		ScanTargets:   cfg.ScanTargets,
		Scan:          reg.Scan,
		Engine:        reg.Engine,
		Anneal:        reg.Anneal,
		Adoption:      reg.Adoption,
		Logger:        reg.Logger,
	})
	mux.Handle("/internal/scan", internalTasks)
	mux.Handle("/internal/tick", internalTasks)
	mux.Handle("/", webhookhttp.NewDashboardHandler(webhookhttp.DashboardOptions{
		Dashboard: reg.Dashboard,
		Logger:    reg.Logger,
	}))

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx := runContext()
	signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ListenAndServe はブロックするため別 goroutine で動かし、その終了結果を
	// チャネルで main 側へ渡す。これにより (1) bind 失敗などの起動時エラーを
	// select で即座に検知して返せる（チャネルが無いと signalCtx.Done() を
	// 永遠に待ち続けてしまう）、(2) shutdown 後に ListenAndServe が実際に
	// 抜けるのを待ってから戻れる。バッファ 1 は、main がエラー経路で先に抜けても
	// goroutine の送信がブロックせず leak しないようにするため。
	errCh := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	reg.Logger.Info(ctx, "http server started", "addr", cfg.HTTPAddr)

	// 起動時エラーなら即返し、シグナルを受けたら shutdown へ進む。
	select {
	case err := <-errCh:
		return err
	case <-signalCtx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	reg.Logger.Info(ctx, "http server shutting down")
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return <-errCh
}

func cmdImprove() error {
	reg, err := newRegistry()
	if err != nil {
		return err
	}
	defer func() { _ = reg.Close() }()
	return reg.Anneal.MaybeAnneal(runContext())
}

// cmdApprove は候補（candidate）を人間承認し approved にする。生成された改善版の
// プロンプト文言を本番の判断に載せる前の必須ゲート。承認後、次の adopt で canary へ
// 昇格される。
func cmdApprove(args []string) error {
	fs := flag.NewFlagSet("approve", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	id := fs.Arg(0)
	if id == "" {
		return fmt.Errorf("usage: anneal approve <improvementID>")
	}
	reg, err := newRegistry()
	if err != nil {
		return err
	}
	defer func() { _ = reg.Close() }()
	return reg.Adoption.Approve(runContext(), id)
}

// cmdAdopt は A/B 採用ループを 1 ステップ進める（承認済み候補の canary 昇格、または
// canary の採用/巻き戻し）。
func cmdAdopt() error {
	reg, err := newRegistry()
	if err != nil {
		return err
	}
	defer func() { _ = reg.Close() }()
	return reg.Adoption.Evaluate(runContext())
}

func cmdStatus() error {
	reg, err := newRegistry()
	if err != nil {
		return err
	}
	defer func() { _ = reg.Close() }()
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
	reg, err := registry.New(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = reg.Close() }()
	ctx := runContext()

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

	// 生成された候補を承認する（本番では人間が `anneal approve` を実行する承認ゲート。
	// デモは all-mock なので自動で通す）。承認しないと candidate は canary へ昇格しない。
	fmt.Println("\n=== 5) 🙋 Approve candidate (human gate) ===")
	if imps, err := reg.Improvements.ListImprovements(); err == nil {
		for _, imp := range imps {
			if imp.Status == model.ImprovementCandidate {
				if err := reg.Adoption.Approve(ctx, imp.ImprovementID); err != nil {
					return err
				}
			}
		}
	}

	fmt.Println("\n=== 6) A/B adoption (approved → canary) ===")
	if err := reg.Adoption.Evaluate(ctx); err != nil {
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
