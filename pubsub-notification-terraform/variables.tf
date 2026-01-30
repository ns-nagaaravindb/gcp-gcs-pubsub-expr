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

variable "landing_bucket_name" {
  description = "Name of the GCS landing bucket"
  type        = string
  default     = "test-dp-gcspubsub-bucket-landing"
}

variable "topic_name" {
  description = "Name of the Pub/Sub topic"
  type        = string
  default     = "test-dp-gcspubsub-bucket"
}

variable "force_destroy" {
  description = "When deleting a bucket, this boolean option will delete all contained objects"
  type        = bool
  default     = true
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

variable "consumer_project_id" {
  description = "The GCP project ID for the consumer (can be different from the publisher project)"
  type        = string
  default     = "" # If empty, uses the same project as the publisher
}

variable "create_cross_project_subscription" {
  description = "Whether to create a subscription in a different project for cross-project consumption"
  type        = bool
  default     = false
}

variable "service_account_id" {
  description = "The account ID for the service account (used for bucket operations)"
  type        = string
  default     = "gcs-bucket-operator"
}

variable "create_service_account_key" {
  description = "Whether to create a service account key for authentication"
  type        = bool
  default     = true
}

variable "service_account_key_filename" {
  description = "Filename for the service account key JSON file"
  type        = string
  default     = "service-account-key.json"
}

