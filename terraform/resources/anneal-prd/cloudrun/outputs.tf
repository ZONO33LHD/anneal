output "service_uri" {
  description = "Cloud Run サービスの URL。GitHub Webhook の送信先は <uri>/webhooks/github。"
  value       = google_cloud_run_v2_service.anneal.uri
}
