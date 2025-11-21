#!/bin/bash

# Google Cloud IAM Setup Script for GCS Pub/Sub Notifications
# This script sets up the required IAM permissions for GCS to publish to Pub/Sub

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Get project ID
if [ -z "$1" ]; then
    echo "Usage: $0 <PROJECT_ID>"
    echo "Example: $0 67453487014"
    exit 1
fi

PROJECT_ID="$1"
GCS_SERVICE_ACCOUNT="service-${PROJECT_ID}@gs-project-accounts.iam.gserviceaccount.com"

print_header "Setting up IAM Permissions for GCS Pub/Sub Notifications"
echo "Project ID: $PROJECT_ID"
echo "GCS Service Account: $GCS_SERVICE_ACCOUNT"
echo ""

# Check if gcloud is available
if ! command -v gcloud &> /dev/null; then
    print_error "gcloud CLI is not installed or not in PATH"
    echo ""
    echo "Please install Google Cloud CLI from:"
    echo "https://cloud.google.com/sdk/docs/install"
    echo ""
    echo "Or manually set up the IAM permission in Google Cloud Console:"
    echo "1. Go to https://console.cloud.google.com/iam-admin/iam"
    echo "2. Find the service account: $GCS_SERVICE_ACCOUNT"
    echo "3. Add the 'Pub/Sub Publisher' role"
    exit 1
fi

print_success "gcloud CLI found"

# Set the project
echo "Setting active project..."
gcloud config set project "$PROJECT_ID"

# Add IAM policy binding
echo "Adding IAM policy binding..."
if gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:$GCS_SERVICE_ACCOUNT" \
    --role="roles/pubsub.publisher" \
    --condition=None \
    --quiet 2>/dev/null; then
    
    print_success "IAM permission granted successfully!"
else
    print_warning "Failed to set IAM policy automatically"
    echo ""
    echo "This could be due to insufficient permissions. Please try one of these alternatives:"
    echo ""
    echo "1. Use a service account with Project IAM Admin role:"
    echo "   gcloud auth activate-service-account --key-file=/path/to/admin-key.json"
    echo ""
    echo "2. Ask a project owner/admin to run this command:"
    echo "   gcloud projects add-iam-policy-binding $PROJECT_ID \\"
    echo "     --member=\"serviceAccount:$GCS_SERVICE_ACCOUNT\" \\"
    echo "     --role=\"roles/pubsub.publisher\" \\"
    echo "     --condition=None"
    echo ""
    echo "3. Use Google Cloud Console (recommended for users without admin access):"
    echo "   a) Go to https://console.cloud.google.com/iam-admin/iam?project=$PROJECT_ID"
    echo "   b) Click 'Grant Access'"
    echo "   c) Add principal: $GCS_SERVICE_ACCOUNT"
    echo "   d) Select role: Pub/Sub Publisher"
    echo "   e) Click 'Save'"
    echo ""
    echo "4. If you have access to a different Google account with admin rights:"
    echo "   gcloud auth login"
    echo "   # Then re-run this script"
    exit 1
fi
echo ""
echo "The GCS service account can now publish messages to Pub/Sub topics."
echo "You can proceed with running the notification demo."
