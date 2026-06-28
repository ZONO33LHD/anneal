output "runtime_service_account_email" {
  description = "Cloud Run の runtime service account として使う SA のメールアドレス。"
  value       = google_service_account.runtime.email
}
