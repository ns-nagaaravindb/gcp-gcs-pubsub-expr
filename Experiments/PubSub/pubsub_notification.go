package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
)

// GCSNotification represents the structure of a GCS notification message
type GCSNotification struct {
	Name               string    `json:"name"`
	Bucket             string    `json:"bucket"`
	Generation         string    `json:"generation"`
	MetaGeneration     string    `json:"metageneration"`
	ContentType        string    `json:"contentType"`
	TimeCreated        time.Time `json:"timeCreated"`
	Updated            time.Time `json:"updated"`
	StorageClass       string    `json:"storageClass"`
	Size               string    `json:"size"`
	MD5Hash            string    `json:"md5Hash"`
	CRC32C             string    `json:"crc32c"`
	EventType          string    `json:"eventType"`
	EventTime          time.Time `json:"eventTime"`
	NotificationConfig string    `json:"notificationConfig"`
}

// NotificationProcessor handles GCS notifications
type NotificationProcessor struct {
	storageClient       *storage.Client
	pubsubClient        *pubsub.Client
	projectID           string
	topicName           string
	subscriptionName    string
	dlqTopicName        string
	dlqSubscriptionName string
	maxRetries          int
}

// NewNotificationProcessor creates a new notification processor
func NewNotificationProcessor(ctx context.Context, projectID, topicName, subscriptionName, dlqTopicName, dlqSubscriptionName string) (*NotificationProcessor, error) {
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	return &NotificationProcessor{
		storageClient:       storageClient,
		pubsubClient:        pubsubClient,
		projectID:           projectID,
		topicName:           topicName,
		subscriptionName:    subscriptionName,
		dlqTopicName:        dlqTopicName,
		dlqSubscriptionName: dlqSubscriptionName,
		maxRetries:          5,
	}, nil
}

// Close closes the clients
func (np *NotificationProcessor) Close() error {
	if err := np.storageClient.Close(); err != nil {
		return err
	}
	return np.pubsubClient.Close()
}

// SetupPubSubResources creates the necessary Pub/Sub topics and subscriptions
func (np *NotificationProcessor) SetupPubSubResources(ctx context.Context) error {
	// Create main topic
	mainTopic := np.pubsubClient.Topic(np.topicName)
	exists, err := mainTopic.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if topic exists: %w", err)
	}
	if !exists {
		mainTopic, err = np.pubsubClient.CreateTopic(ctx, np.topicName)
		if err != nil {
			return fmt.Errorf("failed to create topic: %w", err)
		}
		fmt.Printf("Created topic: %s\n", np.topicName)
	}

	// Create DLQ topic
	dlqTopic := np.pubsubClient.Topic(np.dlqTopicName)
	exists, err = dlqTopic.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if DLQ topic exists: %w", err)
	}
	if !exists {
		dlqTopic, err = np.pubsubClient.CreateTopic(ctx, np.dlqTopicName)
		if err != nil {
			return fmt.Errorf("failed to create DLQ topic: %w", err)
		}
		fmt.Printf("Created DLQ topic: %s\n", np.dlqTopicName)
	}

	// Create main subscription with DLQ
	mainSub := np.pubsubClient.Subscription(np.subscriptionName)
	exists, err = mainSub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if subscription exists: %w", err)
	}
	if !exists {
		dlqTopicPath := fmt.Sprintf("projects/%s/topics/%s", np.projectID, np.dlqTopicName)
		subConfig := pubsub.SubscriptionConfig{
			Topic: mainTopic,
			DeadLetterPolicy: &pubsub.DeadLetterPolicy{
				DeadLetterTopic:     dlqTopicPath,
				MaxDeliveryAttempts: np.maxRetries,
			},
		}
		mainSub, err = np.pubsubClient.CreateSubscription(ctx, np.subscriptionName, subConfig)
		if err != nil {
			return fmt.Errorf("failed to create subscription: %w", err)
		}
		fmt.Printf("Created subscription: %s with DLQ: %s\n", np.subscriptionName, np.dlqTopicName)
	}

	// Create DLQ subscription
	dlqSub := np.pubsubClient.Subscription(np.dlqSubscriptionName)
	exists, err = dlqSub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if DLQ subscription exists: %w", err)
	}
	if !exists {
		dlqSubConfig := pubsub.SubscriptionConfig{
			Topic: dlqTopic,
		}
		dlqSub, err = np.pubsubClient.CreateSubscription(ctx, np.dlqSubscriptionName, dlqSubConfig)
		if err != nil {
			return fmt.Errorf("failed to create DLQ subscription: %w", err)
		}
		fmt.Printf("Created DLQ subscription: %s\n", np.dlqSubscriptionName)
	}

	// Grant the GCS service account permission to publish to the topic (informational only)
	fmt.Printf("\n📋 IAM Permission Information:\n")
	fmt.Printf("For GCS notifications to work, the service account below needs Pub/Sub Publisher role:\n")
	fmt.Printf("Service Account: service-%s@gs-project-accounts.iam.gserviceaccount.com\n", np.projectID)
	fmt.Printf("\nIf you encounter permission errors, run:\n")
	fmt.Printf("  ./setup_iam_permissions.sh %s\n", np.projectID)
	fmt.Printf("or manually grant the role in Google Cloud Console.\n\n")

	return nil
}

