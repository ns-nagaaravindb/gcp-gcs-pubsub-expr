package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// GCSClientBucketCreation wraps the Google Cloud Storage client
type GCSClientBucketCreation struct {
	client    *storage.Client
	projectID string
}

// NewGCSClient creates a new GCS client
func NewGCSClient(ctx context.Context, projectID string) (*GCSClientBucketCreation, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	return &GCSClientBucketCreation{
		client:    client,
		projectID: projectID,
	}, nil
}

// Close closes the GCS client
func (g *GCSClientBucketCreation) Close() error {
	return g.client.Close()
}

// CreateBucket creates a new bucket in Google Cloud Storage
func (g *GCSClientBucketCreation) CreateBucket(ctx context.Context, bucketName, region string) error {
	bucket := g.client.Bucket(bucketName)

	// Check if bucket already exists
	_, err := bucket.Attrs(ctx)
	if err == nil {
		fmt.Printf("Bucket %s already exists\n", bucketName)
		return nil
	}

	bucketAttrs := &storage.BucketAttrs{
		Location: region,
	}

	if err := bucket.Create(ctx, g.projectID, bucketAttrs); err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
	}

	fmt.Printf("Successfully created bucket: %s\n", bucketName)
	return nil
}

// UploadFile uploads a file to the specified bucket
func (g *GCSClientBucketCreation) UploadFile(ctx context.Context, bucketName, objectName string, content []byte) error {
	bucket := g.client.Bucket(bucketName)
	obj := bucket.Object(objectName)

	// Create a writer to upload the file
	writer := obj.NewWriter(ctx)
	defer writer.Close()

	// Set content type
	writer.ContentType = "text/plain"

	// Write the content
	if _, err := writer.Write(content); err != nil {
		return fmt.Errorf("failed to write to object %s: %w", objectName, err)
	}

	fmt.Printf("Successfully uploaded file: %s to bucket: %s\n", objectName, bucketName)
	return nil
}

// ListFiles lists all files in the specified bucket
func (g *GCSClientBucketCreation) ListFiles(ctx context.Context, bucketName string) error {
	bucket := g.client.Bucket(bucketName)

	it := bucket.Objects(ctx, nil)
	fmt.Printf("Files in bucket %s:\n", bucketName)
	fmt.Println(strings.Repeat("-", 50))

	fileCount := 0
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to iterate over objects: %w", err)
		}

		fileCount++
		fmt.Printf("File: %s\n", attrs.Name)
		fmt.Printf("  Size: %d bytes\n", attrs.Size)
		fmt.Printf("  Created: %s\n", attrs.Created.Format(time.RFC3339))
		fmt.Printf("  Updated: %s\n", attrs.Updated.Format(time.RFC3339))
		fmt.Printf("  Content Type: %s\n", attrs.ContentType)
		fmt.Println()
	}

	if fileCount == 0 {
		fmt.Println("No files found in the bucket.")
	} else {
		fmt.Printf("Total files: %d\n", fileCount)
	}
	fmt.Println(strings.Repeat("-", 50))

	return nil
}

// DownloadFile downloads a file from the bucket and returns its content
func (g *GCSClientBucketCreation) DownloadFile(ctx context.Context, bucketName, objectName string) ([]byte, error) {
	bucket := g.client.Bucket(bucketName)
	obj := bucket.Object(objectName)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader for object %s: %w", objectName, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read object %s: %w", objectName, err)
	}

	fmt.Printf("Successfully downloaded file: %s from bucket: %s\n", objectName, bucketName)
	return data, nil
}

// DeleteFile deletes a file from the specified bucket
func (g *GCSClientBucketCreation) DeleteFile(ctx context.Context, bucketName, objectName string) error {
	bucket := g.client.Bucket(bucketName)
	obj := bucket.Object(objectName)

	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete object %s: %w", objectName, err)
	}

	fmt.Printf("Successfully deleted file: %s from bucket: %s\n", objectName, bucketName)
	return nil
}

