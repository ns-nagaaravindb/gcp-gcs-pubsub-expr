#!/usr/bin/env python3
"""
GCS Pub/Sub Notification Setup Script
Creates a Pub/Sub notification configuration for a GCS bucket to trigger on file creation events.
"""

from google.cloud import storage
from google.cloud import pubsub_v1
import argparse
import os


def grant_gcs_pubsub_permission(project_id: str, topic_name: str, credentials_path: str = None) -> None:
    """
    Grant the GCS service account permission to publish to the Pub/Sub topic.
    
    Args:
        project_id: GCP project ID
        topic_name: Name of the Pub/Sub topic
        credentials_path: Path to service account key file
    """
    if credentials_path:
        os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = credentials_path
    
    publisher = pubsub_v1.PublisherClient()
    topic_path = publisher.topic_path(project_id, topic_name)
    
    # GCS service account format
    gcs_service_account = f"service-{project_id}@gs-project-accounts.iam.gserviceaccount.com"
    
    # Get existing IAM policy
    policy = publisher.get_iam_policy(request={"resource": topic_path})
    
    # Add publisher binding
    binding = {
        "role": "roles/pubsub.publisher",
        "members": {f"serviceAccount:{gcs_service_account}"}
    }
    
    # Check if binding already exists
    binding_exists = False
    for existing_binding in policy.bindings:
        if existing_binding.role == "roles/pubsub.publisher":
            if f"serviceAccount:{gcs_service_account}" in existing_binding.members:
                binding_exists = True
            else:
                existing_binding.members.append(f"serviceAccount:{gcs_service_account}")
                binding_exists = True
            break
    
    if not binding_exists:
        policy.bindings.append(
            type(policy.bindings[0])(
                role="roles/pubsub.publisher",
                members=[f"serviceAccount:{gcs_service_account}"]
            )
        )
    
    # Update the IAM policy
    publisher.set_iam_policy(request={"resource": topic_path, "policy": policy})
    print(f"✓ Granted Pub/Sub Publisher role to GCS service account: {gcs_service_account}")


def create_pubsub_topic_if_not_exists(project_id: str, topic_name: str, credentials_path: str = None) -> str:
    """
    Create a Pub/Sub topic if it doesn't already exist.
    
    Args:
        project_id: GCP project ID
        topic_name: Name of the Pub/Sub topic
        credentials_path: Path to service account key file
        
    Returns:
        Full topic path
    """
    if credentials_path:
        os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = credentials_path
    
    publisher = pubsub_v1.PublisherClient()
    topic_path = publisher.topic_path(project_id, topic_name)
    
    try:
        publisher.get_topic(request={"topic": topic_path})
        print(f"Topic already exists: {topic_path}")
    except Exception:
        topic = publisher.create_topic(request={"name": topic_path})
        print(f"Created topic: {topic.name}")
    
    return topic_path


