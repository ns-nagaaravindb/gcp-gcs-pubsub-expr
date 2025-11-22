# Cross-Project Pub/Sub Consumer Setup

This guide explains how to set up a Pub/Sub consumer in a different GCP project from the publisher.

## Overview

This setup allows you to:
- Create GCS bucket and Pub/Sub topic in **Publisher Project** (Project A)
- Create a Pub/Sub subscription in **Consumer Project** (Project B)
- Enable the consumer project to receive notifications when files are created in the publisher's GCS bucket

## Architecture

```
┌─────────────────────────────────┐
│   Publisher Project (Project A) │
│                                 │
│  ┌──────────┐                  │
│  │   GCS    │                  │
│  │  Bucket  │──────┐           │
│  └──────────┘      │           │
│                    │           │
│  ┌─────────────────▼────────┐  │
│  │   Pub/Sub Topic          │  │
│  │  (test-dp-gcspubsub...)  │  │
│  └──────────────────────────┘  │
│           │                     │
└───────────┼─────────────────────┘
            │
            │ IAM Permission
            │ (pubsub.subscriber)
            │
┌───────────▼─────────────────────┐
│   Consumer Project (Project B)  │
│                                 │
│  ┌──────────────────────────┐  │
│  │   Pub/Sub Subscription   │  │
│  │  (..consumer-subscription)│  │
│  └──────────────────────────┘  │
│           │                     │
│  ┌────────▼────────┐           │
│  │  Go Application │           │
│  │   (Listener)    │           │
│  └─────────────────┘           │
└─────────────────────────────────┘
```

## Prerequisites

1. **Two GCP Projects**:
   - Publisher Project (where GCS bucket and topic will be created)
   - Consumer Project (where subscription will be created)

2. **Required Permissions**:
   - Publisher Project: Editor or Owner role (to create GCS, Pub/Sub resources)
   - Consumer Project: Pub/Sub Admin role (to create subscription)

3. **Tools**:
   - Terraform >= 1.0
   - Go >= 1.21
   - gcloud CLI (authenticated)

## Setup Instructions

### Step 1: Authenticate with GCP

```bash
# Authenticate with gcloud
gcloud auth application-default login

# Set default project (optional)
gcloud config set project YOUR-PUBLISHER-PROJECT-ID
```

### Step 2: Configure Terraform Variables

Create a `terraform.tfvars` file:

```hcl
# Publisher project configuration
project_id = "your-publisher-project-id"
region     = "us-central1"

# Bucket and topic names
bucket_name = "test-dp-gcspubsub-bucket"
topic_name  = "test-dp-gcspubsub-bucket"

# Cross-project consumer configuration
create_cross_project_subscription = true
consumer_project_id = "your-consumer-project-id"

labels = {
  environment = "test"
  managed_by  = "terraform"
}
```

### Step 3: Deploy Infrastructure

#### Using Terraform Directly

```bash
cd pubsub-notification-terraform

# Initialize Terraform
terraform init

# Plan the deployment
terraform plan

# Apply the configuration
terraform apply
```

#### Using Makefile (Recommended)

```bash
# Apply infrastructure with cross-project setup
make cross-project-apply \
  PROJECT_ID=your-publisher-project \
  CONSUMER_PROJECT_ID=your-consumer-project

# Or use environment variables
export PROJECT_ID=your-publisher-project
export CONSUMER_PROJECT_ID=your-consumer-project
make cross-project-apply
```

### Step 4: Test the Setup

#### Option 1: Using Makefile

```bash
# Full test: Create file and listen for notifications
make cross-project-test \
  PROJECT_ID=your-publisher-project \
  CONSUMER_PROJECT_ID=your-consumer-project

# Or just listen for notifications
make cross-project-listen \
  PROJECT_ID=your-publisher-project \
  CONSUMER_PROJECT_ID=your-consumer-project
```

#### Option 2: Using Go Application Directly