// DeleteBucket deletes a bucket from Google Cloud Storage
// Note: The bucket must be empty before it can be deleted
func (g *GCSClientBucketCreation) DeleteBucket(ctx context.Context, bucketName string) error {
	bucket := g.client.Bucket(bucketName)

	// Check if bucket exists
	_, err := bucket.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("bucket %s does not exist or cannot be accessed: %w", bucketName, err)
	}

	// Delete the bucket
	if err := bucket.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete bucket %s: %w", bucketName, err)
	}

	fmt.Printf("Successfully deleted bucket: %s\n", bucketName)
	return nil
}

func main() {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/Users/nagaaravindb/Downloads/cred/data-qe-da7e1252-3baad2049955.json")
	projectID := "data-qe-da7e1252" // Using project ID instead of project number
	bucketName := "ns-nonprod-eng-data-pipeline-events-us-west1-dev"
	region := "us-central1"
	fileName := "test-file.txt"
	fileContent := []byte("Hello, Google Cloud Storage!\nThis is a test file created by Go program.\nTimestamp: " + time.Now().Format(time.RFC3339))

	ctx := context.Background()

	// Create GCS client
	fmt.Println("Creating GCS client...")
	client, err := NewGCSClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create GCS client: %v", err)
	}
	defer client.Close()

	fmt.Printf("Project ID: %s\n", projectID)
	fmt.Printf("Bucket Name: %s\n", bucketName)
	fmt.Printf("Region: %s\n", region)
	fmt.Printf("Test File: %s\n", fileName)
	fmt.Println()

	// Step 1: Create bucket
	fmt.Println("Step 1: Creating bucket...")
	if err := client.CreateBucket(ctx, bucketName, region); err != nil {
		log.Printf("Failed to create bucket: %v", err)
		return
	}
	fmt.Println()

	// Step 2: Upload file
	fmt.Println("Step 2: Uploading file...")
	if err := client.UploadFile(ctx, bucketName, fileName, fileContent); err != nil {
		log.Printf("Failed to upload file: %v", err)
		return
	}
	fmt.Println()

	// Step 3: List files in bucket
	fmt.Println("Step 3: Listing files in bucket...")
	if err := client.ListFiles(ctx, bucketName); err != nil {
		log.Printf("Failed to list files: %v", err)
		return
	}
	fmt.Println()

	// Step 4: Download and display file content
	fmt.Println("Step 4: Downloading file...")
	downloadedContent, err := client.DownloadFile(ctx, bucketName, fileName)
	if err != nil {
		log.Printf("Failed to download file: %v", err)
		return
	}
	fmt.Printf("Downloaded file content:\n%s\n", string(downloadedContent))
	fmt.Println()

	//Step 5: Delete file
	fmt.Println("Step 5: Deleting file...")
	if err := client.DeleteFile(ctx, bucketName, fileName); err != nil {
		log.Printf("Failed to delete file: %v", err)
		return
	}
	fmt.Println()

	// Step 6: Verify file deletion by listing files again
	fmt.Println("Step 6: Verifying file deletion...")
	if err := client.ListFiles(ctx, bucketName); err != nil {
		log.Printf("Failed to list files: %v", err)
		return
	}
	fmt.Println()

	// Step 7: Delete bucket (only if it's empty)
	fmt.Println("Step 7: Deleting bucket...")
	if err := client.DeleteBucket(ctx, bucketName); err != nil {
		log.Printf("Failed to delete bucket: %v", err)
		return
	}
	fmt.Println()

	fmt.Println("=" + strings.Repeat("=", 60) + "=")
	fmt.Println("Google Cloud Storage Test Completed Successfully!")
	fmt.Println("All operations including bucket deletion completed!")
	fmt.Println("=" + strings.Repeat("=", 60) + "=")
}
