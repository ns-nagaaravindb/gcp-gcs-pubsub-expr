terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# GCS Bucket
resource "google_storage_bucket" "main" {
  name                        = var.bucket_name
  location                    = var.region
  force_destroy               = var.force_destroy
  uniform_bucket_level_access = true

  versioning {
    enabled = var.enable_versioning
  }

  lifecycle_rule {
    condition {
      age = var.lifecycle_age
    }
    action {
      type = "Delete"
    }
  }

  labels = var.labels
}

# Pub/Sub Topic
resource "google_pubsub_topic" "main" {
  name = var.topic_name

  labels = var.labels

  message_retention_duration = var.message_retention_duration
}

# Pub/Sub Subscription (for the application to consume messages)
resource "google_pubsub_subscription" "main" {
  name  = "${var.topic_name}-subscription"
  topic = google_pubsub_topic.main.name

  ack_deadline_seconds = var.ack_deadline_seconds

  expiration_policy {
    ttl = var.subscription_ttl
  }

  retry_policy {
    minimum_backoff = var.minimum_backoff
    maximum_backoff = var.maximum_backoff
  }

  enable_message_ordering = false
}

# IAM: Grant GCS service account permission to publish to Pub/Sub topic
resource "google_pubsub_topic_iam_member" "gcs_publisher" {
  topic  = google_pubsub_topic.main.name
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"
}

# Data source to get GCS service account
data "google_storage_project_service_account" "gcs_account" {
  project = var.project_id
}

# GCS Bucket Notification
resource "google_storage_notification" "main" {
  bucket         = google_storage_bucket.main.name
  payload_format = "JSON_API_V1"
  topic          = google_pubsub_topic.main.id
  event_types    = ["OBJECT_FINALIZE"]

  depends_on = [google_pubsub_topic_iam_member.gcs_publisher]
}