// SetupBucketNotification configures bucket notifications
func (np *NotificationProcessor) SetupBucketNotification(ctx context.Context, bucketName string) error {
	bucket := np.storageClient.Bucket(bucketName)

	topicName := fmt.Sprintf("projects/%s/topics/%s", np.projectID, np.topicName)

	notification := &storage.Notification{
		TopicProjectID: np.projectID,
		TopicID:        np.topicName,
		PayloadFormat:  storage.JSONPayload,
		EventTypes: []string{
			"OBJECT_FINALIZE", // Object created
			"OBJECT_DELETE",   // Object deleted
		},
	}

	createdNotification, err := bucket.AddNotification(ctx, notification)
	if err != nil {
		return fmt.Errorf("failed to add notification: %w", err)
	}

	fmt.Printf("Created bucket notification: %s for topic: %s\n", createdNotification.ID, topicName)
	return nil
}

// ProcessFile reads and processes a file from GCS
func (np *NotificationProcessor) ProcessFile(ctx context.Context, bucketName, fileName string) error {
	fmt.Printf("Processing file: gs://%s/%s\n", bucketName, fileName)

	// Simulate potential processing errors for demonstration
	if strings.Contains(fileName, "error") {
		return fmt.Errorf("simulated processing error for file: %s", fileName)
	}

	bucket := np.storageClient.Bucket(bucketName)
	obj := bucket.Object(fileName)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Close()

	// Read file content
	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err != nil && err.Error() != "EOF" {
		return fmt.Errorf("failed to read file: %w", err)
	}

	content := string(buf[:n])
	fmt.Printf("File content preview (first 1024 bytes):\n%s\n", content)
	fmt.Printf("File size: %d bytes\n", reader.Attrs.Size)
	fmt.Printf("Content type: %s\n", reader.Attrs.ContentType)
	fmt.Printf("Last modified: %s\n", reader.Attrs.LastModified)

	return nil
}

// StartListening starts listening for notifications
func (np *NotificationProcessor) StartListening(ctx context.Context) error {
	sub := np.pubsubClient.Subscription(np.subscriptionName)

	// Configure subscription settings
	sub.ReceiveSettings.Synchronous = false
	sub.ReceiveSettings.NumGoroutines = 10
	sub.ReceiveSettings.MaxOutstandingMessages = 100

	fmt.Printf("Starting to listen for messages on subscription: %s\n", np.subscriptionName)

	err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		fmt.Printf("\n--- New notification received ---\n")
		fmt.Printf("Message ID: %s\n", msg.ID)
		fmt.Printf("Publish time: %s\n", msg.PublishTime)

		// Parse the GCS notification
		var notification GCSNotification
		if err := json.Unmarshal(msg.Data, &notification); err != nil {
			fmt.Printf("Failed to parse notification: %v\n", err)
			msg.Nack()
			return
		}

		fmt.Printf("Event type: %s\n", notification.EventType)
		fmt.Printf("Bucket: %s\n", notification.Bucket)
		fmt.Printf("File: %s\n", notification.Name)
		fmt.Printf("Size: %s bytes\n", notification.Size)
		fmt.Printf("Content type: %s\n", notification.ContentType)

		// Process the file
		if notification.EventType == "OBJECT_FINALIZE" {
			if err := np.ProcessFile(ctx, notification.Bucket, notification.Name); err != nil {
				fmt.Printf("Error processing file: %v\n", err)
				msg.Nack() // This will retry and eventually go to DLQ
				return
			}
		}

		fmt.Printf("Successfully processed notification\n")
		msg.Ack()
	})

	return err
}

