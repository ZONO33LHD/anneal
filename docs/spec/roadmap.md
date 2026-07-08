# Anneal — 残タスク PR ロードマップ（T7〜T11）

> 本書は [`requirements.md`](./requirements.md) の「15. MVP スコープ」到達後に残る **将来スコープ**を、レビュワー負担の小さい stacked PR 列に落とし込んだもの。
> 進捗管理は Git のマイルストーンブランチ `feat/tN-*` で行う（T1〜T6 は実装済み、T4 は欠番）。

## 現在地

| 区分 | 内容 |
|---|---|
| 済み | T1 実モード GitHub / T2 Webhook HTTP / T3 Firestore / T5 A/B 採用 / T6 ダッシュボード（T4 欠番）。`usecase/engine.go` の `Dispatch`、CLI サブコマンド（`scan`/`tick`/`serve`/`adopt`/`reconcile`/`improve`/`status`/`demo`）、`serve` の GitHub Webhook（`check_suite`/`pull_request_review`/`pull_request`）、Firestore 切替、Terraform（`google_cloud_run_v2_service`）+ GitHub Actions（build/push/deploy）|
| 前提パターン | エコシステムは `gateway.Ecosystem` ポート + `infrastructure/ecosystem/provider.go`（`NewProvider` に `NPM{}, GoMod{}` 直書き）。メタデータは `infrastructure/metadata/osv.go`（`switch eco`）。LLM は `gateway.LLM` ポート、プロンプト本文は `usecase/engine_analysis.go`・`usecase/anneal.go` に**ハードコード文字列**。|

### ⚠️ 最重要の設計判断点：採用版プロンプト差替が現状 no-op

`AgentVersion` は record/eval に刻印されるが、**プロンプト本文へは一切反映されていない**。`AgentImprovement.ProposedChange` は生成・保存されるだけで未適用。A/B 採用ループの「使うほど賢くなる」が実質効いていない状態。これが本ロードマップ最大の関心事（機能4 / T10）。

---

## 残タスク（要件定義書のトリガー対応）

| 機能 | 内容 | 要件 | 状態 |
|---|---|---|---|
| T7 | 定期 `scan` / `tick` の自動起動（internal endpoint + Cloud Scheduler） | T1 / T9 | ✅ 完了（main マージ済み） |
| T8 | Dependabot Alert Webhook による検知トリガー | T2 / F-002 / F-019 | ✅ 完了（main マージ済み） |
| T9 | npm / Go 以外のエコシステム（Python 等）対応 | F-003 | 🧊 凍結（Issue #33・MVP スコープ外） |
| T10 | 採用版に応じたプロンプト本文の動的差替 | F-056 系 | ✅ 完了（4a/4b/C-1/C-2 マージ済み） |
| **T11** | **対象 repo 起点 push モデル（GitOps ワークフロー）＋リモート scanner** | NF-012/013・§14 | 🔜 計画（本書下部に詳細） |

> **進捗サマリ**：T7・T8・T10 は main マージ済み。T9 は凍結。以下の T7〜T10 の細分化表は着手前の計画で、T7/T8 は実際には codex が別粒度で実装した。**新規計画は末尾の T11**（対象 repo に置くワークフローから Anneal を起動する push モデル）。

---

## 細分化 PR 一覧（合計 8 本・手動 diff ~620 行）

各機能内は「a=契約/下位、b=利用側」の 2 本 stack。**機能1〜4 は相互独立で並行着手可**。各 PR は単独でビルド/テスト可能。

### T7: 定期起動

| # | PR タイトル | scope | 規模 | この PR マージ時点の挙動 | 依存 |
|---|---|---|---|---|---|
| 1a | `[anneal]` scan/tick を起動する internal HTTP エンドポイントを追加 | anneal | ~110行 | `serve` が cron 起動用エンドポイントに応答（手動 curl で scan/tick 起動可）| — |
| 1b | `[terraform]` Cloud Scheduler で scan/tick を定期起動 | terraform | ~90行 | 本番で定期 scan/tick が自動起動 | 1a |

