# Terraform

Anneal の Terraform は、PR で `plan`、`main` への merge で `apply` します。最初の対象は `terraform/resources/anneal-prd/api` のみで、後続リソースは `terraform/path-filter/anneal.yml` の記述順に追加します。

## GitHub Secrets

Terraform Actions を動かす前に、リポジトリに以下の secrets を設定してください。

- `GCP_PROJECT_ID`: Google Cloud project ID (`anneal-prd`)
- `GCP_WIF_PROVIDER`: GitHub Actions 用 Workload Identity Provider
- `GCP_SERVICE_ACCOUNT`: Terraform を実行する service account のメールアドレス

## Backend

Terraform state は GCS backend を使います。初回実行前に bucket を事前作成してください。

```sh
gcloud storage buckets create gs://anneal-terraforms --project anneal-prd --location asia-northeast1
```

各 module の state prefix は module directory から生成します。たとえば `terraform/resources/anneal-prd/api` は `terraform.resources.anneal-prd.api` になります。

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
