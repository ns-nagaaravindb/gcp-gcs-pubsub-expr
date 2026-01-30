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

# GCS Landing Bucket
resource "google_storage_bucket" "landing" {
  name                        = var.landing_bucket_name
  location                    = var.region
  force_destroy               = true
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
  name  = "${var.topic_name}-sub"
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

# Cross-project consumer setup (optional)
# Provider for consumer project
provider "google" {
  alias   = "consumer"
  project = var.consumer_project_id != "" ? var.consumer_project_id : var.project_id
  region  = var.region
}

# Cross-project Pub/Sub Subscription (in consumer project)
resource "google_pubsub_subscription" "consumer" {
  count    = var.create_cross_project_subscription && var.consumer_project_id != "" ? 1 : 0
  provider = google.consumer

  name  = "${var.topic_name}-consumer-sub"
  topic = google_pubsub_topic.main.id

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

# IAM: Grant consumer project subscription access to the topic
resource "google_pubsub_topic_iam_member" "consumer_subscriber" {
  count  = var.create_cross_project_subscription && var.consumer_project_id != "" ? 1 : 0
  topic  = google_pubsub_topic.main.name
  role   = "roles/pubsub.subscriber"
  member = "serviceAccount:service-${data.google_project.consumer[0].number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

# Data source to get consumer project number
data "google_project" "consumer" {
  count      = var.create_cross_project_subscription && var.consumer_project_id != "" ? 1 : 0
  provider   = google.consumer
  project_id = var.consumer_project_id
}

# Service Account for bucket operations (upload/download)
resource "google_service_account" "bucket_operator" {
  account_id   = var.service_account_id
  display_name = "GCS Bucket Operator Service Account"
  description  = "Service account for uploading and downloading files to/from GCS bucket"
}

# IAM: Grant service account permission to upload files to bucket (objectCreator)
resource "google_storage_bucket_iam_member" "bucket_operator_creator" {
  bucket = google_storage_bucket.main.name
  role   = "roles/storage.objectCreator"
  member = "serviceAccount:${google_service_account.bucket_operator.email}"
}

# IAM: Grant service account permission to download files from bucket (objectViewer)
resource "google_storage_bucket_iam_member" "bucket_operator_viewer" {
  bucket = google_storage_bucket.main.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.bucket_operator.email}"
}

# IAM: Grant service account permission to upload files to landing bucket (objectCreator)
resource "google_storage_bucket_iam_member" "landing_bucket_operator_creator" {
  bucket = google_storage_bucket.landing.name
  role   = "roles/storage.objectCreator"
  member = "serviceAccount:${google_service_account.bucket_operator.email}"
}

# IAM: Grant service account permission to download files from landing bucket (objectViewer)
resource "google_storage_bucket_iam_member" "landing_bucket_operator_viewer" {
  bucket = google_storage_bucket.landing.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.bucket_operator.email}"
}



