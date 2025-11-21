package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
)

const (
	projectID         = "data-qe-da7e1252"
	topicID           = "demo-topic-dp"
	subscriptionID    = "demo-subscription-dp"
	dlqTopicID        = "demo-dlq-topic-dp"
	dlqSubscriptionID = "demo-dlq-subscription-dp"
	maxRetries        = 5
	ackDeadline       = 10 * time.Second
)

// PubSubManager manages Pub/Sub topics, subscriptions, and message processing
type PubSubManager struct {
	client       *pubsub.Client
	topic        *pubsub.Topic
	subscription *pubsub.Subscription
	dlqTopic     *pubsub.Topic
	dlqSub       *pubsub.Subscription
}

// NewPubSubManager creates a new PubSubManager
func NewPubSubManager(ctx context.Context) (*PubSubManager, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Pub/Sub client: %w", err)
	}

	return &PubSubManager{
		client: client,
	}, nil
}

// SetupTopicsAndSubscriptions creates all necessary topics and subscriptions
func (p *PubSubManager) SetupTopicsAndSubscriptions(ctx context.Context) error {
	// Create main topic
	mainTopic := p.client.Topic(topicID)
	exists, err := mainTopic.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if main topic exists: %w", err)
	}
	if !exists {
		mainTopic, err = p.client.CreateTopic(ctx, topicID)
		if err != nil {
			return fmt.Errorf("failed to create main topic: %w", err)
		}
		log.Printf("Created main topic: %s", topicID)
	} else {
		log.Printf("Using existing main topic: %s", topicID)
	}
	p.topic = mainTopic

	// Create DLQ topic
	dlqTopic := p.client.Topic(dlqTopicID)
	exists, err = dlqTopic.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if DLQ topic exists: %w", err)
	}
	if !exists {
		dlqTopic, err = p.client.CreateTopic(ctx, dlqTopicID)
		if err != nil {
			return fmt.Errorf("failed to create DLQ topic: %w", err)
		}
		log.Printf("Created DLQ topic: %s", dlqTopicID)
	} else {
		log.Printf("Using existing DLQ topic: %s", dlqTopicID)
	}
	p.dlqTopic = dlqTopic

	// Delete existing main subscription if it exists (to recreate with DLQ policy)
	mainSub := p.client.Subscription(subscriptionID)
	exists, err = mainSub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if main subscription exists: %w", err)
	}
	if exists {
		log.Printf("Deleting existing subscription: %s to recreate with DLQ policy", subscriptionID)
		if err := mainSub.Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete existing subscription: %w", err)
		}
		time.Sleep(2 * time.Second) // Wait for deletion to propagate
	}

	// Create main subscription with DLQ policy
	config := pubsub.SubscriptionConfig{
		Topic:             p.topic,
		AckDeadline:       ackDeadline,
		RetentionDuration: 24 * time.Hour,
		DeadLetterPolicy: &pubsub.DeadLetterPolicy{
			DeadLetterTopic:     p.dlqTopic.String(),
			MaxDeliveryAttempts: maxRetries,
		},
	}
	mainSub, err = p.client.CreateSubscription(ctx, subscriptionID, config)
	if err != nil {
		return fmt.Errorf("failed to create main subscription: %w", err)
	}
	log.Printf("Created main subscription: %s with DLQ policy (max %d retries)", subscriptionID, maxRetries)
	p.subscription = mainSub

	// Delete and recreate DLQ subscription
	dlqSub := p.client.Subscription(dlqSubscriptionID)
	exists, err = dlqSub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if DLQ subscription exists: %w", err)
	}
	if exists {
		log.Printf("Deleting existing DLQ subscription: %s", dlqSubscriptionID)
		if err := dlqSub.Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete existing DLQ subscription: %w", err)
		}
		time.Sleep(2 * time.Second) // Wait for deletion to propagate
	}

	// Create DLQ subscription
	config = pubsub.SubscriptionConfig{
		Topic:             p.dlqTopic,
		AckDeadline:       ackDeadline,
		RetentionDuration: 7 * 24 * time.Hour, // Keep DLQ messages longer
	}
	dlqSub, err = p.client.CreateSubscription(ctx, dlqSubscriptionID, config)
	if err != nil {
		return fmt.Errorf("failed to create DLQ subscription: %w", err)
	}
	log.Printf("Created DLQ subscription: %s", dlqSubscriptionID)
	p.dlqSub = dlqSub

	return nil
}

// PublishMessage publishes a message to the main topic
func (p *PubSubManager) PublishMessage(ctx context.Context, data string, attributes map[string]string) error {
	if attributes == nil {
		attributes = make(map[string]string)
	}
	attributes["timestamp"] = time.Now().Format(time.RFC3339)

	result := p.topic.Publish(ctx, &pubsub.Message{
		Data:       []byte(data),
		Attributes: attributes,
	})

	// Wait for the message to be published
	msgID, err := result.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published message with ID: %s, Data: %s", msgID, data)
	return nil
}

