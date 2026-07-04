# Cloud Run で動く Anneal の実行 ID（サービスアカウント）。後続の Cloud Run モジュールが
# この SA を runtime service account として参照する。
resource "google_service_account" "runtime" {
  account_id   = "anneal-runtime"
  display_name = "Anneal Cloud Run runtime"
}

resource "google_service_account" "scheduler" {
  account_id   = "anneal-scheduler"
  display_name = "Anneal Cloud Scheduler invoker"
}

# 最小権限の付与:
# - datastore.user:        Firestore（T3 の永続化先）への読み書き
# - secretmanager.secretAccessor: GITHUB_TOKEN 等の secret 値の取得
# - logging.logWriter:     構造化ログ（slog/JSON）の Cloud Logging への書き込み
resource "google_project_iam_member" "runtime" {
  for_each = toset([
    "roles/datastore.user",
    "roles/secretmanager.secretAccessor",
    "roles/logging.logWriter",
  ])

  project = var.project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.runtime.email}"
}
