# アプリが実行時に参照する secret の「箱」だけを作る。値（version）は Terraform で
# 管理しない（state やコードに秘密が載らないようにするため）。値は運用で手動投入する:
#   gcloud secrets versions add GITHUB_TOKEN --data-file=- <<<"$TOKEN"
resource "google_secret_manager_secret" "app" {
  for_each = toset([
    "GITHUB_TOKEN",
    "GITHUB_WEBHOOK_SECRET",
    "GEMINI_API_KEY",
    "SLACK_WEBHOOK_URL",
    # 内部 endpoint(/internal/*) の共有トークン。対象 repo の anneal.yml が
    # X-Anneal-Internal-Token として送る値と一致させる（ANNEAL_INTERNAL_TOKEN）。
    "ANNEAL_INTERNAL_TOKEN",
  ])

  secret_id = each.value

  replication {
    auto {}
  }
}
