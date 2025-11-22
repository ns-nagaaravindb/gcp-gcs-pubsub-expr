#!/bin/bash

# Test script for cross-project Pub/Sub consumer setup
# This script verifies that the cross-project consumer can receive notifications

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PUBLISHER_PROJECT="${1:-}"
CONSUMER_PROJECT="${2:-}"
BUCKET_NAME="${3:-test-dp-gcspubsub-bucket}"
TOPIC_NAME="${4:-test-dp-gcspubsub-bucket}"
TEST_FILE="test-cross-project-$(date +%s).txt"

# Functions
print_header() {
    echo -e "\n${GREEN}=====================================${NC}"
    echo -e "${GREEN}$1${NC}"
    echo -e "${GREEN}=====================================${NC}\n"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

check_prerequisites() {
    print_header "Checking Prerequisites"
    
    # Check if projects are provided
    if [ -z "$PUBLISHER_PROJECT" ] || [ -z "$CONSUMER_PROJECT" ]; then
        print_error "Usage: $0 <publisher-project> <consumer-project> [bucket-name] [topic-name]"
        print_info "Example: $0 my-publisher-project my-consumer-project"
        exit 1
    fi
    
    # Check if gcloud is installed
    if ! command -v gcloud &> /dev/null; then
        print_error "gcloud CLI is not installed"
        exit 1
    fi
    print_success "gcloud CLI found"
    
    # Check authentication
    if ! gcloud auth application-default print-access-token &> /dev/null; then
        print_error "Not authenticated. Run: gcloud auth application-default login"
        exit 1
    fi
    print_success "Authenticated with gcloud"
    
    print_success "All prerequisites met"
}

verify_infrastructure() {
    print_header "Verifying Infrastructure"
    
    # Check if bucket exists
    print_info "Checking GCS bucket in publisher project..."
    if gsutil ls -p "$PUBLISHER_PROJECT" | grep -q "gs://${BUCKET_NAME}/"; then
        print_success "Bucket exists: gs://${BUCKET_NAME}"
    else
        print_error "Bucket not found: gs://${BUCKET_NAME}"
        exit 1
    fi
    
    # Check if topic exists
    print_info "Checking Pub/Sub topic in publisher project..."
    if gcloud pubsub topics describe "$TOPIC_NAME" --project="$PUBLISHER_PROJECT" &> /dev/null; then
        print_success "Topic exists: $TOPIC_NAME"
    else
        print_error "Topic not found: $TOPIC_NAME"
        exit 1
    fi
    
    # Check if subscription exists in consumer project
    print_info "Checking Pub/Sub subscription in consumer project..."
    SUBSCRIPTION_NAME="${TOPIC_NAME}-consumer-subscription"
    if gcloud pubsub subscriptions describe "$SUBSCRIPTION_NAME" --project="$CONSUMER_PROJECT" &> /dev/null; then
        print_success "Subscription exists: $SUBSCRIPTION_NAME"
    else
        print_error "Subscription not found: $SUBSCRIPTION_NAME"
        print_info "Run: make cross-project-apply PROJECT_ID=$PUBLISHER_PROJECT CONSUMER_PROJECT_ID=$CONSUMER_PROJECT"
        exit 1
    fi
}

test_notification() {
    print_header "Testing Cross-Project Notification"
    
    # Create a test file in the bucket
    print_info "Creating test file: gs://${BUCKET_NAME}/${TEST_FILE}"
    echo "Test file created at $(date)" | gsutil cp - "gs://${BUCKET_NAME}/${TEST_FILE}"
    print_success "Test file created"
    
    # Wait for notification to propagate
    print_info "Waiting for notification to propagate (5 seconds)..."
    sleep 5
    
    # Pull message from subscription
    print_info "Pulling message from subscription in consumer project..."
    SUBSCRIPTION_NAME="${TOPIC_NAME}-consumer-subscription"
    MESSAGE=$(gcloud pubsub subscriptions pull "$SUBSCRIPTION_NAME" \
        --project="$CONSUMER_PROJECT" \
        --limit=1 \
        --auto-ack \
        --format=json 2>/dev/null || echo "[]")
    
    if [ "$MESSAGE" != "[]" ] && [ -n "$MESSAGE" ]; then
        print_success "Message received in consumer project!"
        echo "$MESSAGE" | jq -r '.[0].message.data' | base64 -d | jq . || true
        print_success "Cross-project setup is working correctly!"
    else
        print_error "No message received"
        print_info "This might be because:"
        print_info "  1. The notification hasn't propagated yet (try again in a few seconds)"
        print_info "  2. The message was already pulled by another consumer"
        print_info "  3. IAM permissions are not set correctly"
    fi
}

cleanup_test_file() {
    print_header "Cleanup"
    print_info "Removing test file: gs://${BUCKET_NAME}/${TEST_FILE}"
    gsutil rm "gs://${BUCKET_NAME}/${TEST_FILE}" 2>/dev/null || true
    print_success "Test file removed"
}

show_summary() {
    print_header "Test Summary"
    echo "Publisher Project: $PUBLISHER_PROJECT"
    echo "Consumer Project:  $CONSUMER_PROJECT"
    echo "Bucket:            $BUCKET_NAME"
    echo "Topic:             $TOPIC_NAME"
    echo "Subscription:      ${TOPIC_NAME}-consumer-subscription"
    echo ""
    print_success "Cross-project test completed!"
    echo ""
    print_info "To manually test, run:"
    echo "  # Listen for notifications:"
    echo "  make cross-project-listen PROJECT_ID=$PUBLISHER_PROJECT CONSUMER_PROJECT_ID=$CONSUMER_PROJECT"
    echo ""
    echo "  # Create a test file:"
    echo "  make app-create-file PROJECT_ID=$PUBLISHER_PROJECT"
}

# Main execution
main() {
    echo -e "${GREEN}"
    echo "╔════════════════════════════════════════════════════╗"
    echo "║   Cross-Project Pub/Sub Consumer Test Script      ║"
    echo "╚════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    check_prerequisites
    verify_infrastructure
    test_notification
    cleanup_test_file
    show_summary
}

# Run main function
main

