# GCS Pub/Sub Notification Demo

This Go program demonstrates a complete Google Cloud Storage (GCS) to Pub/Sub notification system with Dead Letter Topic (DLT) support. It shows how to:

1. Set up Pub/Sub topics and subscriptions with Dead Letter Queues
2. Configure GCS bucket notifications to publish to Pub/Sub
3. Process file upload notifications in real-time
4. Handle processing errors with DLT for failed messages
5. Automatically retry failed messages with exponential backoff

## Architecture Overview

```
GCS Bucket → Pub/Sub Topic → Subscription → Message Processor
                                ↓ (on failure)
                         Dead Letter Topic → DLQ Subscription → Error Handler
```

## Features

- **Real-time Processing**: Automatically processes files as they are uploaded to GCS
- **Error Handling**: Failed messages are automatically sent to Dead Letter Queue
- **Retry Logic**: Built-in retry mechanism with configurable attempts
- **Concurrent Processing**: Handles multiple notifications simultaneously
- **Comprehensive Logging**: Detailed logging for monitoring and debugging
- **Graceful Shutdown**: Proper cleanup of resources on exit

## Prerequisites

1. **Google Cloud Project** with the following APIs enabled:
   - Cloud Storage API
   - Cloud Pub/Sub API

2. **IAM Permissions**: Your service account needs these roles:
   - `Pub/Sub Admin` (to create topics and subscriptions)
   - `Storage Admin` (to create buckets and configure notifications)
   - `Storage Object Viewer` (to read uploaded files)

3. **Authentication**: Set up authentication using one of these methods:
   
   **Option A: Service Account Key**
   ```bash
   export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
   ```
   
   **Option B: Application Default Credentials**
   ```bash
   gcloud auth application-default login
   ```

4. **Environment Variable**:
   ```bash
   export GOOGLE_CLOUD_PROJECT="your-project-id"
   ```

## Quick Start

1. **Set up your environment**:
   ```bash
   export GOOGLE_CLOUD_PROJECT="your-project-id"
   export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
   ```

2. **Set up IAM permissions** (IMPORTANT):
   ```bash
   ./setup_iam_permissions.sh your-project-id
   ```

3. **Navigate to the project directory**:
   ```bash
   cd /Users/nagaaravindb/rs/expr
   ```

4. **Run the demo**:
   ```bash
   go run pubsub_notification_demo.go
   ```

## Important: IAM Permissions

Before running the demo, you **must** grant the Google Cloud Storage service account permission to publish to Pub/Sub topics. This is required for GCS bucket notifications to work.

The program will show you the exact command to run if permissions are missing.

## What the Program Does

### Setup Phase
1. **Creates Pub/Sub Resources**:
   - Main topic for GCS notifications
   - Subscription with Dead Letter Policy
   - Dead Letter Topic for failed messages
   - Dead Letter Subscription for error handling

2. **Creates GCS Bucket**: Temporary bucket for demonstration

3. **Configures Bucket Notifications**: Sets up automatic notifications for:
   - `OBJECT_FINALIZE` (file uploads)
   - `OBJECT_DELETE` (file deletions)

### Processing Phase
4. **Starts Message Listeners**:
   - Main listener for processing notifications
   - DLQ listener for handling failed messages

5. **Creates Test Files**: Uploads sample files to trigger notifications:
   - `test-success.txt` - Will process successfully
   - `test-error.txt` - Will simulate processing error
   - `data/sample.json` - JSON data file
   - `logs/app.log` - Log file

6. **Processes Notifications**: For each uploaded file:
   - Parses the GCS notification message
   - Downloads and reads the file content
   - Displays file information and content preview
   - Handles errors and retries as configured

### Cleanup Phase
7. **Resource Cleanup**:
   - Deletes uploaded test files
   - Removes the temporary bucket
   - Gracefully shuts down listeners

## Sample Output

