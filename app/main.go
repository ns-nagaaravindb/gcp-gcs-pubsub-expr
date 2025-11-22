package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
)

// getKeys returns a slice of keys from a map
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

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

// AppConfig holds application configuration
type AppConfig struct {
	ProjectID         string
	ConsumerProjectID string // Project ID for the consumer (can be different from publisher)
	BucketName        string
	TopicName         string
	SubscriptionName  string
}

// App handles GCS and Pub/Sub operations
type App struct {
	storageClient        *storage.Client
	pubsubClient         *pubsub.Client
	consumerPubsubClient *pubsub.Client // Separate client for consumer project
	config               AppConfig
}

// NewApp creates a new application instance
func NewApp(ctx context.Context, config AppConfig) (*App, error) {
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	pubsubClient, err := pubsub.NewClient(ctx, config.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	// Create consumer pubsub client if consumer project is different
	var consumerPubsubClient *pubsub.Client
	if config.ConsumerProjectID != "" && config.ConsumerProjectID != config.ProjectID {
		consumerPubsubClient, err = pubsub.NewClient(ctx, config.ConsumerProjectID)
		if err != nil {
			return nil, fmt.Errorf("failed to create consumer pubsub client: %w", err)
		}
		log.Printf("✓ Created separate Pub/Sub client for consumer project: %s", config.ConsumerProjectID)
	} else {
		// Use the same client if consumer project is not specified or is the same
		consumerPubsubClient = pubsubClient
	}

	return &App{
		storageClient:        storageClient,
		pubsubClient:         pubsubClient,
		consumerPubsubClient: consumerPubsubClient,
		config:               config,
	}, nil
}

// Close closes all clients
func (a *App) Close() error {
	if err := a.storageClient.Close(); err != nil {
		return err
	}
	if a.consumerPubsubClient != a.pubsubClient {
		if err := a.consumerPubsubClient.Close(); err != nil {
			return err
		}
	}
	return a.pubsubClient.Close()
}

// CreateFile creates a file in the GCS bucket
func (a *App) CreateFile(ctx context.Context, fileName string, content []byte, contentType string) error {
	bucket := a.storageClient.Bucket(a.config.BucketName)
	obj := bucket.Object(fileName)
	writer := obj.NewWriter(ctx)
	writer.ContentType = contentType

	if _, err := writer.Write(content); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	fileURL := fmt.Sprintf("gs://%s/%s", a.config.BucketName, fileName)
	log.Printf("✓ Created file: %s", fileURL)
	return nil
}

// GetFileURL generates the GCS file URL
func (a *App) GetFileURL(fileName string) string {
	return fmt.Sprintf("gs://%s/%s", a.config.BucketName, fileName)
}

// ConstructGCSURL constructs a GCS URL from bucket and file name
func ConstructGCSURL(bucketName, fileName string) string {
	return fmt.Sprintf("gs://%s/%s", bucketName, fileName)
}

// DownloadAndPrintFile downloads a file from GCS and prints its contents
func (a *App) DownloadAndPrintFile(ctx context.Context, bucketName, fileName string) error {
	// Construct GCS URL from bucket and file name
	fileURL := ConstructGCSURL(bucketName, fileName)
	log.Printf("\n📄 Constructed GCS URL: %s", fileURL)
	log.Printf("  Reading file from GCS...\n")

	// Validate inputs
	if bucketName == "" {
		return fmt.Errorf("bucket name is empty")
	}
	if fileName == "" {
		return fmt.Errorf("file name is empty")
	}

	bucket := a.storageClient.Bucket(bucketName)
	obj := bucket.Object(fileName)

	// Check if object exists and get attributes first
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get object attributes for %s: %w", fileURL, err)
	}

	log.Printf("📄 File Details:")
	log.Printf("  URL: %s", fileURL)
	log.Printf("  Size: %d bytes", attrs.Size)
	log.Printf("  Content Type: %s", attrs.ContentType)
	log.Printf("  Created: %s", attrs.Created)
	log.Printf("  Updated: %s", attrs.Updated)
	log.Printf("\n📝 File Content:")
	log.Printf("  %s", strings.Repeat("-", 60))

	// Create reader and read content
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to create reader for %s: %w", fileURL, err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file content from %s: %w", fileURL, err)
	}

	log.Printf("\n%s\n", string(content))
	log.Printf("  %s\n", strings.Repeat("-", 60))

	return nil
}

