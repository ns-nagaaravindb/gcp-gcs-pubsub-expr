# Troubleshooting Guide for GCS Pub/Sub Notification Demo

## Common Error: IAM Permission Denied

### Error Message:
```
Error 403: The service account 'service-PROJECT_ID@gs-project-accounts.iam.gserviceaccount.com' does not have permission to publish messages to the Cloud Pub/Sub topic
```

### Solution:

This error occurs because the Google Cloud Storage service account needs permission to publish messages to your Pub/Sub topic.

#### Option 1: Use the Setup Script (Recommended)
```bash
./setup_iam_permissions.sh YOUR_PROJECT_ID
```

#### Option 2: Manual gcloud Command
```bash
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member=serviceAccount:service-YOUR_PROJECT_ID@gs-project-accounts.iam.gserviceaccount.com \
  --role=roles/pubsub.publisher
```

#### Option 3: Google Cloud Console (GUI)
1. Go to [Google Cloud Console IAM](https://console.cloud.google.com/iam-admin/iam)
2. Click "ADD" to add a new principal
3. Enter the service account email: `service-YOUR_PROJECT_ID@gs-project-accounts.iam.gserviceaccount.com`
4. Select role: "Pub/Sub Publisher"
5. Click "Save"

#### Option 4: Using Google Cloud Shell
If you don't have gcloud installed locally:
1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Click the Cloud Shell icon (>_) in the top right
3. Run the gcloud command from Option 2

### Verification:
After setting up the permission, run the demo again:
```bash
go run pubsub_notification_demo.go
```

## Other Common Issues:

### 1. Bucket Permission Error
**Error:** `Permission 'storage.buckets.create' denied`
**Solution:** The service account needs `Storage Admin` role or use an existing bucket.

### 2. API Not Enabled
**Error:** `API [pubsub.googleapis.com] not enabled`
**Solution:** 
```bash
gcloud services enable pubsub.googleapis.com
gcloud services enable storage.googleapis.com
```

### 3. Authentication Error
**Error:** `could not find default credentials`
**Solution:** Set up authentication:
```bash
# Option A: Service Account Key
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/key.json"

# Option B: User Credentials
gcloud auth application-default login
```

### 4. Project Not Found
**Error:** `The project ... does not exist or is not active`
**Solution:** 
```bash
gcloud config set project YOUR_PROJECT_ID
gcloud projects describe YOUR_PROJECT_ID
```

## Step-by-Step Setup for First-Time Users:

1. **Install Google Cloud CLI** (if not already installed):
   - macOS: `brew install google-cloud-sdk`
   - Windows/Linux: [Installation Guide](https://cloud.google.com/sdk/docs/install)

2. **Authenticate with Google Cloud**:
   ```bash
   gcloud auth login
   gcloud config set project YOUR_PROJECT_ID
   ```

3. **Enable Required APIs**:
   ```bash
   gcloud services enable storage.googleapis.com
   gcloud services enable pubsub.googleapis.com
   ```

4. **Set up IAM Permissions**:
   ```bash
   ./setup_iam_permissions.sh YOUR_PROJECT_ID
   ```

5. **Run the Demo**:
   ```bash
   go run pubsub_notification_demo.go
   ```

## Understanding the Service Account

The service account `service-YOUR_PROJECT_ID@gs-project-accounts.iam.gserviceaccount.com` is:
- **Automatically created by Google Cloud** for each project
- **Used by Google Cloud Storage** to perform operations on behalf of your project
- **Required for GCS notifications** to publish messages to Pub/Sub topics
- **Different from your personal service accounts** - this is a Google-managed account

## Testing the Setup

After fixing permissions, you should see output like:
```
Step 3: Setting up bucket notifications...
Created bucket notification: 123 for topic: projects/YOUR_PROJECT_ID/topics/gcs-notifications-test-dp

Step 4: Starting message listeners...
Starting to listen for messages on subscription: gcs-notifications-sub-test-dp
Starting DLQ listener on subscription: gcs-notifications-dlq-sub-test-dp

Step 5: Creating test files to trigger notifications...
Created test file: gs://test-dp-gcs/test-success.txt
```

## Getting Help

If you continue to have issues:
1. Check the [Google Cloud Storage Notifications Documentation](https://cloud.google.com/storage/docs/pubsub-notifications)
2. Verify your project billing is enabled
3. Ensure you have the necessary IAM roles on your user account
4. Check Google Cloud Console logs for additional error details