// StartDLQListener starts listening for messages in the Dead Letter Queue
func (np *NotificationProcessor) StartDLQListener(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	dlqSub := np.pubsubClient.Subscription(np.dlqSubscriptionName)
	fmt.Printf("Starting DLQ listener on subscription: %s\n", np.dlqSubscriptionName)

	err := dlqSub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		fmt.Printf("\n--- DLQ message received ---\n")
		fmt.Printf("Message ID: %s\n", msg.ID)
		fmt.Printf("Publish time: %s\n", msg.PublishTime)

		var notification GCSNotification
		if err := json.Unmarshal(msg.Data, &notification); err != nil {
			fmt.Printf("Failed to parse DLQ notification: %v\n", err)
			msg.Ack() // Acknowledge to prevent infinite loop
			return
		}

		fmt.Printf("DLQ - Failed processing file: gs://%s/%s\n", notification.Bucket, notification.Name)
		fmt.Printf("DLQ - Event type: %s\n", notification.EventType)

		// Here you could implement additional error handling:
		// - Log to monitoring system
		// - Send alert
		// - Store in database for manual processing
		// - Implement exponential backoff retry

		msg.Ack()
	})

	if err != nil {
		fmt.Printf("DLQ listener error: %v\n", err)
	}
}

// CreateTestFiles creates some test files in the bucket
func (np *NotificationProcessor) CreateTestFiles(ctx context.Context, bucketName string) error {
	bucket := np.storageClient.Bucket(bucketName)

	testFiles := map[string]string{
		"test-success.txt": "This is a successful test file.\nContent will be processed successfully.",
		"test-error.txt":   "This file contains 'error' in name and will trigger processing error.",
		"data/sample.json": `{"message": "Sample JSON data", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`,
		"logs/app.log":     "2023-09-05 10:30:00 INFO Application started\n2023-09-05 10:30:01 INFO Processing request",
	}

	for fileName, content := range testFiles {
		obj := bucket.Object(fileName)
		writer := obj.NewWriter(ctx)
		writer.ContentType = "text/plain"

		if strings.HasSuffix(fileName, ".json") {
			writer.ContentType = "application/json"
		}

		if _, err := writer.Write([]byte(content)); err != nil {
			writer.Close()
			return fmt.Errorf("failed to write file %s: %w", fileName, err)
		}

		if err := writer.Close(); err != nil {
			return fmt.Errorf("failed to close writer for file %s: %w", fileName, err)
		}

		fmt.Printf("Created test file: gs://%s/%s\n", bucketName, fileName)
		time.Sleep(1 * time.Second) // Small delay to avoid rate limits
	}

	return nil
}

