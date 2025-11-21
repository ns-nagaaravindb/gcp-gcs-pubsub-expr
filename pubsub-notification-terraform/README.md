# GCS Pub/Sub Notification Terraform Module

This Terraform module creates a GCS bucket with Pub/Sub notifications configured for file creation events.

## Resources Created

- **GCS Bucket**: `test-dp-gcspubsub-bucket` (configurable)
- **Pub/Sub Topic**: `test-dp-gcspubsub-bucket` (configurable)
- **Pub/Sub Subscription**: For consuming messages
- **GCS Notification**: Configured to publish OBJECT_FINALIZE events to the Pub/Sub topic
- **IAM Permissions**: GCS service account granted Pub/Sub Publisher role

## Prerequisites

1. Google Cloud SDK installed and configured
2. Authenticated with `gcloud auth application-default login`
3. Terraform >= 1.0 installed
4. Appropriate GCP project permissions

## Usage

1. Copy `terraform.tfvars.example` to `terraform.tfvars`:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   ```

2. Edit `terraform.tfvars` with your project details:
   ```hcl
   project_id = "your-gcp-project-id"
   region     = "us-central1"
   ```

3. Initialize Terraform:
   ```bash
   terraform init
   ```

4. Review the plan:
   ```bash
   terraform plan
   ```

5. Apply the configuration:
   ```bash
   terraform apply
   ```

6. Destroy resources when done:
   ```bash
   terraform destroy
   ```

## Variables

See `variables.tf` for all available variables and their descriptions.

## Outputs

After applying, the module outputs:
- `bucket_name`: Name of the created bucket
- `bucket_url`: URL of the bucket
- `topic_name`: Name of the Pub/Sub topic
- `subscription_name`: Name of the subscription
- `notification_id`: ID of the notification configuration

## Authentication

This module uses `gcloud` credentials. Ensure you're authenticated:
```bash
gcloud auth application-default login
```

## Production Considerations

- Set `force_destroy = false` in production
- Enable versioning if needed: `enable_versioning = true`
- Configure appropriate lifecycle rules
- Add monitoring and alerting
- Use remote state backend (GCS recommended)
- Enable audit logs

