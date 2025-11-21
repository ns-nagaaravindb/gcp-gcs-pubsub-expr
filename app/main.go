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
	ProjectID        string
	BucketName       string
	TopicName        string
	SubscriptionName string
}

// App handles GCS and Pub/Sub operations
type App struct {
	storageClient *storage.Client
	pubsubClient  *pubsub.Client
	config        AppConfig
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

	return &App{
		storageClient: storageClient,
		pubsubClient:  pubsubClient,
		config:        config,
	}, nil
}

// Close closes all clients
func (a *App) Close() error {
	if err := a.storageClient.Close(); err != nil {
		return err
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

// DownloadAndPrintFile downloads a file from GCS and prints its contents
func (a *App) DownloadAndPrintFile(ctx context.Context, bucketName, fileName string) error {
	bucket := a.storageClient.Bucket(bucketName)
	obj := bucket.Object(fileName)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Close()

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get object attributes: %w", err)
	}

	fileURL := a.GetFileURL(fileName)
	log.Printf("\n📄 File Details:")
	log.Printf("  URL: %s", fileURL)
	log.Printf("  Size: %d bytes", attrs.Size)
	log.Printf("  Content Type: %s", attrs.ContentType)
	log.Printf("  Created: %s", attrs.Created)
	log.Printf("  Updated: %s", attrs.Updated)
	log.Printf("\n📝 File Content:")
	log.Printf("  %s", strings.Repeat("-", 60))

	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	log.Printf("\n%s\n", string(content))
	log.Printf("  %s\n", strings.Repeat("-", 60))

	return nil
}

// StartListening starts listening for Pub/Sub messages
func (a *App) StartListening(ctx context.Context) error {
	sub := a.pubsubClient.Subscription(a.config.SubscriptionName)

	exists, err := sub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check subscription existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("subscription %s does not exist", a.config.SubscriptionName)
	}

	log.Printf("🎧 Listening for messages on subscription: %s", a.config.SubscriptionName)
	log.Printf("   Press Ctrl+C to stop\n")

	return sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		var notification GCSNotification
		if err := json.Unmarshal(msg.Data, &notification); err != nil {
			log.Printf("❌ Failed to parse notification: %v", err)
			msg.Nack()
			return
		}

		log.Printf("\n🔔 Notification Received:")
		log.Printf("  Event Type: %s", notification.EventType)
		log.Printf("  Bucket: %s", notification.Bucket)
		log.Printf("  File: %s", notification.Name)
		log.Printf("  Size: %s bytes", notification.Size)
		log.Printf("  Content Type: %s", notification.ContentType)

		if notification.EventType == "OBJECT_FINALIZE" {
			if err := a.DownloadAndPrintFile(ctx, notification.Bucket, notification.Name); err != nil {
				log.Printf("❌ Error processing file: %v", err)
				msg.Nack()
				return
			}
		}

		msg.Ack()
	})
}

func main() {
	var (
		projectID        = flag.String("project", "", "GCP project ID")
		bucketName       = flag.String("bucket", "test-dp-gcspubsub-bucket", "GCS bucket name")
		topicName        = flag.String("topic", "test-dp-gcspubsub-bucket", "Pub/Sub topic name")
		subscriptionName = flag.String("subscription", "", "Pub/Sub subscription name (defaults to topic-subscription)")
		mode             = flag.String("mode", "listen", "Mode: 'create', 'listen', or 'both'")
		fileName         = flag.String("file", "", "File name to create (required for create mode)")
		fileContent      = flag.String("content", "Hello, World!", "Content for the file")
	)
	flag.Parse()

	if *projectID == "" {
		log.Fatal("❌ -project flag is required")
	}

	if *subscriptionName == "" {
		*subscriptionName = fmt.Sprintf("%s-subscription", *topicName)
	}

	config := AppConfig{
		ProjectID:        *projectID,
		BucketName:       *bucketName,
		TopicName:        *topicName,
		SubscriptionName: *subscriptionName,
	}

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
