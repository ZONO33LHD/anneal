variable "project_id" {
  description = "Google Cloud project ID. 既定は anneal プロジェクト（anneal-500804）。CI は TF_VAR_project_id（GitHub Secret GCP_PROJECT_ID）で上書きする。"
  type        = string
  default     = "anneal-500804"
}

variable "ci_service_account" {
  description = "GitHub Actions が WIF で認証する CI/Terraform 実行 SA のメール。scheduler SA を impersonate して内部 endpoint 用の OIDC ID トークンを発行するため token creator を付与する。"
  type        = string
  default     = "terraform-ci@anneal-500804.iam.gserviceaccount.com"
}
