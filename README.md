# Anneal 🔥

> **Refine your dependencies. Refine yourself.**

Anneal は、依存ライブラリ・脆弱性のアップグレードを AI が自律的に回し、その AI の振る舞い自体もスコアリングして自己改善し続ける **自己改善型 Dependency / Security 更新エージェント**です。

要件定義書: [`docs/spec/requirements.md`](docs/spec/requirements.md)

- **実装言語: Go**（単一バイナリ配布・Cloud Run / CLI 向き）
- **更新対象: npm（`package.json`）/ Go（`go.mod`）**
- 設計: **クリーンアーキテクチャ**（`domain` → `usecase` → `infrastructure`、`registry` で DI）
- 外部サービス（Gemini / GitHub / Slack）は **domain のポート**として抽象化。API キーが無ければ自動でモックにフォールバックし、**鍵ゼロで E2E が通る**。`.env` に鍵を入れると実サービスへ切替。
- 永続ストアはローカル JSON / **Firestore** を `ANNEAL_STORE_BACKEND` で切替（`repository` ポート）。本番は **Cloud Run + Firestore** で稼働。

## クイックスタート

```bash
go build ./...
go test -race ./...        # 全テスト
go run ./cmd/anneal demo   # フルライフサイクルをワンコマンドで再現
```

## デモの流れ

`go run ./cmd/anneal demo` は `fixtures/sample-repo`（脆弱・旧版依存を仕込んだサンプル）を隔離ディレクトリにコピーし、全モックで以下を実行します。

1. **検知 → PR 自動作成** — lodash の CVE、axios/express の minor、chalk の major などを検知し、影響分析付き PR を生成（Slack 相当の通知をコンソールへ）。
2. **CI 失敗 → 自己修復** — minor 更新で CI が失敗 → AI が失敗を分類 → 追加コミットで修復 → CI 再実行で成功。
3. **🔥 Annealing Loop** — 確定スコアが基準を下回ると失敗事例を蓄積し、改善仮説とプロンプト改善案を生成。

major 更新 (chalk) や認証系 (golang.org/x/crypto) は **人間承認ゲート**へ回されます。

## 使い方（3 モード）

全体像: `scan` で検知 → `tick`（または Webhook）で状態機械が 1 ステップずつ進む（分析 → 承認ゲート → PR → CI → 自己修復 → マージ → スコア）→ 低スコアなら Annealing Loop が改善候補を生成 → `adopt` で A/B 採用判断 → ダッシュボードで可視化。

### 1. お試し（鍵ゼロ）

```bash
go run ./cmd/anneal demo
```

全モックで検知 → PR → CI 自己修復 → スコア → Annealing → A/B を一気に再現します（挙動理解用）。

### 2. ローカル実運用（実 API・ポーリング駆動）

`.env`（`.env.example` 参照）に鍵を設定し、定期実行（cron 等）で回します。

```bash
anneal scan /path/to/repo -r owner/repo   # 検知（detected レコードを冪等生成）
anneal tick                               # 状態を 1 ステップ進める（PR/CI/スコア/Annealing/A/B まで）
anneal reconcile                          # 取りこぼしレコードを実状態へ追従
anneal status                             # 現状確認
```

Webhook 無しでも「定期 `scan` + `tick`」で運用できます。

### 3. 本番（GCP / Cloud Run・イベント駆動）

1. Terraform で `api → artifactregistry → firestore → secretmanager → iam → cloudrun` を apply（[`terraform/README.md`](terraform/README.md)）し、secret 値を投入。
2. `main` への push で `image-build` ワークフローがイメージを Artifact Registry へ push し、Cloud Run へ deploy。
3. Cloud Run は `ANNEAL_STORE_BACKEND=firestore` で `serve` を起動。GitHub の Webhook を `https://<cloud-run-url>/webhooks/github` に設定（`check_suite` / `pull_request_review` / `pull_request`）すると、イベントで状態機械が進みます（署名は `GITHUB_WEBHOOK_SECRET` で検証）。
4. ステータスダッシュボードは Cloud Run の `/`（または `/dashboard`）で閲覧できます。

> 注: 定期 `scan` / `tick` の Cloud Scheduler 配線は未実装です。現状は Webhook 駆動 + 手動/外部からの `scan` 起動を想定しています。

## アーキテクチャ（クリーンアーキテクチャ）

依存方向は内向き一方向：`cmd → registry → usecase → domain`。`infrastructure` は domain のポート（interface）を実装し、`registry` で合成します。

