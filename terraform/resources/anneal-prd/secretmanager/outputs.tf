output "secret_ids" {
  description = "作成した secret の ID 一覧。値（version）は別途手動で投入する。"
  value       = [for s in google_secret_manager_secret.app : s.secret_id]
}
