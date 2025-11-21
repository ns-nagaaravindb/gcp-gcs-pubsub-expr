# GCS Pub/Sub Notification Application

A production-grade Go application that creates files in GCS buckets and reads Pub/Sub notifications to download and display file contents.

## Features

- **Create Files**: Upload files to GCS bucket
- **Listen for Notifications**: Subscribe to Pub/Sub topic and receive GCS file creation events
- **Download & Display**: Automatically download files from notifications and print their contents
- **Production Ready**: Error handling, graceful shutdown, structured logging

## Prerequisites

1. Go 1.24+ installed
2. Google Cloud SDK configured
3. Authenticated with `gcloud auth application-default login`
4. GCP project with appropriate permissions
5. Terraform infrastructure deployed (see `../pubsub-notification-terraform/`)

## Installation

```bash
go mod download
go build -o gcs-pubsub-app ./app
```

## Usage

### Create a file in the bucket

```bash
./gcs-pubsub-app \
  -mode=create \
  -project=your-project-id \
  -bucket=test-dp-gcspubsub-bucket \
  -file=test.txt \
  -content="Hello, World!"
```

### Listen for notifications

```bash
./gcs-pubsub-app \
  -mode=listen \
  -project=your-project-id \
  -bucket=test-dp-gcspubsub-bucket \
  -topic=test-dp-gcspubsub-bucket
```

### Create file and listen (both modes)

```bash
./gcs-pubsub-app \
  -mode=both \
  -project=your-project-id \
  -bucket=test-dp-gcspubsub-bucket \
  -topic=test-dp-gcspubsub-bucket \
  -file=test.txt \
  -content="Hello, World!"
```

## Command Line Flags

- `-project`: GCP project ID (required)
- `-bucket`: GCS bucket name (default: `test-dp-gcspubsub-bucket`)
- `-topic`: Pub/Sub topic name (default: `test-dp-gcspubsub-bucket`)
- `-subscription`: Pub/Sub subscription name (default: `{topic}-subscription`)
- `-mode`: Operation mode - `create`, `listen`, or `both` (default: `listen`)
- `-file`: File name to create (required for `create` mode)
- `-content`: Content for the file (default: `Hello, World!`)

## How It Works

1. **File Creation**: When creating a file, the app uploads it to the specified GCS bucket
2. **Notification**: GCS automatically publishes a notification to the Pub/Sub topic
3. **Message Processing**: The app receives the notification message
4. **File Download**: The app extracts the file name and bucket from the notification
5. **Content Display**: The app downloads the file and prints its contents

## Authentication

The application uses Application Default Credentials (ADC). Ensure you're authenticated:

```bash
gcloud auth application-default login
```

Or set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to point to a service account key file.

## Production Considerations

- Use service account credentials in production
- Implement retry logic for transient failures
- Add metrics and monitoring
- Configure appropriate logging levels
- Handle large files efficiently (streaming)
- Implement rate limiting
- Add health checks