// StartListening starts listening for Pub/Sub messages
func (a *App) StartListening(ctx context.Context) error {
	// Use consumer client for subscription
	client := a.consumerPubsubClient
	projectID := a.config.ConsumerProjectID
	if projectID == "" {
		projectID = a.config.ProjectID
	}

	sub := client.Subscription(a.config.SubscriptionName)

	exists, err := sub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check subscription existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("subscription %s does not exist in project %s", a.config.SubscriptionName, projectID)
	}

	log.Printf("🎧 Listening for messages on subscription: %s (Project: %s)", a.config.SubscriptionName, projectID)
	log.Printf("   Press Ctrl+C to stop\n")

	return sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		// Log raw message for debugging
		log.Printf("\n📨 Raw message received (ID: %s)", msg.ID)

		// First, parse into a map to check available fields
		var rawData map[string]interface{}
		if err := json.Unmarshal(msg.Data, &rawData); err != nil {
			log.Printf("❌ Failed to parse notification as JSON: %v", err)
			log.Printf("   Raw message data: %s", string(msg.Data))
			msg.Nack()
			return
		}

		// Log available fields for debugging
		log.Printf("   Available JSON fields: %v", getKeys(rawData))

		// Now parse into struct
		var notification GCSNotification
		if err := json.Unmarshal(msg.Data, &notification); err != nil {
			log.Printf("❌ Failed to parse notification into struct: %v", err)
			log.Printf("   Raw message data: %s", string(msg.Data))
			msg.Nack()
			return
		}

		// Try to extract eventType from raw data if struct parsing failed
		if notification.EventType == "" {
			if eventType, ok := rawData["eventType"].(string); ok && eventType != "" {
				notification.EventType = eventType
			} else if eventType, ok := rawData["event_type"].(string); ok && eventType != "" {
				notification.EventType = eventType
			} else if eventType, ok := rawData["EventType"].(string); ok && eventType != "" {
				notification.EventType = eventType
			}
		}

		// Print notification details
		log.Printf("\n%s", strings.Repeat("=", 70))
		log.Printf("🔔 NOTIFICATION DETAILS")
		log.Printf("%s", strings.Repeat("=", 70))
		log.Printf("  Event Type:        %s", notification.EventType)
		log.Printf("  Bucket:            %s", notification.Bucket)
		log.Printf("  File Name:         %s", notification.Name)
		log.Printf("  Size:              %s bytes", notification.Size)
		log.Printf("  Content Type:      %s", notification.ContentType)
		log.Printf("  Generation:        %s", notification.Generation)
		log.Printf("  Meta Generation:   %s", notification.MetaGeneration)
		log.Printf("  Storage Class:     %s", notification.StorageClass)
		log.Printf("  MD5 Hash:          %s", notification.MD5Hash)
		log.Printf("  CRC32C:            %s", notification.CRC32C)

		// Format time fields safely
		if !notification.TimeCreated.IsZero() {
			log.Printf("  Time Created:      %s", notification.TimeCreated.Format(time.RFC3339))
		} else {
			log.Printf("  Time Created:      (not provided)")
		}
		if !notification.Updated.IsZero() {
			log.Printf("  Updated:           %s", notification.Updated.Format(time.RFC3339))
		} else {
			log.Printf("  Updated:           (not provided)")
		}
		if !notification.EventTime.IsZero() {
			log.Printf("  Event Time:        %s", notification.EventTime.Format(time.RFC3339))
		} else {
			log.Printf("  Event Time:        (not provided)")
		}

		log.Printf("  Notification Config: %s", notification.NotificationConfig)

		// Construct GCS URL from notification
		gcsURL := ConstructGCSURL(notification.Bucket, notification.Name)
		log.Printf("  Constructed GCS URL: %s", gcsURL)
		log.Printf("%s\n", strings.Repeat("=", 70))

		// Validate required fields
		if notification.Bucket == "" {
			log.Printf("❌ Error: Bucket name is empty in notification")
			msg.Nack()
			return
		}
		if notification.Name == "" {
			log.Printf("❌ Error: File name is empty in notification")
			msg.Nack()
			return
		}

		// Process file for OBJECT_FINALIZE events (or if EventType is empty but we have valid bucket/file)
		shouldProcess := notification.EventType == "OBJECT_FINALIZE" ||
			(notification.EventType == "" && notification.Bucket != "" && notification.Name != "")

		if shouldProcess {
			log.Printf("\n📥 Processing file from GCS...")
			log.Printf("   Attempting to download: %s", gcsURL)
			if err := a.DownloadAndPrintFile(ctx, notification.Bucket, notification.Name); err != nil {
				log.Printf("❌ Error processing file: %v", err)
				log.Printf("   Bucket: %s, File: %s", notification.Bucket, notification.Name)
				msg.Nack()
				return
			}
			log.Printf("\n✅ File processed successfully\n")
		} else {
			log.Printf("ℹ️  Skipping file download (Event Type: '%s')\n", notification.EventType)
		}

		msg.Ack()
	})
}