- 1a: `interface/http` に `/internal/scan`・`/internal/tick` handler / registry・serve mux 配線 / OIDC or 内部トークン検証
- 1b: `google_cloud_scheduler_job`×2 / 起動用 SA + `run.invoker` IAM / env（`ANNEAL_SCAN_TARGETS`）/ `cloudscheduler.googleapis.com` 有効化

### T8: Dependabot Alert

| # | PR タイトル | scope | 規模 | この PR マージ時点の挙動 | 依存 |
|---|---|---|---|---|---|
| 2a | `[anneal]` webhook 契約に alert 起点シグナルを追加 | anneal | ~70行 | 型は増えるが挙動変化なし（既存パス不変・後方互換）| — |
| 2b | `[anneal]` dependabot_alert の HTTP parser を追加 | anneal | ~90行 | alert webhook 受信で該当 record を突合・遷移 | 2a |

- 2a: `usecase/webhook.go` に alert 用 optional フィールド/シグナル定数追加（nil 可）/ 対応付け方針の関数骨格
- 2b: `interface/http/github_webhook.go` の `parseGitHubEvent` switch に `dependabot_alert` / payload 構造体 / usecase 結線

### T9: Python 対応

| # | PR タイトル | scope | 規模 | この PR マージ時点の挙動 | 依存 |
|---|---|---|---|---|---|
| 3a | `[anneal]` EcosystemPyPI 定数と OSV の PyPI 分岐を追加 | anneal | ~60行 | model/metadata は PyPI を扱えるが呼び出し元なし | — |
| 3b | `[anneal]` Python ecosystem スキャナを実装し provider 登録 | anneal | ~100行 | Python repo を scan で検出（垂直スライス 1 マニフェスト種別）| 3a |

- 3a: `domain/model` に `EcosystemPyPI` / `osv.go` の `LatestVersion`・`Advisories` に PyPI 分岐（latest = PyPI JSON API、advisory ecosystem="PyPI"）
- 3b: `infrastructure/ecosystem/python.go`（Detect/Scan/ApplyUpdate）/ `provider.go` の `all` に追加

### T10: プロンプト版差替

| # | PR タイトル | scope | 規模 | この PR マージ時点の挙動 | 依存 |
|---|---|---|---|---|---|
| 4a | `[anneal]` PromptProvider ポートとカタログ実装を追加 | anneal | ~110行 | ポート追加のみ・既存プロンプトは同一文言（挙動不変）| — |
| 4b | `[anneal]` LLM 呼び出しを PromptProvider 経由に差替 | anneal | ~90行 | 採用/canary 版があればその版のプロンプト文で LLM 実行 | 4a |

- 4a: `domain/gateway` に `PromptProvider`（version, key → template）/ infra カタログ実装（`prompt_v1` = 現行文言逐語コピー）/ registry 配線 / golden test で差分ゼロ担保
- 4b: 判断プロンプト（`engine_analysis.go` のリスク/影響分析 1 箇所）の `llm.Generate` を `PromptProvider.For(rec.AgentVersion, key)` 経由に置換

#### 設計方針（dig で確定）

段階移行方式：**まず A（版で引ける配線）を安全に作り、自律改善は C へ拡張**する。B（LLM 生成文の即時直挿し）は非決定・インジェクション risk のため採らない。

| # | 論点 | 決定 |
|---|---|---|
| 1 | プロンプト差替の作り方 | **A → 段階的に C**。B は不採用 |
| 2 | 「版 → プロンプト」の表現 | **A''：`{Base, Appendix}` 構造体**。A では全版 `Appendix=""`、C では `Appendix = ProposedChange` を詰めるだけ。ポート契約 `For(version, key) → string` は A/C 共通で不変（内部で `Base+Appendix` を結合） |
| 3 | 版管理する対象プロンプト | **判断プロンプト 1 本のみ**（`engine_analysis.go` のリスク/影響分析）。改善ループの `ProposedChange` が狙う対象と一致。ci-summary / pr-body / メタプロンプト（hypothesis 等）は対象外。拡張は `key` 追加で対応 |
| 4 | カタログに無い版のフォールバック | **A1：既定版 `prompt_v1` にフォールバック**。パイプラインは止めない |
| 5 | A / C の線引き | **A 単体** = 版差し替えの配線を通し「手書きカタログ版なら効く」まで。ループが自動生成した版が実際に別プロンプトになるのは **C から**。デモの「78.2→84.7」は A の時点でも**手書き `prompt_v2` を 1 本カタログに仕込めば実演可能** |

