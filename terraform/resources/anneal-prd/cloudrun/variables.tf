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

variable "mounted_secrets" {
  description = "Cloud Run に env として mount する Secret Manager secret 名。値（version）を投入済みのものだけを指定する。version が無い secret を latest 参照すると Cloud Run 起動が失敗するため。既定は Gemini のみ。GitHub/Slack などを使う場合は値を投入してからこの list に追加する（例: [\"GEMINI_API_KEY\", \"GITHUB_TOKEN\", \"GITHUB_WEBHOOK_SECRET\", \"SLACK_WEBHOOK_URL\"]）。"
  type        = list(string)
  default     = ["GEMINI_API_KEY"]
}
