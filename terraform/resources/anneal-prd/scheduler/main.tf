locals {
  internal_jobs = {
    scan = {
      path     = "/internal/scan"
      schedule = var.scan_schedule
    }
    tick = {
      path     = "/internal/tick"
      schedule = var.tick_schedule
    }
  }
}

resource "google_cloud_run_v2_service_iam_member" "scheduler" {
  project  = data.google_cloud_run_v2_service.anneal.project
  location = data.google_cloud_run_v2_service.anneal.location
  name     = data.google_cloud_run_v2_service.anneal.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${data.google_service_account.scheduler.email}"
}

resource "google_cloud_scheduler_job" "internal" {
  for_each = local.internal_jobs

  name        = "anneal-${each.key}"
  description = "Invoke Anneal ${each.key} internal endpoint"
  region      = var.region
  schedule    = each.value.schedule
  time_zone   = var.time_zone

  retry_config {
    retry_count          = 3
    min_backoff_duration = "10s"
    max_backoff_duration = "300s"
    max_doublings        = 3
  }

  http_target {
    uri         = "${data.google_cloud_run_v2_service.anneal.uri}${each.value.path}"
    http_method = "POST"

    headers = {
      User-Agent = "anneal-cloud-scheduler"
    }

    oidc_token {
      service_account_email = data.google_service_account.scheduler.email
      audience              = var.internal_oidc_audience
    }
  }

  depends_on = [google_cloud_run_v2_service_iam_member.scheduler]
}