func runPubSubNotificationDemo() {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/Users/nagaaravindb/Downloads/cred/data-qe-da7e1252-3baad2049955.json")
	projectID := "67453487014"
	bucketName := fmt.Sprintf("test-dp-gcs")
	// region := "us-central1"

	topicName := "gcs-notifications-test-dp"
	subscriptionName := "gcs-notifications-sub-test-dp"
	dlqTopicName := "gcs-notifications-dlq-test-dp"
	dlqSubscriptionName := "gcs-notifications-dlq-sub-test-dp"

	ctx := context.Background()

	fmt.Println("=" + strings.Repeat("=", 70) + "=")
	fmt.Println("Starting GCS Pub/Sub Notification Demo")
	fmt.Println("=" + strings.Repeat("=", 70) + "=")
	fmt.Printf("Project ID: %s\n", projectID)
	fmt.Printf("Bucket: %s\n", bucketName)
	fmt.Printf("Topic: %s\n", topicName)
	fmt.Printf("Subscription: %s\n", subscriptionName)
	fmt.Printf("DLQ Topic: %s\n", dlqTopicName)
	fmt.Printf("DLQ Subscription: %s\n", dlqSubscriptionName)
	fmt.Println()

	// Create the notification processor
	processor, err := NewNotificationProcessor(ctx, projectID, topicName, subscriptionName, dlqTopicName, dlqSubscriptionName)
	if err != nil {
		log.Fatalf("Failed to create notification processor: %v", err)
	}
	defer processor.Close()

	// Step 1: Setup Pub/Sub resources
	fmt.Println("Step 1: Setting up Pub/Sub resources...")
	if err := processor.SetupPubSubResources(ctx); err != nil {
		log.Fatalf("Failed to setup Pub/Sub resources: %v", err)
	}
	fmt.Println()

	// Step 3: Setup bucket notification
	fmt.Println("Step 3: Setting up bucket notifications...")
	if err := processor.SetupBucketNotification(ctx, bucketName); err != nil {
		if strings.Contains(err.Error(), "does not have permission to publish messages") {
			fmt.Printf("\n IAM Permission Required! \n")
			fmt.Printf("The Google Cloud Storage service account needs permission to publish to Pub/Sub.\n\n")
			fmt.Printf("🔧 Quick Fix - Use Google Cloud Console:\n")
			fmt.Printf("1. Go to: https://console.cloud.google.com/iam-admin/iam?project=%s\n", projectID)
			fmt.Printf("2. Click 'Grant Access'\n")
			fmt.Printf("3. Add principal: service-%s@gs-project-accounts.iam.gserviceaccount.com\n", projectID)
			fmt.Printf("4. Select role: 'Pub/Sub Publisher'\n")
			fmt.Printf("5. Click 'Save'\n\n")
			fmt.Printf("📖 For detailed instructions, see: MANUAL_SETUP.md\n")
			fmt.Printf("⚡ Alternative: Ask an admin to run: ./setup_iam_permissions.sh %s\n\n", projectID)
			log.Fatalf("Please set up the IAM permission and run the demo again.")
		}
		log.Fatalf("Failed to setup bucket notification: %v", err)
	}
	fmt.Println()

	// Step 4: Start listening for notifications and DLQ messages
	fmt.Println("Step 4: Starting message listeners...")

	// Create a context that can be canceled
	listenCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start DLQ listener in a separate goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go processor.StartDLQListener(listenCtx, &wg)

	// Start main listener in a separate goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := processor.StartListening(listenCtx); err != nil {
			fmt.Printf("Listener error: %v\n", err)
		}
	}()

	// Wait a moment for listeners to start
	time.Sleep(3 * time.Second)

	// Step 5: Create test files to trigger notifications
	fmt.Println("Step 5: Creating test files to trigger notifications...")
	if err := processor.CreateTestFiles(ctx, bucketName); err != nil {
		log.Printf("Failed to create test files: %v", err)
	}
	fmt.Println()

	// Step 6: Wait for messages to be processed
	fmt.Println("Step 6: Waiting for notifications to be processed...")
	fmt.Println("Press Ctrl+C to stop listening and cleanup resources")

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either signal or timeout
	select {
	case <-sigChan:
		fmt.Println("\nReceived interrupt signal, cleaning up...")
	case <-time.After(60 * time.Second):
		fmt.Println("\nTimeout reached, cleaning up...")
	}

	// Cancel the context to stop listeners
	cancel()

	// Wait for all goroutines to finish
	wg.Wait()

	// Step 7: Cleanup resources
	fmt.Println("Step 7: Cleaning up resources...")

	// Delete test files
	bucket := processor.storageClient.Bucket(bucketName)
	it := bucket.Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if err != nil {
			break
		}
		if err := bucket.Object(attrs.Name).Delete(ctx); err != nil {
			fmt.Printf("Failed to delete file %s: %v\n", attrs.Name, err)
		} else {
			fmt.Printf("Deleted file: %s\n", attrs.Name)
		}
	}
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 70) + "=")
	fmt.Println("GCS Pub/Sub Notification Demo Completed!")
	fmt.Println("=" + strings.Repeat("=", 70) + "=")
}

func main() {
	runPubSubNotificationDemo()
}