> ⚠️ **A 単体は生成版に関しては no-op のまま**（未知版は `prompt_v1` にフォールバックするため）。自律的な「使うほど賢くなる」は C（`Appendix = ProposedChange` の適用）で初めて成立する。C 着手時にサニタイズ／承認フローを別途設計する（懸念 #5）。

---

## 依存グラフ

```
T7:  1a ──▶ 1b
T8:  2a ──▶ 2b
T9:  3a ──▶ 3b
T10: 4a ──▶ 4b
（T7/T8/T9/T10 は相互独立・並行可）
```

---

## 懸念点と対策

| # | 懸念点 | 該当 PR | 影響度 | 対策 |
|---|---|---|---|---|
| 1 | 定期起動を HTTP endpoint にすると未認証 invoke で外部から scan/tick 実行される | 1a/1b | 🔴高 | OIDC(`run.invoker`) or 内部トークン必須。public IAM member とは別 path |
| 2 | scan 対象 repo リストの供給元が未定（env 列挙 か Firestore 管理か） | 1b | 🟡中 | env 一括を一次対応、動的管理は別トラック |
| 3 | tick と webhook 駆動の同時実行で同一 record を二重 Dispatch（競合） | 1b | 🟡中 | 既存 `Dispatch` の冪等性を確認。Firestore なら楽観ロック要否を検証 |
| 4 | `dependabot_alert` は PR 起点でないため既存突合（repo+PR/branch）が効かない | 2a/2b | 🔴高 | alert→detected 新規生成か既存突合かを先に確定 |
| 5 | 採用版プロンプト差替の「LLM 生成 ProposedChange 直挿し」は非決定・インジェクション risk | 4a | 🔴高 | 一次は事前定義カタログ選択のみ（生成文の直挿しはしない） |
| 6 | PyPI の ApplyUpdate（lockfile 再生成）は pip resolve が必要で副作用大 | 3b | 🟡中 | 一次は manifest 直書換のみ、lock 再生成はスコープ外明記 |
| 7 | プロンプト差替後、`prompt_v1` の文言が 1 文字でもズレると全 record のスコア基準が変わる | 4a | 🟡中 | カタログの `prompt_v1` は現行文言を逐語コピー。golden test で差分ゼロ担保 |
| 8 | Cloud Scheduler 追加時の Terraform apply 順（API 有効化→SA→job） | 1b | 🔵低 | `api/main.tf` に `cloudscheduler.googleapis.com` 追加を同 PR に含める |

---

## 条件付き +1（PR 本数を左右する未確定要素）

| 調査項目 | 結果次第で増える PR |
|---|---|
| 定期起動を HTTP endpoint でなく **Cloud Run Job** にする場合 | 1a が「Job 用 CLI entrypoint 整備」に変わり、1b に `google_cloud_run_v2_job` が加わる（±0〜+1本） |
| scan 対象 repo を **Firestore で動的管理**する場合 | repo リスト CRUD の repository + usecase で **+1〜+2本** |
| Dependabot alert を **既存 record 突合でなく新規 detected 生成**にする場合 | 2b が 2 本に割れる **+1本** |
| プロンプト差替で **LLM 生成 ProposedChange の実適用**まで踏む場合 | サニタイズ/承認フロー + 適用の **+2本**（別トラック推奨） |
| Python の対象マニフェストが **複数種別**必要な場合 | 3b が種別ごとに **+1本** |

---

## スコープ外（別トラック）

- FE（ダッシュボード）への新トリガ表示・alert 由来 record の可視化
- 外形監視・SLO
- LLM 生成プロンプトの自動適用（安全性設計が別議論）
- PyPI lockfile の resolve/再生成、npm/Go 以外の追加エコシステム（Rust/Ruby 等）
- Cloud Scheduler の repo リスト管理 UI

