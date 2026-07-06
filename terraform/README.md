# Terraform

Anneal の Terraform は、PR で `plan`、`main` への merge で `apply` します。後続リソースは `terraform/path-filter/anneal.yml` の記述順に追加します。

## モジュール構成（apply 順）

`terraform/path-filter/anneal.yml` の記述順 = apply 順です。後段は前段に依存します。

1. `api` — 後続が依存する Google Cloud API を有効化
2. `artifactregistry` — Cloud Run 用 Docker イメージのリポジトリ
3. `firestore` — アプリ状態の永続化先（`(default)` DB, Native, asia-northeast1, 削除保護あり）
4. `secretmanager` — `GITHUB_TOKEN` / `GITHUB_WEBHOOK_SECRET` / `GEMINI_API_KEY` / `SLACK_WEBHOOK_URL` の secret（箱のみ）
5. `iam` — Cloud Run 実行用 SA（`anneal-runtime`）と最小権限（datastore.user / secretmanager.secretAccessor / logging.logWriter）
6. `cloudrun` — Webhook 受け口（`serve`）を動かす Cloud Run サービス。runtime SA を使い、Firestore backend と secret（env 経由）を参照する。GitHub からの Webhook 到達のため公開（`run.invoker` = allUsers）だが、リクエストは serve 側の HMAC 署名検証で保護する
7. `scheduler` — Cloud Scheduler から `/internal/scan` と `/internal/tick` を OIDC token 付きで定期起動する。`cloudrun` の `scan_targets` 変数で `ANNEAL_SCAN_TARGETS` を渡す

イメージの初回 push（`artifactregistry` への `:latest`）は `cloudrun` の apply より前に必要です（`image-build` ワークフローが供給）。Cloud Run のイメージは `lifecycle.ignore_changes` 対象で、実ロールアウトは deploy ワークフローが `:<sha>` で更新します。GitHub Webhook の送信先は `<service_uri>/webhooks/github`、内部定期起動は `<service_uri>/internal/scan` / `<service_uri>/internal/tick` です。

### Secret の値の投入

`secretmanager` モジュールは secret の**箱だけ**を作り、値（version）は Terraform で管理しません（state やコードに秘密を載せないため）。値の投入方法は 2 通りです。

**A. GitHub Actions で同期（推奨: `GEMINI_API_KEY`）**

GitHub Secrets に `GEMINI_API_KEY` を登録し、`.github/workflows/secret-sync.yml`（`secret-sync`）を手動実行すると、その値を Secret Manager の同名 secret へ反映します。

- トリガー: Actions タブ → `secret-sync` → Run workflow（`workflow_dispatch`）。GitHub Secrets の変更はイベントを発火しないため手動実行。
- 冪等: 既存の `latest` と一致する場合は version を増やしません（version 増殖と課金を回避）。
- 反映: `redeploy` 入力（既定 true）が有効かつ新 version を追加した場合、Cloud Run を再デプロイして `latest` を取り込みます（サービス未作成時はスキップ）。

投入手順:

1. `secretmanager` モジュールを apply して箱を作成（初回のみ）。
2. リポジトリ Settings → Secrets and variables → Actions に `GEMINI_API_KEY` を登録。
3. `secret-sync` ワークフローを実行。

なお、Terraform 実行用 SA（`GCP_SERVICE_ACCOUNT`）には version の追加・参照権限が必要です（`roles/secretmanager.admin`、または `secretVersionAdder` ＋ `secretAccessor`）。

**B. 手動投入（`gcloud`）**

```sh
printf '%s' "$GEMINI_API_KEY" | gcloud secrets versions add GEMINI_API_KEY --data-file=- --project anneal-500804
```

## GitHub Secrets

Terraform Actions を動かす前に、リポジトリに以下の secrets を設定してください。

- `GCP_PROJECT_ID`: Google Cloud project ID（実プロジェクトは `anneal-500804`。terraform 変数 `project_id` の既定値も同じ）
- `GCP_WIF_PROVIDER`: GitHub Actions 用 Workload Identity Provider
- `GCP_SERVICE_ACCOUNT`: Terraform を実行する service account のメールアドレス
- `GEMINI_API_KEY`: Gemini API キー（`secret-sync` ワークフローで Secret Manager に同期）。最安ティアの `gemini-2.5-flash-lite` を利用する前提。

PR では secrets が未設定でも `fmt` / `validate` までは実行し、GCP 認証が必要な `plan` はスキップします。`main` merge 後の `apply` ではこれらの secrets が必須です。

## Backend

Terraform state は GCS backend を使います。初回実行前に bucket を事前作成してください。

```sh
gcloud storage buckets create gs://anneal-terraforms --project anneal-500804 --location asia-northeast1
```

各 module の state prefix は module directory から生成します。たとえば `terraform/resources/anneal-prd/api` は `terraform.resources.anneal-prd.api` になります（`anneal-prd` は directory 名 = 環境ラベルで、GCP プロジェクト ID `anneal-500804` とは別軸です）。

## Workload Identity Federation

GitHub Actions は `google-github-actions/auth` で Workload Identity Federation により認証します。

- `project_id`: `${{ secrets.GCP_PROJECT_ID }}`
- `workload_identity_provider`: `${{ secrets.GCP_WIF_PROVIDER }}`
- `service_account`: `${{ secrets.GCP_SERVICE_ACCOUNT }}`
- `audience`: `https://github.com/${{ github.repository }}`

Provider の attribute condition は、この repository からの実行だけを許可する形にしてください。Terraform 実行用 service account には、少なくとも backend bucket の読み書き権限と、対象リソースを管理するための権限が必要です。

## Operation

1. Terraform 変更の PR を作る。
2. `.github/workflows/terraform-plan.yml` が変更された Terraform directory を検出し、`terraform fmt -check`、`terraform validate`、`terraform plan` を実行する。
3. plan 結果が PR にコメントされる。
4. PR を `main` に merge すると、`.github/workflows/terraform-apply.yml` が同じ directory を検出し、記述順に `terraform apply -auto-approve` を実行する。
