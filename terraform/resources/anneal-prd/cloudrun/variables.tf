variable "project_id" {
  description = "Google Cloud project ID. 既定は anneal プロジェクト（anneal-500804）。CI は TF_VAR_project_id（GitHub Secret GCP_PROJECT_ID）で上書きする。"
  type        = string
  default     = "anneal-500804"
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
  description = "ANNEAL_SCAN_TARGETS に渡す scan 対象。カンマ区切りで /path/to/repo または /path/to/repo=owner/repo。既定は空（未設定）。CI apply では TF_VAR_scan_targets が未指定でも apply が通るよう default を持たせる。"
  type        = string
  default     = ""
}