```bash
# Build the application
cd app
go build -o ../gcs-pubsub-app .

# Listen for notifications (in consumer project)
../gcs-pubsub-app \
  -mode=listen \
  -project=your-publisher-project \
  -consumer-project=your-consumer-project \
  -bucket=test-dp-gcspubsub-bucket \
  -topic=test-dp-gcspubsub-bucket

# In another terminal, create a test file
../gcs-pubsub-app \
  -mode=create \
  -project=your-publisher-project \
  -bucket=test-dp-gcspubsub-bucket \
  -file=test.txt \
  -content="Hello from cross-project setup!"
```

#### Option 3: Using Environment Variables

```bash
# Set environment variables
export GCP_PROJECT_ID=your-publisher-project
export GCP_CONSUMER_PROJECT_ID=your-consumer-project

# Run the application
./gcs-pubsub-app -mode=both -file=test.txt
```

## Makefile Targets

### Standard Targets (Same Project)

- `make terraform-apply PROJECT_ID=your-project` - Deploy infrastructure in single project
- `make app-listen PROJECT_ID=your-project` - Listen for notifications
- `make app-test PROJECT_ID=your-project` - Create file and listen

### Cross-Project Targets

- `make cross-project-apply PROJECT_ID=pub-project CONSUMER_PROJECT_ID=con-project` - Deploy cross-project infrastructure
- `make cross-project-listen PROJECT_ID=pub-project CONSUMER_PROJECT_ID=con-project` - Listen in consumer project
- `make cross-project-test PROJECT_ID=pub-project CONSUMER_PROJECT_ID=con-project` - Full cross-project test
- `make cross-project-demo PROJECT_ID=pub-project CONSUMER_PROJECT_ID=con-project` - Complete demo
- `make cross-project-destroy PROJECT_ID=pub-project CONSUMER_PROJECT_ID=con-project` - Destroy infrastructure

### Information Targets

- `make info` - Show current configuration
- `make terraform-output` - Show Terraform outputs

## Configuration Options

### Terraform Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `project_id` | Publisher project ID | - | Yes |
| `consumer_project_id` | Consumer project ID | "" | No* |
| `create_cross_project_subscription` | Enable cross-project subscription | false | No |
| `region` | GCP region | us-central1 | No |
| `bucket_name` | GCS bucket name | test-dp-gcspubsub-bucket | No |
| `topic_name` | Pub/Sub topic name | test-dp-gcspubsub-bucket | No |

*Required if `create_cross_project_subscription = true`

### Application Flags

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `-project` | Publisher project ID | - | Yes* |
| `-consumer-project` | Consumer project ID | Same as publisher | No |
| `-bucket` | GCS bucket name | test-dp-gcspubsub-bucket | No |
| `-topic` | Pub/Sub topic name | test-dp-gcspubsub-bucket | No |
| `-subscription` | Subscription name | Auto-generated | No |
| `-mode` | Operation mode | listen | No |

*Can be set via `GCP_PROJECT_ID` environment variable

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GCP_PROJECT_ID` | Publisher project ID |
| `GCP_CONSUMER_PROJECT_ID` | Consumer project ID |

## How It Works

1. **Terraform creates resources**:
   - GCS bucket in Publisher Project
   - Pub/Sub topic in Publisher Project
   - Pub/Sub subscription in Publisher Project (standard)
   - Pub/Sub subscription in Consumer Project (if cross-project enabled)
   - IAM permissions for GCS to publish to topic
   - IAM permissions for Consumer to subscribe to topic

2. **File Upload Triggers Notification**:
   - File uploaded to GCS bucket
   - GCS automatically publishes notification to Pub/Sub topic
   - Notification is available to all subscriptions

3. **Consumer Receives Notification**:
   - Go application in Consumer Project listens to its subscription
   - Receives notification with file metadata
   - Downloads and processes the file from GCS bucket

## IAM Permissions Explained

### Automatic IAM Bindings

Terraform automatically creates these IAM bindings:

1. **GCS Service Account → Pub/Sub Topic**:
   - Role: `roles/pubsub.publisher`
   - Allows GCS to publish notifications

2. **Consumer Project → Pub/Sub Topic**:
   - Role: `roles/pubsub.subscriber`
   - Allows consumer subscription to pull messages

### Required User Permissions

**Publisher Project** (to deploy infrastructure):
- `storage.buckets.create`
- `pubsub.topics.create`
- `pubsub.subscriptions.create`
- `resourcemanager.projects.setIamPolicy`

**Consumer Project** (to create subscription):
- `pubsub.subscriptions.create`
- `pubsub.subscriptions.get`

**Both Projects** (to run application):
- `pubsub.subscriptions.consume`
- `storage.objects.get`
- `storage.objects.list`

## Troubleshooting

### Issue: Subscription not receiving messages

**Solution**: Check IAM permissions
```bash
# Verify topic IAM policy
gcloud pubsub topics get-iam-policy test-dp-gcspubsub-bucket \
  --project=your-publisher-project