---

## レビュー負担を下げる運用メモ

- 4機能は独立なので **4 本の stack を並行**で開ける。機能内は 2 本 stack（a=契約/下位、b=利用側）。
- 契約先行（2a・4a）を先に main へ入れれば、利用側 PR（2b・4b）は単一モジュールの極小差分になる。
- 生成物（`PromptProvider` の moq 等）は各 PR 内で別コミットに分離。
- 4a の `prompt_v1` は現行文言の逐語コピー + golden test で「挙動不変」を単独検証可能にする。

---

## 関連ファイル

- `cmd/anneal/main.go`
- `usecase/webhook.go`
- `interface/http/github_webhook.go`
- `infrastructure/ecosystem/provider.go`
- `infrastructure/metadata/osv.go`
- `usecase/engine_analysis.go`, `usecase/anneal.go`（プロンプトのハードコード箇所）
- `domain/model/improvement.go`, `domain/model/version.go`（agent_version の刻印元）
- `terraform/resources/anneal-prd/cloudrun/main.tf`, `terraform/resources/anneal-prd/api/main.tf`

---

# T11: 対象 repo 起点 push モデル（GitOps ワークフロー）＋リモート scanner

> 運用手順は [how-to/manage-projects.md](../how-to/manage-projects.md) を参照。

## ゴール

「どの GitHub プロジェクトを監視するか・除外・パラメータ」を、**エンジニアが対象 repo にコミットするだけ**で Anneal に管理させる。専用フロントも外部 SaaS も作らない。操作者はエンジニアのみ。

- 監視したい repo に `.github/workflows/anneal.yml` を置くと、その **CI（cron/手動）が Anneal（Cloud Run）の scan エンドポイントを認証付きで叩く**（push モデル）。
- Anneal 側に中央の監視対象一覧は持たない（**完全オプトイン**）。
- 除外・パラメータは対象 repo の `.anneal.yml`（既存 `RepoConfig`/`IsIgnored` を流用）。

## 設計判断（dig で確定）

| 論点 | 決定 |
|---|---|
| モデル | 対象 repo 起点の **push**（中央 pull＝旧案は破棄）。中央 `config/projects.yml`・`go:embed`・Cloud Scheduler-for-scan は**不要** |
| 起動 | 対象 repo の `anneal.yml` が `POST /internal/scan {repository:"owner/repo"}` を叩く。認証は既存 token/OIDC を流用 |
| scan の読み取り | Cloud Run にローカル checkout が無いため **GitHub Contents API でマニフェスト（package.json/go.mod＋lockfile）だけ取得**。clone しない。既存 scanner の `os.ReadFile` を `ManifestSource` ポートへ抽象化（LocalFS 実装で現挙動維持） |
| 責務分担 | 監視参加＝対象 repo の `anneal.yml`／除外・パラメータ＝対象 repo の `.anneal.yml`／組織デフォルト＝既存 `DefaultRepoConfig` |

## 細分化 PR 一覧（8 本・手動 ~620 行・自動生成なし）

| # | PR タイトル | scope | 規模 | この PR マージ時点の挙動 | 依存 |
|---|---|---|---|---|---|
| 1 | `[core]` `ManifestSource` ポート＋LocalFS 実装 | gateway, ecosystem | ~70 | port と LocalFS 追加のみ・既存未使用 | — |
| 2 | `[core]` Ecosystem を repoPath→ManifestSource 経由へ切替 | gateway, ecosystem, usecase, cmd | ~130 | LocalFS 注入で**挙動不変**・全 caller 統一 | 1 |
| 3 | `[core]` Contents API 版 ManifestSource | ecosystem/git | ~130 | リモート取得が単体で動作・未配線 | 1 |
| 4 | `[core]` `.anneal.yml` のリモート Loader | gateway, repoconfig | ~60 | remote でも ignore/override が効く・LocalFS 不変 | 1 |
| 5 | `[core]` owner/repo 起点の remote scan 経路（`RunRemote`） | usecase | ~90 | remote 1 件を scan し detected 作成可・endpoint 未接続 | 2,3,4 |
| 6 | `[interface]` `/internal/scan` に per-request `repository` 受け口 | interface/http | ~90 | 認証済みリクエストで自 repo scan 可・env 経路も温存 | 5 |
| 7 | `[registry]` remote source/loader を DI 配線 | registry | ~50 | 本番 Cloud Run で remote scan 有効 | 3,4,5 |
| 8 | `[docs]` 対象 repo 用ワークフローテンプレ＋how-to | docs | ~70 | 対象 repo がコピペで push scan 導入可 | 6 |

