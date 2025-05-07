package storage

import (
	"testing"

	"backupdb/config"
)

func TestS3Provider(t *testing.T) {
	// Create test config
	cfg := config.StorageConfig{
		Kind:      "s3",
		Enabled:   true,
		Bucket:    "test-bucket",
		Region:    "us-west-2",
		AccessKey: "test-key",
		SecretKey: "test-secret",
		Path:      "backups/",
	}

	// Create provider
	provider := NewS3Provider("test_s3", cfg)
	if provider == nil {
		t.Fatal("failed to create S3 provider")
	}

	// Test provider name
	if name := provider.Name(); name != "test_s3" {
		t.Errorf("expected provider name 'test_s3', got '%s'", name)
	}

	// Test with invalid config
	invalidCfg := config.StorageConfig{
		Kind:    "s3",
		Enabled: true,
		// Missing required fields
	}

	provider = NewS3Provider("invalid_s3", invalidCfg)
	if provider != nil {
		t.Error("expected nil provider with invalid config")
	}
}

func TestS3ProviderSend(t *testing.T) {
	// Create test config
	cfg := config.StorageConfig{
		Kind:      "s3",
		Enabled:   true,
		Bucket:    "test-bucket",
		Region:    "us-west-2",
		AccessKey: "test-key",
		SecretKey: "test-secret",
		Path:      "backups/",
	}

	// Create provider
	provider := NewS3Provider("test_s3", cfg)
	if provider == nil {
		t.Fatal("failed to create S3 provider")
	}

	// Test sending file
	// Note: This is a mock test since we don't want to actually send to S3
	// In a real test, you would use a mock S3 client
	err := provider.Send("test.txt")
	if err == nil {
		t.Error("expected error when sending to S3 without proper credentials")
	}
} 