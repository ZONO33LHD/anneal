variable "project_id" {
  description = "Google Cloud project ID. 既定は anneal プロジェクト（anneal-500804）。CI は TF_VAR_project_id（GitHub Secret GCP_PROJECT_ID）で上書きする。"
  type        = string
  default     = "anneal-500804"
}

variable "region" {
  description = "Cloud Run と Cloud Scheduler のリージョン。"
  type        = string
  default     = "asia-northeast1"
}

variable "cloud_run_service_name" {
  description = "内部 endpoint を持つ Cloud Run service 名。"
  type        = string
  default     = "anneal"
}

variable "internal_oidc_audience" {
  description = "Cloud Run 側の ANNEAL_INTERNAL_OIDC_AUDIENCE と一致させる audience。"
  type        = string
  default     = "anneal-scheduler"
}

variable "time_zone" {
  description = "Cloud Scheduler job の timezone。"
  type        = string
  default     = "Asia/Tokyo"
}

variable "scan_schedule" {
  description = "/internal/scan の cron。"
  type        = string
  default     = "0 3 * * *"
}

variable "tick_schedule" {
  description = "/internal/tick の cron。"
  type        = string
  default     = "*/30 * * * *"
}
