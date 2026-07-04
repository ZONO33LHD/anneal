output "job_names" {
  description = "作成された Cloud Scheduler job 名。"
  value       = [for job in google_cloud_scheduler_job.internal : job.name]
}
