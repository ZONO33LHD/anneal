# Cloud Run にデプロイするコンテナイメージの置き場。後続の Cloud Run / イメージ
# ビルドパイプラインがここへ push し、ここから pull する。
resource "google_artifact_registry_repository" "cloudrun" {
  location      = "asia-northeast1"
  repository_id = "cloudrun"
  description   = "Docker images for Anneal Cloud Run"
  format        = "DOCKER"
}
