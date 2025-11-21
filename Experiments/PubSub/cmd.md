# Run with service account key file
python Experiments/PubSub/pubsub-notification.py --service-account-key /Users/nagaaravindb/Downloads/cred/data-qe-da7e1252-3baad2049955.json

# Or with custom values
python3 Experiments/PubSub/pubsub-notification.py \
  --service-account-key /Users/nagaaravindb/Downloads/cred/data-qe-da7e1252-3baad2049955.json \
  --project-id 67453487014 \
  --bucket test-dp-gcs \
  --topic gcs-notifications-test-dp

# List existing notifications
python Experiments/PubSub/pubsub-notification.py \
  --service-account-key /Users/nagaaravindb/Downloads/cred/data-qe-da7e1252-3baad2049955.json \
  --list