func main() {
	var (
		projectID         = flag.String("project", "", "GCP project ID (publisher project)")
		consumerProjectID = flag.String("consumer-project", "", "GCP consumer project ID (if different from publisher)")
		bucketName        = flag.String("bucket", "test-dp-gcspubsub-bucket", "GCS bucket name")
		topicName         = flag.String("topic", "test-dp-gcspubsub-bucket", "Pub/Sub topic name")
		subscriptionName  = flag.String("subscription", "", "Pub/Sub subscription name (defaults to topic-subscription)")
		mode              = flag.String("mode", "listen", "Mode: 'create', 'listen', or 'both'")
		fileName          = flag.String("file", "", "File name to create (required for create mode)")
		fileContent       = flag.String("content", "Hello, World!", "Content for the file")
	)
	flag.Parse()

	// Check for environment variables as fallback
	if *projectID == "" {
		*projectID = os.Getenv("GCP_PROJECT_ID")
	}
	if *consumerProjectID == "" {
		*consumerProjectID = os.Getenv("GCP_CONSUMER_PROJECT_ID")
	}

	if *projectID == "" {
		log.Fatal("❌ -project flag or GCP_PROJECT_ID environment variable is required")
	}

	// If consumer project is not set, use the same as publisher project
	if *consumerProjectID == "" {
		*consumerProjectID = *projectID
	}

	if *subscriptionName == "" {
		// Use different subscription name if consumer project is different
		if *consumerProjectID != *projectID {
			*subscriptionName = fmt.Sprintf("%s-consumer-subscription", *topicName)
		} else {
			*subscriptionName = fmt.Sprintf("%s-subscription", *topicName)
		}
	}

	config := AppConfig{
		ProjectID:         *projectID,
		ConsumerProjectID: *consumerProjectID,
		BucketName:        *bucketName,
		TopicName:         *topicName,
		SubscriptionName:  *subscriptionName,
	}

	log.Printf("📋 Configuration:")
	log.Printf("   Publisher Project: %s", config.ProjectID)
	log.Printf("   Consumer Project:  %s", config.ConsumerProjectID)
	log.Printf("   Bucket:            %s", config.BucketName)
	log.Printf("   Topic:             %s", config.TopicName)
	log.Printf("   Subscription:      %s\n", config.SubscriptionName)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app, err := NewApp(ctx, config)
	if err != nil {
		log.Fatalf("❌ Failed to create app: %v", err)
	}
	defer app.Close()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	switch *mode {
	case "create":
		if *fileName == "" {
			log.Fatal("❌ -file flag is required for create mode")
		}
		if err := app.CreateFile(ctx, *fileName, []byte(*fileContent), "text/plain"); err != nil {
			log.Fatalf("❌ Failed to create file: %v", err)
		}
		log.Printf("✅ File created successfully")

	case "listen":
		go func() {
			<-sigChan
			log.Printf("\n🛑 Shutting down...")
			cancel()
		}()
		if err := app.StartListening(ctx); err != nil && err != context.Canceled {
			log.Fatalf("❌ Listening error: %v", err)
		}

	case "both":
		// Create file first
		if *fileName == "" {
			*fileName = fmt.Sprintf("test-%d.txt", time.Now().Unix())
		}
		if err := app.CreateFile(ctx, *fileName, []byte(*fileContent), "text/plain"); err != nil {
			log.Fatalf("❌ Failed to create file: %v", err)
		}

		// Wait a moment for notification to propagate
		time.Sleep(2 * time.Second)

		// Start listening
		go func() {
			<-sigChan
			log.Printf("\n🛑 Shutting down...")
			cancel()
		}()
		if err := app.StartListening(ctx); err != nil && err != context.Canceled {
			log.Fatalf("❌ Listening error: %v", err)
		}

	default:
		log.Fatalf("❌ Invalid mode: %s. Use 'create', 'listen', or 'both'", *mode)
	}
}