// ProcessMessages processes messages from the main subscription
func (p *PubSubManager) ProcessMessages(ctx context.Context, processFunc func(msg *pubsub.Message) error) error {
	log.Printf("Starting to process messages from subscription: %s", subscriptionID)

	return p.subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		log.Printf("Received message ID: %s, Data: %s", msg.ID, string(msg.Data))

		// Add processing logic
		if err := processFunc(msg); err != nil {
			log.Printf("Error processing message %s: %v", msg.ID, err)
			msg.Nack()
			return
		}

		log.Printf("Successfully processed message: %s", msg.ID)
		msg.Ack()
	})
}

// ProcessDLQMessages processes messages from the Dead Letter Queue
func (p *PubSubManager) ProcessDLQMessages(ctx context.Context) error {
	log.Printf("Starting to process DLQ messages from subscription: %s", dlqSubscriptionID)

	messageCount := 0
	return p.dlqSub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		messageCount++
		log.Printf("*** DLQ Message %d ***", messageCount)
		log.Printf("  ID: %s", msg.ID)
		log.Printf("  Data: %s", string(msg.Data))
		log.Printf("  Attributes: %v", msg.Attributes)
		log.Printf("  Publish Time: %v", msg.PublishTime)
		log.Printf("  Delivery Attempt: %d", msg.DeliveryAttempt)

		// Acknowledge the DLQ message
		msg.Ack()
		log.Printf("Acknowledged DLQ message: %s", msg.ID)
	})
}

// Close closes the Pub/Sub client and all associated resources
func (p *PubSubManager) Close() error {
	if p.topic != nil {
		p.topic.Stop()
	}
	if p.dlqTopic != nil {
		p.dlqTopic.Stop()
	}
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

// MessageProcessor defines how to process messages
func MessageProcessor(msg *pubsub.Message) error {
	data := string(msg.Data)

	// Simulate processing logic
	if data == "error_message" {
		return fmt.Errorf("intentional processing error for message: %s", msg.ID)
	}

	if data == "slow_message" {
		// Simulate slow processing
		time.Sleep(2 * time.Second)
	}

	// Simulate successful processing
	log.Printf("Processing completed for message: %s with data: %s", msg.ID, data)
	return nil
}

func main() {
	// Set up Google Cloud credentials
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/Users/nagaaravindb/Downloads/cred/data-qe-da7e1252-3baad2049955.json")

	ctx := context.Background()

	// Create PubSub manager
	manager, err := NewPubSubManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create PubSub manager: %v", err)
	}
	defer manager.Close()

	// Setup topics and subscriptions
	if err := manager.SetupTopicsAndSubscriptions(ctx); err != nil {
		log.Fatalf("Failed to setup topics and subscriptions: %v", err)
	}

	// Publish some test messages
	testMessages := []string{
		"Hello, World! Message 1",
		"error_message", // This will cause processing to fail and go to DLQ
		"Hello, World! Message 2",
	}

	log.Println("\n=== Publishing Messages ===")
	for i, msg := range testMessages {
		attributes := map[string]string{
			"message_index": fmt.Sprintf("%d", i),
		}
		if err := manager.PublishMessage(ctx, msg, attributes); err != nil {
			log.Printf("Error publishing message %d: %v", i, err)
		}
	}

	// Wait a bit for messages to be published
	time.Sleep(3 * time.Second)

	// Start processing messages in separate goroutines
	var wg sync.WaitGroup

	// Process main messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second) // Process for 40s
		defer cancel()

		if err := manager.ProcessMessages(ctx, MessageProcessor); err != nil {
			log.Printf("Error processing main messages: %v", err)
		}
	}()

	// Process DLQ messages (wait longer for messages to reach DLQ after retries)
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Wait for messages to fail and be moved to DLQ
		log.Printf("\n=== Waiting for messages to be moved to DLQ (after %d retries) ===", maxRetries)
		time.Sleep(45 * time.Second) // Wait longer for DLQ messages

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := manager.ProcessDLQMessages(ctx); err != nil {
			log.Printf("Error processing DLQ messages: %v", err)
		}
	}()

	wg.Wait()

	log.Println("\n=== Demo Summary ===")
	fmt.Printf("Project ID: %s\n", projectID)
	fmt.Printf("Main Topic: %s\n", topicID)
	fmt.Printf("Main Subscription: %s\n", subscriptionID)
	fmt.Printf("DLQ Topic: %s\n", dlqTopicID)
	fmt.Printf("DLQ Subscription: %s\n", dlqSubscriptionID)
	fmt.Printf("Messages published: %d\n", len(testMessages))
	fmt.Printf("Max retry attempts: %d\n", maxRetries)
	fmt.Printf("Ack deadline: %v\n", ackDeadline)

	log.Println("Demo completed!")
}
