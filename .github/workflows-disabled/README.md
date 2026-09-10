# 無効化済みワークフロー

コスト削減のため、2026-09-11 に GitHub Actions の CI/CD と Terraform の自動 plan/apply を停止しました。
GitHub は `.github/workflows/` 直下のファイルしか実行しないため、このディレクトリに置いたワークフローは動きません。

| ファイル | 役割 |
|---|---|
| `go-ci.yml` | Go の build / vet / test / lint |
| `image-build.yml` | コンテナイメージの build → Artifact Registry push → Cloud Run deploy |
| `terraform-plan.yml` | PR での `terraform plan` |
| `terraform-apply.yml` | `main` merge 時の `terraform apply` |
| `secret-sync.yml` | GitHub Secrets → Secret Manager 同期（手動） |
| `run-anneal.yml` | Cloud Run の `/internal/tick` `/internal/scan` 手動起動 |
| `anneal.yml` | 自己スキャン（cron） |

## 再有効化

```sh
git mv .github/workflows-disabled/<name>.yml .github/workflows/<name>.yml
```

GitHub 側で `gh workflow disable` した履歴が残っている場合は、復帰後に `gh workflow enable <name>` も実行してください。
Terraform を再稼働させる前提条件（WIF / SA / state bucket）は `terraform/README.md` を参照してください。
