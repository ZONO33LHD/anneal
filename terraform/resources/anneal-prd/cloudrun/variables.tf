variable "project_id" {
  description = "Google Cloud project ID."
  type        = string
}

variable "region" {
  description = "Cloud Run と Artifact Registry のリージョン。"
  type        = string
  default     = "asia-northeast1"
}

variable "image" {
  description = "起動するコンテナイメージ。既定は Artifact Registry の :latest。実際のロールアウトは deploy ワークフローが :<sha> で上書きする。"
  type        = string
  default     = ""
}

variable "internal_oidc_audience" {
  description = "Cloud Scheduler が内部 endpoint を呼ぶ OIDC token の audience。scheduler モジュールと同じ値にする。"
  type        = string
  default     = "anneal-scheduler"
}

variable "scan_targets" {
  description = "ANNEAL_SCAN_TARGETS に渡す scan 対象。カンマ区切りで /path/to/repo または /path/to/repo=owner/repo。"
  type        = string
}