```text
cmd/anneal/                CLI（delivery 層）
registry/                  DI コンテナ（実 or モックを鍵に応じて選択）
usecase/                   アプリケーション層（scan / engine / anneal）
domain/                    最内層（外部に依存しない）
  model/                   エンティティ＋状態機械（update_key で冪等）
  repository/              永続化ポート（interface）
  gateway/                 外部サービス・アダプタのポート（LLM/Git/Notifier/Metadata/Ecosystem/Logger）
  service/                 純粋なドメインロジック（分類・判断・リスク・スコア・CI失敗分類）
  policy/                  しきい値・重み・定数
  config/ errors/          設定ロード・アプリエラー
infrastructure/            ポートの実装（adapter）
  persistence/             JSON 永続（repository 実装、単一の真実）
  ecosystem/               npm / go マニフェスト解析・更新・ソース走査
  llm/ git/ notify/ metadata/   Gemini / GitHub / Slack / OSV+registry ＋各モック
  log/ repoconfig/         logger・.anneal.yml ローダ
```

### 設計の背骨

どのトリガーも「対象レコードを読み → 状態機械を1ステップ進め → 書き戻す」という**単一規律**（`usecase/engine.go` の `Dispatch`）に収束します。常駐プロセスを持たず、ステートレス実行に載せられます。

### 主要な設計判断

- **冪等性 / PR 乱立防止**: `update_key = repository + package + target_version`。同一キーのアクティブレコードがあれば新規作成しない。
- **段階スコアリング**: 評価レコードは `partial` で作られ、CI → レビュー → マージ → 回帰の入力到着で更新、マージ確定で `final`。
- **最安モデル前提の品質担保**: 影響分析・判断は `domain/service` の決定論的ルールが土台、LLM は要約/仮説生成の補助。低品質・高リスクは必ず人間承認ゲートで受ける。

## CLI

```text
anneal scan <repoPath> [-r owner/repo]   リポジトリを走査し detected レコードを冪等生成
anneal tick                              アクティブな全レコードを1ステップ進める（Annealing/A/B も実行）
anneal reconcile                         取りこぼしレコードを実状態へ追従
anneal serve                             GitHub Webhook を受けるHTTPサーバ＋ダッシュボードを起動
anneal improve                           スコア低下チェック → Annealing Loop
anneal adopt                             A/B 採用を1ステップ進める（canary → 採用/巻き戻し）
anneal status                            レコードとスコアの一覧
anneal demo                              バンドル fixture でフルフロー（全モック）
```

## 実サービス接続

`.env`（`.env.example` 参照）に鍵を設定すると、その能力だけ実装へ切り替わります（`registry` が選択）。

| 変数 | 無し | 有り |
|---|---|---|
| `GEMINI_API_KEY` | Mock LLM | Gemini (`gemini-2.5-flash-lite`) |
| `GITHUB_TOKEN` | Mock git | GitHub REST API（branch/commit/PR を Git data API で実作成） |
| `SLACK_WEBHOOK_URL` | Console 通知 | Slack Incoming Webhook |

`GEMINI_API_KEY` か `GITHUB_TOKEN` のいずれかがある場合、メタデータ取得は OSV.dev + npm レジストリ / Go module proxy に切替わります（すべて標準 `net/http`）。

その他の主な設定（[`.env.example`](.env.example)）:

| 変数 | 既定 | 用途 |
|---|---|---|
| `ANNEAL_STORE_BACKEND` | `json` | 永続化バックエンド（`json` / `firestore`） |
| `ANNEAL_FIRESTORE_PROJECT` | （`GOOGLE_CLOUD_PROJECT`） | Firestore のプロジェクト ID |
| `ANNEAL_HTTP_ADDR` / `PORT` | `:8080` | `serve` の待受アドレス（Cloud Run は `PORT` を注入） |
| `GITHUB_WEBHOOK_SECRET` | （必須・serve 時） | Webhook の HMAC-SHA256 署名検証 |
| `ANNEAL_LOG_FORMAT` | text | `json` で severity 付き構造化ログ（Cloud Logging 向け） |

## テスト

```bash
go test -race ./...
go vet ./...
```

- **unit (domain)**: semver / 状態機械 / update_key / 分類 / 判断ルール / スコアリング
- **integration (infrastructure)**: JSON 永続（永続・active判定）/ npm・go パーサ / ソース走査
- **e2e (usecase)**: scan → drive のフル通し（検知・冪等・自己修復・人間承認・Annealing 発火）

## 実装済みの主な機能

- 検知 → 影響分析 → 人間承認ゲート → PR 作成 → CI 自己修復 → 段階スコアリング
- **実モード GitHub**（Git data API で branch/commit/PR を実作成）
- **Webhook イベント駆動**（`serve`、署名検証・trace 伝播）
- **Firestore 永続化**（`ANNEAL_STORE_BACKEND` で切替）
- **A/B 採用・カナリア・ロールバック**（`adopt` / `tick`）
- **ステータスダッシュボード**（`serve` の `/`）
- **GCP 本番 IaC**（Terraform: API/Artifact Registry/Firestore/Secret/IAM/Cloud Run）＋ コンテナイメージの build/push/deploy（GitHub Actions）

## スコープ外（将来）

- Cloud Scheduler による定期 `scan` / `tick` の自動起動
- Python など npm / Go 以外のエコシステム
- Dependabot Alert Webhook による検知トリガー
- 採用版に応じたプロンプト本文の動的差し替え
