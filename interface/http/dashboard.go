package httpinterface

import (
	"html/template"
	"net/http"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/usecase"
)

// DashboardOptions は Dashboard ハンドラの依存を表す。
type DashboardOptions struct {
	Dashboard usecase.DashboardUsecase
	Logger    gateway.Logger
}

// NewDashboardHandler は状態分布・スコア・改善履歴を表示する HTML ダッシュボードを返す。
func NewDashboardHandler(opts DashboardOptions) http.Handler {
	return traceMiddleware(opts.Logger, &dashboardHandler{
		dashboard: opts.Dashboard,
		log:       opts.Logger,
	})
}

type dashboardHandler struct {
	dashboard usecase.DashboardUsecase
	log       gateway.Logger
}

func (h *dashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ルート直下の GET だけを扱い、未知パスは 404 にする。
	if r.URL.Path != "/" && r.URL.Path != "/dashboard" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	view, err := h.dashboard.Snapshot(r.Context())
	if err != nil {
		if h.log != nil {
			h.log.Error(r.Context(), "dashboard snapshot failed", err)
		}
		http.Error(w, "failed to build dashboard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dashboardTemplate.Execute(w, view); err != nil {
		// ヘッダ送信後の可能性があるためログのみ（クライアントへは追記しない）。
		if h.log != nil {
			h.log.Error(r.Context(), "dashboard render failed", err)
		}
	}
}

// dashboardTemplate は html/template による自動エスケープで XSS を防ぐ。デザインは
// トークン（CSS 変数）と意図的な階層で構成した内部向けの落ち着いた dark UI。
var dashboardTemplate = template.Must(template.New("dashboard").Funcs(template.FuncMap{
	"pct": func(count, total int) int {
		if total == 0 {
			return 0
		}
		return count * 100 / total
	},
}).Parse(`<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Anneal — status</title>
<style>
  :root {
    --bg: oklch(20% 0.02 260);
    --surface: oklch(26% 0.02 260);
    --line: oklch(36% 0.02 260);
    --text: oklch(96% 0 0);
    --muted: oklch(72% 0.02 260);
    --accent: oklch(72% 0.16 250);
    --good: oklch(72% 0.17 150);
    --warn: oklch(78% 0.16 80);
    --bad: oklch(68% 0.20 25);
    --space: clamp(1rem, 0.6rem + 1.5vw, 2rem);
    --radius: 12px;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0; background: var(--bg); color: var(--text);
    font: 15px/1.5 ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
  }
  header {
    padding: var(--space); border-bottom: 1px solid var(--line);
    display: flex; align-items: baseline; gap: 1rem; flex-wrap: wrap;
  }
  h1 { margin: 0; font-size: clamp(1.4rem, 1rem + 1.5vw, 2.2rem); letter-spacing: -0.02em; }
  .tag {
    font-size: 0.8rem; color: var(--muted);
    border: 1px solid var(--line); border-radius: 999px; padding: 0.15rem 0.7rem;
  }
  main { padding: var(--space); display: grid; gap: var(--space);
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); max-width: 1100px; }
  section {
    background: var(--surface); border: 1px solid var(--line);
    border-radius: var(--radius); padding: var(--space);
  }
  h2 { margin: 0 0 0.8rem; font-size: 0.95rem; text-transform: uppercase;
    letter-spacing: 0.08em; color: var(--muted); }
  .big { font-size: 2.6rem; font-weight: 700; letter-spacing: -0.03em; }
  .bar { height: 8px; border-radius: 999px; background: var(--accent); min-width: 2px; }
  .row { display: grid; grid-template-columns: 1fr auto; gap: 0.5rem; align-items: center;
    padding: 0.35rem 0; border-bottom: 1px solid var(--line); }
  .row:last-child { border-bottom: 0; }
  .muted { color: var(--muted); font-size: 0.85rem; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: 0.4rem 0.5rem; border-bottom: 1px solid var(--line); font-size: 0.9rem; }
  th { color: var(--muted); font-weight: 600; }
  .score { font-variant-numeric: tabular-nums; font-weight: 600; }
  .pill { font-size: 0.75rem; padding: 0.1rem 0.5rem; border-radius: 999px; border: 1px solid var(--line); }
  .pill.final { color: var(--good); border-color: var(--good); }
  .pill.partial { color: var(--warn); border-color: var(--warn); }
  .pill.adopted { color: var(--good); border-color: var(--good); }
  .pill.rolled_back { color: var(--bad); border-color: var(--bad); }
  .pill.canary { color: var(--warn); border-color: var(--warn); }
  .empty { color: var(--muted); font-style: italic; }
</style>
</head>
<body>
<header>
  <h1>🔥 Anneal</h1>
  <span class="tag">active version: {{.ActiveVersion}}</span>
  <span class="tag">updates: {{.TotalUpdates}}</span>
</header>
<main>
  <section aria-labelledby="avg-h">
    <h2 id="avg-h">確定スコア平均</h2>
    {{if .HasFinalScores}}
      <div class="big">{{printf "%.1f" .AverageScore}}</div>
      <div class="muted">final 評価の総合スコア平均</div>
    {{else}}
      <div class="empty">確定スコアはまだありません</div>
    {{end}}
  </section>

  <section aria-labelledby="states-h">
    <h2 id="states-h">状態分布</h2>
    {{if .StateCounts}}
      {{range .StateCounts}}
        <div class="row">
          <div>{{.State}}</div>
          <div class="score">{{.Count}}</div>
        </div>
        <div class="bar" style="width: {{pct .Count $.TotalUpdates}}%"></div>
      {{end}}
    {{else}}
      <div class="empty">レコードがありません</div>
    {{end}}
  </section>

  <section aria-labelledby="scores-h">
    <h2 id="scores-h">最近のスコア</h2>
    {{if .RecentScores}}
      <table>
        <thead><tr><th>package</th><th>version</th><th>status</th><th>score</th></tr></thead>
        <tbody>
        {{range .RecentScores}}
          <tr>
            <td>{{.Package}}</td>
            <td class="muted">{{.Version}}</td>
            <td><span class="pill {{.Status}}">{{.Status}}</span></td>
            <td class="score">{{printf "%.1f" .Total}}</td>
          </tr>
        {{end}}
        </tbody>
      </table>
    {{else}}
      <div class="empty">スコアはまだありません</div>
    {{end}}
  </section>

  <section aria-labelledby="imp-h">
    <h2 id="imp-h">改善履歴 (A/B)</h2>
    {{if .Improvements}}
      <table>
        <thead><tr><th>version</th><th>status</th><th>baseline</th><th>candidate</th></tr></thead>
        <tbody>
        {{range .Improvements}}
          <tr>
            <td>{{.Version}}</td>
            <td><span class="pill {{.Status}}">{{.Status}}</span></td>
            <td class="score">{{printf "%.1f" .Baseline}}</td>
            <td class="score">{{printf "%.1f" .Candidate}}</td>
          </tr>
        {{end}}
        </tbody>
      </table>
    {{else}}
      <div class="empty">改善候補はまだありません</div>
    {{end}}
  </section>
</main>
</body>
</html>`))
