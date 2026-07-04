data "google_project" "project" {}

# 実行用 SA は iam モジュールで作成済み。ここでは参照するだけ
# （cloudrun は iam の後に apply される前提）。
data "google_service_account" "runtime" {
  account_id = "anneal-runtime"
}

data "google_service_account" "scheduler" {
  account_id = "anneal-scheduler"
}

locals {
  # var.image 未指定時は Artifact Registry の :latest を使う。
  # （artifactregistry モジュールの repository_id=cloudrun, イメージ名=anneal に一致）
  image = var.image != "" ? var.image : "${var.region}-docker.pkg.dev/${var.project_id}/cloudrun/anneal:latest"
}