def create_bucket_notification(
    project_id: str,
    bucket_name: str,
    topic_name: str,
    credentials_path: str = None
) -> None:
    """
    Create a Pub/Sub notification configuration for a GCS bucket.
    Triggers on OBJECT_FINALIZE events (file creation/upload completion).
    
    Args:
        project_id: GCP project ID
        bucket_name: Name of the GCS bucket
        topic_name: Name of the Pub/Sub topic
        credentials_path: Path to service account key file
    """
    # Set credentials if provided
    if credentials_path:
        os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = credentials_path
    
    # Initialize GCS client
    storage_client = storage.Client(project=project_id)
    bucket = storage_client.bucket(bucket_name)
    
    # Create topic if it doesn't exist
    topic_path = create_pubsub_topic_if_not_exists(project_id, topic_name, credentials_path)
    
    # Grant GCS service account permission to publish to the topic
    gcs_service_account = f"service-{project_id}@gs-project-accounts.iam.gserviceaccount.com"
    print(f"Attempting to grant Pub/Sub Publisher role to: {gcs_service_account}")
    try:
        grant_gcs_pubsub_permission(project_id, topic_name, credentials_path)
    except Exception as e:
        print(f"⚠ Warning: Could not automatically grant IAM permissions")
        print(f"   This is likely because the service account needs 'Pub/Sub Admin' role")
        print(f"\nTrying to create notification anyway (permissions may already be set)...\n")
    
    # Create notification configuration
    # OBJECT_FINALIZE: Sent when a new object is created or an existing object is overwritten
    notification = bucket.notification(
        topic_name=topic_name,
        topic_project=project_id,
        event_types=["OBJECT_FINALIZE"],  # Triggers on file creation
        custom_attributes={
            "created_by": "python-script",
            "purpose": "file-creation-monitoring"
        }
    )
    
    try:
        notification.create()
        print(f"✓ Successfully created notification configuration!")
        print(f"  Bucket: {bucket_name}")
        print(f"  Topic: {topic_path}")
        print(f"  Event Type: OBJECT_FINALIZE (file creation)")
        print(f"  Notification ID: {notification.notification_id}")
    except Exception as e:
        if "already exists" in str(e).lower():
            print(f"Notification already exists for bucket {bucket_name}")
        elif "does not have permission" in str(e) or "403" in str(e):
            print(f"\n❌ Error: GCS service account lacks Pub/Sub Publisher permission")
            print(f"\nThe GCS service account needs permission to publish to the topic.")
            print(f"Ask someone with Pub/Sub Admin role to run this command:\n")
            print(f"gcloud pubsub topics add-iam-policy-binding {topic_name} \\")
            print(f"  --member='serviceAccount:{gcs_service_account}' \\")
            print(f"  --role='roles/pubsub.publisher' \\")
            print(f"  --project=<PROJECT_ID>\n")
            print(f"(Replace <PROJECT_ID> with your actual project ID, not the project number)")
        else:
            raise


def list_bucket_notifications(project_id: str, bucket_name: str, credentials_path: str = None) -> None:
    """
    List all notification configurations for a bucket.
    
    Args:
        project_id: GCP project ID
        bucket_name: Name of the GCS bucket
        credentials_path: Path to service account key file
    """
    # Set credentials if provided
    if credentials_path:
        os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = credentials_path
    
    storage_client = storage.Client(project=project_id)
    bucket = storage_client.bucket(bucket_name)
    
    notifications = bucket.list_notifications()
    
    print(f"\nNotification configurations for bucket '{bucket_name}':")
    for notification in notifications:
        print(f"  - ID: {notification.notification_id}")
        print(f"    Topic: {notification.topic_name}")
        print(f"    Event Types: {notification.event_types}")
        print(f"    Custom Attributes: {notification.custom_attributes}")
        print()


def main():
    parser = argparse.ArgumentParser(
        description="Setup GCS bucket notification to Pub/Sub topic for file creation events"
    )
    parser.add_argument(
        "--project-id",
        default="67453487014",
        help="GCP Project ID (default: 67453487014)"
    )
    parser.add_argument(
        "--bucket",
        default="test-dp-gcs",
        help="GCS bucket name (default: test-dp-gcs)"
    )
    parser.add_argument(
        "--topic",
        default="gcs-notifications-test-dp",
        help="Pub/Sub topic name (default: gcs-notifications-test-dp)"
    )
    parser.add_argument(
        "--service-account-key",
        required=True,
        help="Path to service account JSON key file"
    )
    parser.add_argument(
        "--list",
        action="store_true",
        help="List existing notification configurations"
    )
    
    args = parser.parse_args()
    
    # Validate service account key file exists
    if not os.path.exists(args.service_account_key):
        print(f"Error: Service account key file not found: {args.service_account_key}")
        return
    
    print(f"Setting up GCS Pub/Sub notification...")
    print(f"Project ID: {args.project_id}")
    print(f"Bucket: {args.bucket}")
    print(f"Topic: {args.topic}")
    print(f"Service Account Key: {args.service_account_key}")
    print()
    
    if args.list:
        list_bucket_notifications(args.project_id, args.bucket, args.service_account_key)
    else:
        create_bucket_notification(args.project_id, args.bucket, args.topic, args.service_account_key)
        print()
        list_bucket_notifications(args.project_id, args.bucket, args.service_account_key)


if __name__ == "__main__":
    main()