**stack**: #1 を土台に #2/#3/#4 を並行 → #5 で合流 → #6 →(#7,#8)。既存 env 全件ループ・LocalFS 経路・CLI・認証はすべて温存＝**加算的**（撤退は #4〜#7 revert で完結）。

## 依存グラフ

```
#1 manifest-source (port+LocalFS)
 ├─▶ #2 ecosystem 署名切替 ─┐
 ├─▶ #3 contents source ────┤
 └─▶ #4 remote repoconfig ──┤
                            ▼
                      #5 remote scan (RunRemote)
                            ▼
                      #6 endpoint repository 受け口
                            ├─▶ #7 registry 配線
                            └─▶ #8 workflow テンプレ+docs
```

## 懸念点と対策

| # | 懸念点 | 該当 | 影響度 | 対策 |
|---|---|---|---|---|
| 1 | Contents 用 GitHub token の private repo 読取権限不足で scan 失敗 | #7 | 🔴高 | `GITHUB_TOKEN` の scope を #7 前に確認、不足なら専用 Secret 追加 |
| 2 | #2 の `Ecosystem` 署名変更が cmd/usecase/テストに広く波及 | #2 | 🟡中 | 契約 PR として全 caller を 1 PR で更新。300 行超なら NPM/GoMod と provider/usecase に分割 |
| 3 | 対象 repo CI の認証方式（共有トークン Secret vs GitHub OIDC）未確定 | #8 | 🟡中 | 既存 token/OIDC 両対応を流用。決定までテンプレに両パターン併記 |
| 4 | Contents API のレート制限・大 lockfile 取得コスト（NF-020 段階実行と衝突） | #3,#5 | 🟡中 | manifest+lockfile のみ取得。dry-run で取得回数を計測 |
| 5 | cron 依存で取りこぼし（対象 repo が未設置/失敗）＝監視の穴 | 全体 | 🔵低 | 完全オプトイン前提を docs 明記。保険に Anneal 側 reconcile 併用（別トラック） |

## 条件付き +1（PR 本数を左右する未確定要素）

- 認証を OIDC 主体に決定 → 対象 repo 用 Workload Identity 設定 docs/terraform が +1
- Contents 用 token が private で権限不足 → 専用 Secret/env 追加 PR が #7 から分離して +1
- Contents API のレート/大 lockfile が段階実行に抵触 → ETag 条件取得キャッシュ +1

## スコープ外（別トラック）

- 監視対象の**一覧化**（完全オプトインの弱点補完＝ダッシュボード/検索）
- cron 取りこぼしの保険となる Anneal 側 reconcile スケジューラ
- 旧 pull 版（中央 `config/projects.yml`・`go:embed`・Cloud Scheduler-for-scan）＝**破棄・非対象**
- GitHub Issue 操作盤／自前フロント／外部 SaaS＝不採用

## トリガーの整理（重要）

T11 が担うのは **①検知（scan）のトリガー**のみ。Anneal 全体は 2 系統で駆動する:

- **① 検知**：対象 repo の CI cron → `/internal/scan`（PR 不要・時間駆動。PR は成果物であって入力ではない）
- **② 進行と自己改善**：Anneal が作った PR の CI/レビュー/マージ/回帰を **Webhook（`serve`）** で受けてスコア確定→ Annealing Loop。状態機械の進行（`tick`）の駆動は Anneal 側 scheduler / webhook が担う（T11 の対象外）

