output "bucket_name" {
  description = "Name of the created GCS bucket"
  value       = google_storage_bucket.main.name
}

output "bucket_url" {
  description = "URL of the created GCS bucket"
  value       = google_storage_bucket.main.url
}

output "landing_bucket_name" {
  description = "Name of the created GCS landing bucket"
  value       = google_storage_bucket.landing.name
}

output "landing_bucket_url" {
  description = "URL of the created GCS landing bucket"
  value       = google_storage_bucket.landing.url
}

output "topic_name" {
  description = "Name of the created Pub/Sub topic"
  value       = google_pubsub_topic.main.name
}

output "topic_id" {
  description = "ID of the created Pub/Sub topic"
  value       = google_pubsub_topic.main.id
}

output "subscription_name" {
  description = "Name of the created Pub/Sub subscription"
  value       = google_pubsub_subscription.main.name
}

output "subscription_id" {
  description = "ID of the created Pub/Sub subscription"
  value       = google_pubsub_subscription.main.id
}

output "notification_id" {
  description = "ID of the created GCS notification"
  value       = google_storage_notification.main.id
}

output "gcs_service_account_email" {
  description = "Email address of the GCS service account"
  value       = data.google_storage_project_service_account.gcs_account.email_address
}

output "consumer_subscription_name" {
  description = "Name of the consumer Pub/Sub subscription (cross-project)"
  value       = var.create_cross_project_subscription && var.consumer_project_id != "" ? google_pubsub_subscription.consumer[0].name : null
}

output "consumer_subscription_id" {
  description = "ID of the consumer Pub/Sub subscription (cross-project)"
  value       = var.create_cross_project_subscription && var.consumer_project_id != "" ? google_pubsub_subscription.consumer[0].id : null
}

output "consumer_project_id" {
  description = "Consumer project ID"
  value       = var.consumer_project_id != "" ? var.consumer_project_id : var.project_id
}

output "service_account_email" {
  description = "Email address of the bucket operator service account"
  value       = google_service_account.bucket_operator.email
}

output "service_account_name" {
  description = "Name of the bucket operator service account"
  value       = google_service_account.bucket_operator.name
}

output "service_account_key_file" {
  description = "Path to the service account key file"
  value       = var.create_service_account_key ? "${path.module}/${var.service_account_key_filename}" : null
  sensitive   = true
}

output "service_account_unique_id" {
  description = "Unique ID of the service account"
  value       = google_service_account.bucket_operator.unique_id
}