# Verify subscription exists in consumer project
gcloud pubsub subscriptions list \
  --project=your-consumer-project
```

### Issue: Permission denied when accessing GCS bucket

**Solution**: Ensure consumer project has access to the bucket
```bash
# Grant read access to consumer service account
gsutil iam ch serviceAccount:YOUR-SERVICE-ACCOUNT@your-consumer-project.iam.gserviceaccount.com:objectViewer \
  gs://test-dp-gcspubsub-bucket
```

### Issue: Terraform fails with "already exists" error

**Solution**: Import existing resources or destroy them first
```bash
# Destroy existing resources
make cross-project-destroy \
  PROJECT_ID=your-publisher-project \
  CONSUMER_PROJECT_ID=your-consumer-project

# Then reapply
make cross-project-apply \
  PROJECT_ID=your-publisher-project \
  CONSUMER_PROJECT_ID=your-consumer-project
```

## Cleanup

### Destroy Cross-Project Infrastructure

```bash
# Using Makefile
make cross-project-destroy \
  PROJECT_ID=your-publisher-project \
  CONSUMER_PROJECT_ID=your-consumer-project

# Using Terraform directly
cd pubsub-notification-terraform
terraform destroy \
  -var="project_id=your-publisher-project" \
  -var="consumer_project_id=your-consumer-project" \
  -var="create_cross_project_subscription=true"
```

## Best Practices

1. **Use Service Accounts**: In production, use dedicated service accounts with minimal permissions
2. **Enable Logging**: Enable Pub/Sub and GCS audit logs for monitoring
3. **Set Quotas**: Configure appropriate quotas for Pub/Sub subscriptions
4. **Monitor Costs**: Cross-project data transfer may incur additional costs
5. **Use Private Service Connect**: For sensitive data, use Private Service Connect
6. **Implement Dead Letter Queues**: Add DLQ for failed message processing
7. **Use Labels**: Tag resources with appropriate labels for cost tracking

## Example: Production Setup

```hcl
# terraform.tfvars for production
project_id = "prod-publisher-project"
consumer_project_id = "prod-consumer-project"
region = "us-central1"

bucket_name = "prod-data-ingestion-bucket"
topic_name = "prod-data-ingestion-notifications"

create_cross_project_subscription = true

# Stricter retention and backoff settings
message_retention_duration = "259200s"  # 3 days
ack_deadline_seconds = 60
minimum_backoff = "30s"
maximum_backoff = "600s"

labels = {
  environment = "production"
  team        = "data-engineering"
  managed_by  = "terraform"
  cost_center = "data-platform"
}
```

## Additional Resources

- [GCP Pub/Sub Documentation](https://cloud.google.com/pubsub/docs)
- [GCS Notifications Documentation](https://cloud.google.com/storage/docs/pubsub-notifications)
- [Cross-project Pub/Sub](https://cloud.google.com/pubsub/docs/access-control)
- [Terraform Google Provider](https://registry.terraform.io/providers/hashicorp/google/latest/docs)

