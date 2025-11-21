variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "region" {
  description = "The GCP region for resources"
  type        = string
  default     = "us-central1"
}

variable "bucket_name" {
  description = "Name of the GCS bucket"
  type        = string
  default     = "test-dp-gcspubsub-bucket"
}

variable "topic_name" {
  description = "Name of the Pub/Sub topic"
  type        = string
  default     = "test-dp-gcspubsub-bucket"
}

variable "force_destroy" {
  description = "When deleting a bucket, this boolean option will delete all contained objects"
  type        = bool
  default     = false
}

variable "enable_versioning" {
  description = "Enable versioning on the bucket"
  type        = bool
  default     = false
}

variable "lifecycle_age" {
  description = "Age in days after which objects should be deleted"
  type        = number
  default     = 365
}

variable "labels" {
  description = "Labels to apply to resources"
  type        = map(string)
  default     = {}
}

variable "message_retention_duration" {
  description = "How long to retain unacknowledged messages in the topic"
  type        = string
  default     = "604800s" # 7 days
}

variable "ack_deadline_seconds" {
  description = "The maximum time after a subscriber receives a message before the subscriber should acknowledge the message"
  type        = number
  default     = 20
}

variable "subscription_ttl" {
  description = "The expiration time for the subscription"
  type        = string
  default     = "" # Never expire
}

variable "minimum_backoff" {
  description = "The minimum delay between consecutive deliveries of a given message"
  type        = string
  default     = "10s"
}

variable "maximum_backoff" {
  description = "The maximum delay between consecutive deliveries of a given message"
  type        = string
  default     = "600s"
}

