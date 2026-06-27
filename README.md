# Anneal 🔥

> **Refine your dependencies. Refine yourself.**

Anneal は、依存ライブラリ・脆弱性のアップグレードを AI が自律的に回し、その AI の振る舞い自体もスコアリングして自己改善し続ける **自己改善型 Dependency / Security 更新エージェント**です。

要件定義書: [`docs/spec/requirements.md`](docs/spec/requirements.md)

このリポジトリは要件定義 v2 の **MVP + Should** スコープを TypeScript で実装したものです。

- 対象エコシステム: **npm (`package.json`)** / **Go (`go.mod`)**
- 外部サービス (Gemini / GitHub / Slack) は**インターフェースで抽象化**。API キーが無ければ自動でモックにフォールバックし、**鍵ゼロで E2E が通る**。`.env` に鍵を入れると実サービスへ切替。
- 永続ストアはローカル JSON（Firestore へ差し替え可能な `Store` インターフェース）。

## クイックスタート

```bash
npm install
npm test          # 47 tests
npm run demo      # フルライフサイクルをワンコマンドで再現
```

## デモの流れ（17章のストーリー）

`npm run demo` は `fixtures/sample-repo`（脆弱・旧版依存を仕込んだサンプル）を隔離ディレクトリにコピーし、全モックで以下を実行します。

1. **検知 → PR 自動作成** — lodash の CVE、axios/express の minor、chalk の major などを検知し、影響分析付き PR を生成（Slack 相当の通知をコンソールへ）。
2. **CI 失敗 → 自己修復** — minor 更新で CI が失敗 → AI が失敗を分類 → 追加コミットで修復 → CI 再実行で成功。
3. **🔥 Annealing Loop** — 確定スコアが基準を下回ると失敗事例を蓄積し、改善仮説とプロンプト改善案を生成。

major 更新 (chalk) や認証系 (golang.org/x/crypto) は **人間承認ゲート**へ回されます（10章）。

## CLI

| コマンド | 説明 | 対応トリガー |
|---|---|---|
| `anneal scan <repoPath> [-r owner/repo]` | リポジトリを走査し `detected` レコードを冪等生成 | T1 / T2 |
| `anneal tick` | アクティブな全レコードを1ステップ進める | T1 (Scheduler単位) |
| `anneal reconcile` | 取りこぼしレコードを実状態へ追従 | T9 |
| `anneal improve` | スコア低下チェック → Annealing Loop | T10 |
| `anneal status` | レコードとスコアの一覧 | — |
| `anneal demo` | バンドル fixture でフルフロー（全モック） | — |

実運用イメージ（ローカル）:

```bash
anneal scan ./path/to/your/repo -r your-org/your-repo
anneal tick      # Cloud Scheduler 相当を繰り返し呼ぶ
anneal status
```

## 実サービス接続

`.env`（`.env.example` 参照）に鍵を設定すると、その能力だけ実装へ切り替わります。

| 変数 | 無し | 有り |
|---|---|---|
| `GEMINI_API_KEY` | MockLlm | Gemini (`gemini-2.5-flash-lite`) |
| `GITHUB_TOKEN` | MockGitProvider | GitHub (Octokit) |
| `SLACK_WEBHOOK_URL` | ConsoleNotifier | Slack Incoming Webhook |

`GEMINI_API_KEY` か `GITHUB_TOKEN` のいずれかがある場合、メタデータ取得は OSV.dev + npm/Go レジストリ（`fetch` のみ）に切替わります。

## アーキテクチャ

設計の背骨は要件書 **7章（状態機械）・8章（トリガー）・6章（イベント駆動 + 定期照合）**。
どのトリガーも「対象レコードを読み → 状態機械を1ステップ進め → 書き戻す」という**単一規律**に収束します。

```
src/
├── domain/        状態機械(states) と集約型 (update_key で冪等)
├── store/         Store インターフェース + JSON 永続実装
├── ecosystems/    npm / go のマニフェスト解析・更新適用
├── providers/     llm / git / notify / metadata（実装 + モック）
├── agent/         分類・影響分析・判断ルール・PR生成・CI失敗分類
├── scoring/       重み付け総合スコア + 段階確定 (partial→final)
├── improvement/   失敗事例蓄積 + Annealing Loop
└── orchestrator/  状態機械・パイプライン・トリガー・tick/reconcile
```

### 主要な設計判断

- **冪等性 / PR 乱立防止 (NF-008/NF-021)**: `update_key = repository + package + target_version`。同一キーのアクティブレコードがあれば新規作成しない。
- **段階スコアリング (9.5)**: 評価レコードは `partial` で作られ、CI → レビュー → マージ → 回帰の入力到着で更新、マージ確定で `final`。
- **最安モデル前提の品質担保 (6.1)**: 影響分析・判断は決定論的ルールが土台、LLM は要約/仮説生成の補助。低品質・高リスクは必ず人間承認ゲート（10章）で受ける。

## テスト

```bash
npm test          # unit + integration + e2e (47 tests)
npm run coverage  # カバレッジ付き
npm run typecheck # tsc --noEmit
```

- **unit**: semver / scorer / 状態機械 / 分類 / 判断ルール / update_key
- **integration**: JSON ストア（永続・active判定）/ npm・go パーサ
- **e2e**: scan → drive のフル通し（検知・冪等・自己修復・人間承認・Annealing 発火）

## スコープ外（今回未実装 = Could / 将来）

- A/B 評価による改善版の自動採用・カナリア・ロールバック（F-043〜F-048, F-057/F-058）
- ダッシュボード UI
- GCP 本番 IaC（Cloud Run / Pub/Sub / Firestore）。`Store` ほか各インターフェースは差し替え前提で設計済み。
