# Anneal の Webhook 受け口（serve）を動かす Cloud Run サービス。
# 依存: iam（runtime SA）/ secretmanager（secret 値）/ firestore（永続化先）/
# artifactregistry（イメージ）。これらの後に apply される前提。
resource "google_cloud_run_v2_service" "anneal" {
  name     = "anneal"
  location = var.region

  # サービスは CI が管理する（状態は Firestore にあり、サービス自体は再作成可能）。
  # provider 既定の deletion_protection=true だと、失敗 revision の taint 置き換えや
  # 不変フィールド変更に伴う destroy がブロックされ apply が止まるため false にする。
  deletion_protection = false

  # GitHub からの Webhook を受けるため外部からの到達を許可する。
  # リクエストは serve 側の HMAC-SHA256 署名検証で保護する。
  ingress = "INGRESS_TRAFFIC_ALL"

  template {
    # 専用 runtime SA を使う（default compute SA は権限が広すぎるため使わない）。
    service_account = data.google_service_account.runtime.email

    scaling {
      min_instance_count = 0
      max_instance_count = 3
    }

    containers {
      image = local.image

      ports {
        container_port = 8080
      }

      # 永続化は Firestore、ログは Cloud Logging 向けの JSON 構造化ログにする。
      env {
        name  = "ANNEAL_STORE_BACKEND"
        value = "firestore"
      }
      env {
        name  = "ANNEAL_LOG_FORMAT"
        value = "json"
      }
      env {
        name  = "GOOGLE_CLOUD_PROJECT"
        value = var.project_id
      }
      env {
        name  = "ANNEAL_INTERNAL_OIDC_AUDIENCE"
        value = var.internal_oidc_audience
      }
      env {
        name  = "ANNEAL_INTERNAL_OIDC_EMAIL"
        value = data.google_service_account.scheduler.email
      }
      env {
        name  = "ANNEAL_SCAN_TARGETS"
        value = var.scan_targets
      }

      # secret は値を埋め込まず、Secret Manager の最新バージョンを参照する。
      # ただし version（値）が存在しない secret を latest 参照すると Cloud Run の
      # 起動が "Secret ... not found" で失敗するため、値を投入済みの secret だけを
      # var.mounted_secrets で mount する（既定は Gemini のみ）。未 mount の機能は
      # アプリ側が env 空としてモック/コンソールにフォールバックする。
      dynamic "env" {
        for_each = toset(var.mounted_secrets)
        content {
          name = env.value
          value_source {
            secret_key_ref {
              secret  = env.value
              version = "latest"
            }
          }
        }
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }
    }
  }

  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }

  # イメージは deploy ワークフローが :<sha> で更新するため drift 対象から外す。
  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
      client,
      client_version,
    ]
  }
}

# GitHub からの Webhook POST を受けられるよう公開する。
# 認証は serve 側の署名検証で行うため、run.invoker は allUsers を許可する。
resource "google_cloud_run_v2_service_iam_member" "public" {
  project  = google_cloud_run_v2_service.anneal.project
  location = google_cloud_run_v2_service.anneal.location
  name     = google_cloud_run_v2_service.anneal.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
