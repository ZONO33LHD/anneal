data "google_cloud_run_v2_service" "anneal" {
  name     = var.cloud_run_service_name
  location = var.region
}

data "google_service_account" "scheduler" {
  account_id = "anneal-scheduler"
}
