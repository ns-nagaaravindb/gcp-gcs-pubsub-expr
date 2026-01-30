# GCS Pub/Sub Notification Terraform Module

This Terraform module creates a GCS bucket with Pub/Sub notifications configured for file creation events.

## ⭐ NEW: Cross-Project Consumer Support

**This module now supports creating Pub/Sub subscriptions in a different GCP project!**

👉 See [CROSS_PROJECT_SETUP.md](./CROSS_PROJECT_SETUP.md) for detailed cross-project setup instructions.

Quick example:
```bash
make cross-project-apply \
  PROJECT_ID=publisher-project \
  CONSUMER_PROJECT_ID=consumer-project
```

## Resources Created

- **GCS Bucket**: `test-dp-gcspubsub-bucket` (configurable)
- **GCS Landing Bucket**: `test-dp-gcspubsub-bucket-landing` (configurable, NEW!)
- **Pub/Sub Topic**: `test-dp-gcspubsub-bucket` (configurable)
- **Pub/Sub Subscription**: For consuming messages (same project)
- **Cross-Project Subscription**: Optional subscription in a different project (NEW!)
- **Service Account**: For uploading/downloading files to/from the buckets (NEW!)
- **Service Account Key**: JSON key for authentication (optional, NEW!)
- **GCS Notification**: Configured to publish OBJECT_FINALIZE events to the Pub/Sub topic
- **IAM Permissions**: 
  - GCS service account granted Pub/Sub Publisher role
  - Bucket operator service account granted Storage Object Creator and Viewer roles (both buckets)
- **Cross-Project IAM**: Consumer project granted Subscriber role (when enabled)

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
- `landing_bucket_name`: Name of the created landing bucket (NEW!)
- `landing_bucket_url`: URL of the landing bucket (NEW!)
- `topic_name`: Name of the Pub/Sub topic
- `subscription_name`: Name of the subscription (same project)
- `consumer_subscription_name`: Name of the consumer subscription (cross-project, if enabled)
- `consumer_project_id`: Consumer project ID
- `notification_id`: ID of the notification configuration
- `service_account_email`: Email of the bucket operator service account (NEW!)
- `service_account_key_file`: Path to the service account key JSON file (NEW!)

## Quick Start Examples

### Single Project Setup
```bash
# Using Makefile
make terraform-apply PROJECT_ID=your-project

# Using Terraform directly
terraform apply -var="project_id=your-project"
```

### Cross-Project Setup
```bash
# Using Makefile
make cross-project-apply \
  PROJECT_ID=publisher-project \
  CONSUMER_PROJECT_ID=consumer-project

# Using Terraform directly
terraform apply \
  -var="project_id=publisher-project" \
  -var="consumer_project_id=consumer-project" \
  -var="create_cross_project_subscription=true"
```

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

## Using the Service Account

The module creates a service account with permissions to upload and download files to/from the GCS bucket.

### Authenticate using the service account key:

```bash
# Set the environment variable
export GOOGLE_APPLICATION_CREDENTIALS="./service-account-key.json"

# Or use gcloud
gcloud auth activate-service-account --key-file=./service-account-key.json
```

### Upload a file to the bucket:

```bash
# Using gsutil - upload to main bucket
gsutil cp myfile.txt gs://your-bucket-name/

# Using gsutil - upload to landing bucket
gsutil cp myfile.txt gs://your-bucket-name-landing/

# Using gcloud - upload to main bucket
gcloud storage cp myfile.txt gs://your-bucket-name/

# Using gcloud - upload to landing bucket
gcloud storage cp myfile.txt gs://your-bucket-name-landing/
```

### Download a file from the bucket:

```bash
# Using gsutil - download from main bucket
gsutil cp gs://your-bucket-name/myfile.txt ./

# Using gsutil - download from landing bucket
gsutil cp gs://your-bucket-name-landing/myfile.txt ./

# Using gcloud - download from main bucket
gcloud storage cp gs://your-bucket-name/myfile.txt ./

# Using gcloud - download from landing bucket
gcloud storage cp gs://your-bucket-name-landing/myfile.txt ./
```

### Security Note:

**⚠️ IMPORTANT**: The service account key (`service-account-key.json`) contains sensitive credentials. 
- Never commit this file to version control (it's already in `.gitignore`)
- Store it securely (use secret management tools in production)
- Rotate keys regularly
- Use Workload Identity or other keyless authentication methods when possible
- Set `create_service_account_key = false` if you prefer to manage keys separately

