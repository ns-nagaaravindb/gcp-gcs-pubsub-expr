# Manual IAM Setup Guide for GCS Pub/Sub Notifications

## Issue
Your user account doesn't have sufficient permissions to modify IAM policies programmatically. This is common in enterprise environments where IAM administration is restricted.

## ✅ **Recommended Solution: Google Cloud Console (GUI)**

### Step-by-Step Instructions:

1. **Open Google Cloud Console**
   - Go to: https://console.cloud.google.com/iam-admin/iam?project=67453487014
   - Make sure you're logged in with an account that has IAM Admin permissions

2. **Grant Access**
   - Click the **"Grant Access"** button (usually near the top of the page)

3. **Add the Service Account**
   - In the "Add principals" field, enter:
     ```
     service-67453487014@gs-project-accounts.iam.gserviceaccount.com
     ```

4. **Select Role**
   - In the "Select a role" dropdown, search for: **"Pub/Sub Publisher"**
   - Select: `Pub/Sub Publisher`

5. **Save**
   - Click **"Save"** button

6. **Verify**
   - You should see the service account listed with the "Pub/Sub Publisher" role

### Alternative: Ask an Administrator

If you don't have IAM Admin permissions, share this command with a project owner or IAM admin:

```bash
gcloud projects add-iam-policy-binding 67453487014 \
  --member="serviceAccount:service-67453487014@gs-project-accounts.iam.gserviceaccount.com" \
  --role="roles/pubsub.publisher" \
  --condition=None
```

## After Setting Up Permissions

Once the permission is granted, test the demo:

```bash
cd /Users/nagaaravindb/rs/expr
go run pubsub_notification_demo.go
```

## What This Permission Does

- **Service Account**: `service-67453487014@gs-project-accounts.iam.gserviceaccount.com`
  - This is Google's internal service account for your project's Cloud Storage
  - It's automatically created and managed by Google
  - NOT a service account you created

- **Role**: `Pub/Sub Publisher`
  - Allows the service account to publish messages to Pub/Sub topics
  - Required for GCS bucket notifications to work
  - This is a minimal, least-privilege permission

## Verification

After granting the permission, the demo should progress past:
```
Step 3: Setting up bucket notifications...
```

And show:
```
Created bucket notification: [ID] for topic: [TOPIC_NAME]
```

## Security Note

This permission is safe and follows Google's recommended practices for GCS to Pub/Sub integration. You're only granting Google's own service account the ability to publish messages that you've explicitly configured.