```
=======================================================================
Starting GCS Pub/Sub Notification Demo
=======================================================================
Project ID: your-project-id
Bucket: gcs-notification-demo-1630123456
Topic: gcs-notifications
Subscription: gcs-notifications-sub
DLQ Topic: gcs-notifications-dlq
DLQ Subscription: gcs-notifications-dlq-sub

Step 1: Setting up Pub/Sub resources...
Created topic: gcs-notifications
Created DLQ topic: gcs-notifications-dlq
Created subscription: gcs-notifications-sub with DLQ: gcs-notifications-dlq
Created DLQ subscription: gcs-notifications-dlq-sub

Step 2: Creating bucket...
Created bucket: gcs-notification-demo-1630123456

Step 3: Setting up bucket notifications...
Created bucket notification: abc123 for topic: projects/your-project-id/topics/gcs-notifications

Step 4: Starting message listeners...
Starting to listen for messages on subscription: gcs-notifications-sub
Starting DLQ listener on subscription: gcs-notifications-dlq-sub

Step 5: Creating test files to trigger notifications...
Created test file: gs://gcs-notification-demo-1630123456/test-success.txt
Created test file: gs://gcs-notification-demo-1630123456/test-error.txt
Created test file: gs://gcs-notification-demo-1630123456/data/sample.json
Created test file: gs://gcs-notification-demo-1630123456/logs/app.log

--- New notification received ---
Message ID: 123456789
Publish time: 2023-09-05T10:30:45Z
Event type: OBJECT_FINALIZE
Bucket: gcs-notification-demo-1630123456
File: test-success.txt
Size: 89 bytes
Content type: text/plain
Processing file: gs://gcs-notification-demo-1630123456/test-success.txt
File content preview (first 1024 bytes):
This is a successful test file.
Content will be processed successfully.
File size: 89 bytes
Content type: text/plain
Created: 2023-09-05 10:30:45.123456 +0000 UTC
Successfully processed notification

--- New notification received ---
Message ID: 123456790
Publish time: 2023-09-05T10:30:46Z
Event type: OBJECT_FINALIZE
Bucket: gcs-notification-demo-1630123456
File: test-error.txt
Size: 76 bytes
Content type: text/plain
Processing file: gs://gcs-notification-demo-1630123456/test-error.txt
Error processing file: simulated processing error for file: test-error.txt

--- DLQ message received ---
Message ID: 123456790
Publish time: 2023-09-05T10:30:46Z
DLQ - Failed processing file: gs://gcs-notification-demo-1630123456/test-error.txt
DLQ - Event type: OBJECT_FINALIZE

Press Ctrl+C to stop listening and cleanup resources
```

## Configuration Options

You can modify these variables in the code:

```go
// Retry configuration
maxRetries := 3

// Subscription settings
sub.ReceiveSettings.MaxConcurrentHandlers = 10
sub.ReceiveSettings.MaxOutstandingMessages = 100

// Notification events
EventTypes: []string{
    "OBJECT_FINALIZE", // Object created
    "OBJECT_DELETE",   // Object deleted
    // "OBJECT_METADATA_UPDATE", // Metadata updated
}
```

## Error Handling Strategy

### Built-in Retry Logic
- Messages are automatically retried up to `maxRetries` times
- Failed messages are moved to Dead Letter Queue
- Exponential backoff prevents overwhelming the system

### Dead Letter Queue Processing
- Failed messages are captured for analysis
- Can implement custom recovery logic
- Prevents loss of important notifications

### Custom Error Scenarios
The demo includes simulated errors:
- Files with "error" in the name trigger processing failures
- Demonstrates DLQ functionality
- Shows how to handle different error types

## Production Considerations

### Security
- Use Workload Identity for GKE deployments
- Follow principle of least privilege for IAM roles
- Encrypt sensitive data in messages

### Monitoring
- Set up Cloud Monitoring alerts for:
  - High message processing latency
  - Increasing DLQ message count
  - Subscription backlog growth

### Scaling
- Adjust `MaxConcurrentHandlers` based on processing requirements
- Use regional subscriptions for high availability
- Consider using Cloud Run or GKE for auto-scaling

### Resource Management
- Set appropriate message retention periods
- Monitor storage costs for large buckets
- Implement message deduplication if needed

## Troubleshooting

### Common Issues

1. **Permission Denied**:
   ```
   Error: failed to create topic: rpc error: code = PermissionDenied
   ```
   **Solution**: Ensure service account has `Pub/Sub Admin` role

2. **Bucket Notification Setup Failed**:
   ```
   Error: failed to add notification: googleapi: Error 400: Invalid topic name
   ```
   **Solution**: Verify topic exists and project ID is correct

3. **Messages Not Received**:
   - Check if bucket notifications are configured correctly
   - Verify subscription is pulling messages
   - Ensure files are actually being uploaded

4. **DLQ Not Working**:
   - Verify Dead Letter Policy is configured
   - Check DLQ topic permissions
   - Ensure max delivery attempts is set correctly

### Debugging Commands

```bash
# List Pub/Sub topics
gcloud pubsub topics list

# List subscriptions
gcloud pubsub subscriptions list

# Check bucket notifications
gsutil notification list gs://your-bucket-name

# View subscription details
gcloud pubsub subscriptions describe your-subscription-name
```

## Advanced Features

### Custom Message Processing
You can extend the `ProcessFile` function to:
- Extract metadata from different file types
- Integrate with other Google Cloud services
- Implement custom business logic

### Batch Processing
For high-volume scenarios, consider:
- Batching multiple files together
- Using Cloud Dataflow for stream processing
- Implementing message aggregation

### Multi-Region Setup
For global applications:
- Use regional Pub/Sub topics
- Implement cross-region replication
- Set up disaster recovery procedures

## Related Documentation

- [Google Cloud Storage Notifications](https://cloud.google.com/storage/docs/pubsub-notifications)
- [Pub/Sub Dead Letter Topics](https://cloud.google.com/pubsub/docs/dead-letter-topics)
- [Go Client Libraries](https://cloud.google.com/go/docs/reference)
- [IAM Roles for Storage](https://cloud.google.com/storage/docs/access-control/iam-roles)
