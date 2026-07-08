# How-to: Anneal に対象プロジェクトを管理させる

> **ステータス**: この文書は **T11（対象 repo 起点 push モデル＋リモート scanner）** の設計に基づく運用ガイド。T11 は計画段階（[roadmap.md](../spec/roadmap.md) 参照）で、対象 repo 起点の scan トリガーとリモート scanner はまだ実装されていない。実装完了後の操作手順として先行して定める。

## 前提・設計方針

- **操作者はエンジニアのみ。** 専用フロント（GUI）や外部 SaaS は用意しない。
- **対象 repo 起点の push モデル。** 監視したい repo 側に設定ファイルを置くと、その repo の **CI（GitHub Actions）が Anneal（Cloud Run）を呼び出し**、Anneal が依存を分析して PR を作る。Anneal 側に「監視対象一覧」を持たない（中央設定ファイル不要）。
- 設定は **対象 repo に完結**する。

| 対象 repo に置くもの | 役割 | 必須 |
|---|---|---|
| `.github/workflows/anneal.yml` | CI から Anneal をトリガーするワークフロー | ✅ 必須（これが「参加表明」） |
| `.anneal.yml` | 除外パッケージ・更新パラメータなど repo 固有ポリシー | 任意（無ければ既定値） |
| リポジトリ Secret / OIDC | Anneal エンドポイントを呼ぶための認証情報 | ✅ 必須 |

> **Dependabot との違い**: Dependabot は GitHub 内部がスケジュールするが、Anneal は自前の Cloud Run サービスなので、対象 repo の Actions から明示的に呼び出す。「設定ファイルを置く → CI が Anneal と通信 → 処理」という流れは同じ。

---

## セットアップ（対象 repo を Anneal 管理下に置く）

### 1. ワークフローを置く

対象 repo に `.github/workflows/anneal.yml` を追加:

```yaml
name: anneal
on:
  schedule:
    - cron: "0 3 * * *"        # 毎日 1 回、依存をスキャン
  workflow_dispatch: {}         # 手動実行も可
jobs:
  trigger:
    runs-on: ubuntu-latest
    steps:
      - name: Trigger Anneal scan
        run: |
          curl -sS -X POST "${{ vars.ANNEAL_URL }}/internal/scan" \
            -H "X-Anneal-Internal-Token: ${{ secrets.ANNEAL_TOKEN }}" \
            -H "Content-Type: application/json" \
            -d "{\"repository\": \"${{ github.repository }}\"}"
```

- `ANNEAL_URL`（リポジトリ変数）= Anneal の Cloud Run URL
- `ANNEAL_TOKEN`（リポジトリ Secret）= エンドポイント認証トークン（または GitHub OIDC を使う）

### 2. （任意）ポリシーを置く

除外や更新方針を変えたい場合のみ、対象 repo のルートに `.anneal.yml`:

```yaml
base_branch: main
ignore:                         # 更新対象から除外
  - "react"                     #   完全一致
  - "@types/*"                  #   末尾 * で前方一致
auto_pr_types: [patch, minor]   # 自動 PR を作る更新種別
regression_window_days: 7       # マージ後の回帰監視ウィンドウ
```

無ければ組織デフォルトが適用される。

### 3. これで完了

以降、ワークフローの cron（または手動実行）ごとに CI が Anneal を叩き、Anneal が依存を分析して PR を作る。**Anneal 側の設定・再デプロイは不要。**

---

## 操作フロー

### A) 新しい repo を監視対象に追加する

1. その repo に `.github/workflows/anneal.yml` を追加（上記テンプレ）
2. リポジトリに `ANNEAL_URL` 変数と `ANNEAL_TOKEN` Secret を設定
3. PR → merge。次の cron から Anneal が処理を開始

### B) 監視を一時停止 / 再開する

- ワークフローを無効化（GitHub UI の Actions で disable、または `on:` を絞る／ファイルを一時削除）
- 再開は逆操作。**対象 repo 内で完結**（Anneal 側の操作は不要）

### C) パッケージを更新対象から除外する

- 対象 repo の `.anneal.yml` の `ignore:` に追記 → PR → merge
- 次回 scan 時に Anneal が `.anneal.yml` を読んで適用

### D) 更新パラメータを変える

- 対象 repo の `.anneal.yml`（`auto_pr_types` / `regression_window_days` / `base_branch`）を編集 → PR → merge
- 組織全体の既定値を変えたい場合は Anneal 側の既定（`DefaultRepoConfig`）を更新して再デプロイ

### E) 実行時に何が起きるか（自動・操作不要）

```
対象 repo の CI (GitHub Actions, cron)
   │  POST /internal/scan {repository: owner/repo}  + 認証
   ▼
Anneal (Cloud Run)
   ├─ 1. リクエストの repo を受ける
   ├─ 2. GitHub Contents API で manifest 取得（package.json / go.mod ＋ lockfile。clone しない）
   ├─ 3. その repo の .anneal.yml を取得 → ignore / パラメータ適用
   ├─ 4. 更新候補を検出 → detected レコードを upsert（update_key で冪等）
   └─ 5. 以降は既存の状態機械（分析 → 承認 → PR → CI → スコア → Annealing）
```

---

## 早見表（どこを操作する？）

| やりたいこと | 操作場所 | Anneal 再デプロイ |
|---|---|---|
| repo を監視追加 | 対象 repo に `anneal.yml` ＋ Secret | 不要 |
| 監視の一時停止 / 再開 | 対象 repo のワークフロー無効化 / 有効化 | 不要 |
| パッケージ除外 | 対象 repo の `.anneal.yml`（`ignore`） | 不要 |
| repo 固有パラメータ | 対象 repo の `.anneal.yml` | 不要 |
| 組織全体の既定値変更 | Anneal 側 `DefaultRepoConfig` | 要 |

## 注意点

- **認証情報の配布が必要**：対象 repo ごとに `ANNEAL_TOKEN`（または OIDC 信頼設定）を持たせる。repo が増えるたびにこの配布作業が発生する。組織 Secret を使えば一括配布も可能。
- **中央に監視対象一覧が無い**：どの repo が Anneal を使っているかは「`anneal.yml` を置いた repo」でしか分からない（＝完全オプトイン）。全体像を一覧したい場合は別途ダッシュボード／検索が要る（将来の別トラック）。
- **トリガーは対象 repo の CI 任せ**：cron を止めれば scan も止まる。確実性を上げたいなら Anneal 側の定期照合（reconcile）を保険として併用できる。
