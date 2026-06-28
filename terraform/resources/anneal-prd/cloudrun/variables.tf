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
