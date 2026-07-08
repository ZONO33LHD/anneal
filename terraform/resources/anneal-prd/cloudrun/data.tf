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
  # var.image 未指定時は Cloud Run の公開サンプルイメージで「初回作成」する。
  # 実イメージは image-build/deploy ワークフローが AR の :<sha> をロールアウトし、
  # image は lifecycle.ignore_changes 対象なので terraform はそれを上書きしない。
  # これにより AR に anneal:latest がまだ無い greenfield でも Cloud Run サービスを
  # 作成でき、apply が「Image not found」で失敗しなくなる。
  # 実イメージを terraform で固定したい場合は var.image を明示指定する。
  image = var.image != "" ? var.image : "us-docker.pkg.dev/cloudrun/container/hello"
}
