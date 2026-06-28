output "repository_id" {
  description = "作成した Artifact Registry リポジトリの ID。"
  value       = google_artifact_registry_repository.cloudrun.repository_id
}

output "repository_url" {
  description = "イメージ push/pull に使う Artifact Registry のホスト + パス。"
  value       = "${google_artifact_registry_repository.cloudrun.location}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.cloudrun.repository_id}"
}
