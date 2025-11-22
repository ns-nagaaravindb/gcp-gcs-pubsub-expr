# GCS Pub/Sub Notification Project

## ⭐ NEW: Cross-Project Consumer Support

This project now supports **cross-project Pub/Sub consumers**! You can now:
- Create GCS bucket and Pub/Sub topic in one project (Publisher)
- Consume notifications in a different project (Consumer)

See [Cross-Project Setup Guide](pubsub-notification-terraform/CROSS_PROJECT_SETUP.md) for details.

---

export projectID="data-qe-da7e1252"

Before terraform destroy 
bash -c 'gsutil -m rm "gs://test-dp-gcspubsub-bucket/*"' 

## Project Structure

```
.
├── pubsub-notification-terraform/  # Terraform module for infrastructure
│   ├── main.tf                      # Main Terraform configuration
│   ├── variables.tf                 # Variable definitions
│   ├── outputs.tf                   # Output definitions
│   └── README.md                    # Terraform module documentation
├── app/                             # Go application
│   ├── main.go                      # Application source code
│   └── README.md                    # Application documentation
└── Makefile                         # Build and deployment automation
```

## Quick Start

### Prerequisites

1. Install [Google Cloud SDK](https://cloud.google.com/sdk/docs/install)
2. Install [Terraform](https://www.terraform.io/downloads)
3. Install [Go](https://go.dev/dl/) 1.24+
4. Authenticate: `gcloud auth application-default login`

### Setup

1. **Set your project ID:**
   ```bash
   export PROJECT_ID=your-gcp-project-id
   ```

2. **Create infrastructure:**
   ```bash
   make terraform-apply PROJECT_ID=$PROJECT_ID
   ```

3. **Build application:**
   ```bash
   make app-build
   ```

4. **Run demo:**
   ```bash
   make demo PROJECT_ID=$PROJECT_ID
   ```

## Makefile Commands

Run `make help` to see all available commands:

- **Infrastructure:**
  - `make terraform-init` - Initialize Terraform
  - `make terraform-plan` - Plan infrastructure changes
  - `make terraform-apply` - Create infrastructure
  - `make terraform-destroy` - Delete infrastructure

- **Application:**
  - `make app-build` - Build the Go application
  - `make app-create-file` - Create a test file in bucket
  - `make app-listen` - Listen for Pub/Sub notifications
  - `make app-test` - Create file and listen for notification

- **Utilities:**
  - `make check-deps` - Check all dependencies
  - `make clean` - Clean build artifacts
  - `make demo` - Full demo workflow

## Documentation

- [Terraform Module README](pubsub-notification-terraform/README.md)
- [Application README](app/README.md)

## Configuration

Default values can be overridden via Makefile variables:

```bash
make terraform-apply PROJECT_ID=my-project BUCKET_NAME=my-bucket TOPIC_NAME=my-topic
```

## License

This project is provided as-is for demonstration purposes.

