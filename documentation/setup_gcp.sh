#!/bin/bash

# Google Cloud Setup Script for GCS Pub/Sub Notification Demo
# This script helps set up the required Google Cloud resources

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

# Check if required tools are installed
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    if ! command -v gcloud &> /dev/null; then
        print_error "gcloud CLI is not installed. Please install it from:"
        echo "https://cloud.google.com/sdk/docs/install"
        exit 1
    fi
    
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install it from:"
        echo "https://golang.org/downloads/"
        exit 1
    fi
    
    print_status "All prerequisites are installed"
}

# Get project information
setup_project() {
    print_header "Project Setup"
    
    if [ -z "${GOOGLE_CLOUD_PROJECT}" ]; then
        echo "Enter your Google Cloud Project ID:"
        read -r PROJECT_ID
        export GOOGLE_CLOUD_PROJECT="$PROJECT_ID"
    else
        PROJECT_ID="${GOOGLE_CLOUD_PROJECT}"
    fi
    
    print_status "Using project: $PROJECT_ID"
    
    # Set the project in gcloud
    gcloud config set project "$PROJECT_ID"
}

# Enable required APIs
enable_apis() {
    print_header "Enabling Required APIs"
    
    print_status "Enabling Cloud Storage API..."
    gcloud services enable storage.googleapis.com
    
    print_status "Enabling Pub/Sub API..."
    gcloud services enable pubsub.googleapis.com
    
    print_status "APIs enabled successfully"
}

# Set up authentication
setup_authentication() {
    print_header "Setting up Authentication"
    
    if [ -z "${GOOGLE_APPLICATION_CREDENTIALS}" ]; then
        print_warning "GOOGLE_APPLICATION_CREDENTIALS not set"
        echo "Choose authentication method:"
        echo "1) Use Application Default Credentials (recommended for testing)"
        echo "2) Use Service Account Key"
        echo "3) Skip authentication setup"
        read -r choice
        
        case $choice in
            1)
                print_status "Setting up Application Default Credentials..."
                gcloud auth application-default login
                print_status "Application Default Credentials configured"
                ;;
            2)
                echo "Enter path to your service account key file:"
                read -r KEY_PATH
                export GOOGLE_APPLICATION_CREDENTIALS="$KEY_PATH"
                print_status "Service account key configured: $KEY_PATH"
                ;;
            3)
                print_warning "Skipping authentication setup"
                ;;
            *)
                print_error "Invalid choice"
                exit 1
                ;;
        esac
    else
        print_status "Using existing service account: $GOOGLE_APPLICATION_CREDENTIALS"
    fi
}

# Create IAM service account (optional)
create_service_account() {
    print_header "Service Account Setup (Optional)"
    
    echo "Do you want to create a new service account for this demo? (y/n):"
    read -r create_sa
    
    if [ "$create_sa" = "y" ]; then
        SA_NAME="gcs-pubsub-demo-sa"
        SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
        
        print_status "Creating service account: $SA_NAME"
        gcloud iam service-accounts create "$SA_NAME" \
            --display-name="GCS Pub/Sub Demo Service Account" \
            --description="Service account for GCS Pub/Sub notification demo"
        
        print_status "Granting required roles..."
        gcloud projects add-iam-policy-binding "$PROJECT_ID" \
            --member="serviceAccount:$SA_EMAIL" \
            --role="roles/storage.admin"
        
        gcloud projects add-iam-policy-binding "$PROJECT_ID" \
            --member="serviceAccount:$SA_EMAIL" \
            --role="roles/pubsub.admin"
        
        echo "Do you want to create and download a key for this service account? (y/n):"
        read -r create_key
        
        if [ "$create_key" = "y" ]; then
            KEY_FILE="gcs-pubsub-demo-key.json"
            gcloud iam service-accounts keys create "$KEY_FILE" \
                --iam-account="$SA_EMAIL"
            
            export GOOGLE_APPLICATION_CREDENTIALS="$(pwd)/$KEY_FILE"
            print_status "Service account key saved to: $KEY_FILE"
            print_warning "Keep this key file secure and do not commit it to version control"
        fi
    else
        print_status "Skipping service account creation"
    fi
}

# Test the setup
test_setup() {
    print_header "Testing Setup"
    
    print_status "Testing authentication..."
    if gcloud auth application-default print-access-token &> /dev/null; then
        print_status "Authentication working correctly"
    else
        print_error "Authentication test failed"
        exit 1
    fi
    
    print_status "Testing Go dependencies..."
    go mod tidy
    
    print_status "Building demo programs..."
    go build -o simple_gcs_demo simple_gcs_demo.go
    go build -o pubsub_notification_demo pubsub_notification_demo.go
    
    print_status "All tests passed!"
}

# Generate environment file
generate_env_file() {
    print_header "Generating Environment File"
    
    cat > setup_env.sh << EOF
#!/bin/bash

# Auto-generated environment setup for GCS Pub/Sub Demo
# Run: source setup_env.sh

export GOOGLE_CLOUD_PROJECT="$PROJECT_ID"
EOF

    if [ -n "${GOOGLE_APPLICATION_CREDENTIALS}" ]; then
        echo "export GOOGLE_APPLICATION_CREDENTIALS=\"$GOOGLE_APPLICATION_CREDENTIALS\"" >> setup_env.sh
    fi

    cat >> setup_env.sh << 'EOF'

# Print current configuration
echo "Environment configured:"
echo "  Project ID: ${GOOGLE_CLOUD_PROJECT}"
if [ -n "${GOOGLE_APPLICATION_CREDENTIALS}" ]; then
    echo "  Service Account Key: ${GOOGLE_APPLICATION_CREDENTIALS}"
else
    echo "  Authentication: Application Default Credentials"
fi
echo ""
echo "Available demos:"
echo "  1. Simple GCS Demo: go run simple_gcs_demo.go"
echo "  2. Full Pub/Sub Notification Demo: go run pubsub_notification_demo.go"
echo "  3. Basic GCS Operations: go run gcs_demo.go"
EOF

    chmod +x setup_env.sh
    print_status "Environment file created: setup_env.sh"
}

# Main setup function
main() {
    print_header "GCS Pub/Sub Notification Demo Setup"
    echo "This script will help you set up the Google Cloud environment"
    echo "for running the GCS Pub/Sub notification demos."
    echo ""
    
    check_prerequisites
    setup_project
    enable_apis
    setup_authentication
    create_service_account
    test_setup
    generate_env_file
    
    print_header "Setup Complete!"
    print_status "To start using the demos, run:"
    echo "  source setup_env.sh"
    echo "  go run simple_gcs_demo.go"
    echo ""
    print_status "For more information, check the README files:"
    echo "  - GCS_README.md"
    echo "  - PUBSUB_NOTIFICATION_README.md"
}

# Run main function
main "$@"
