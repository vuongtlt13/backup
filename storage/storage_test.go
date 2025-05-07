package storage

import (
	"os"
	"path/filepath"
	"testing"

	"backupdb/config"
	"backupdb/logger"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewStorageService(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Storage: map[string]config.StorageConfig{
			"s3_test": {
				Kind:      "s3",
				Enabled:   true,
				Region:    "us-west-2",
				Bucket:    "test-bucket",
				Path:      "test/path",
				AccessKey: "test-key",
				SecretKey: "test-secret",
			},
			"rsync_test": {
				Kind:         "rsync",
				Enabled:      true,
				TargetServer: "test-server",
				TargetPath:   "/backup",
				User:        "test-user",
				Port:        22,
			},
			"disabled_test": {
				Kind:    "s3",
				Enabled: false,
			},
		},
	}

	// Create storage service
	service := NewStorageService(cfg)
	assert.NotNil(t, service, "Storage service should not be nil")

	// Verify providers
	assert.NotNil(t, service.providers["s3_test"], "S3 provider should be initialized")
	assert.NotNil(t, service.providers["rsync_test"], "Rsync provider should be initialized")
	assert.Nil(t, service.providers["disabled_test"], "Disabled provider should not be initialized")
}

func TestSendToStorage(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Storage: map[string]config.StorageConfig{
			"rsync_test": {
				Kind:         "rsync",
				Enabled:      true,
				TargetServer: "test-server",
				TargetPath:   "/backup",
				User:        "test-user",
				Port:        22,
			},
		},
	}

	// Create storage service
	service := NewStorageService(cfg)

	// Create test file
	testFile := "test_backup.tar.gz"
	content := []byte("test content")
	err := os.WriteFile(testFile, content, 0644)
	assert.NoError(t, err, "Failed to create test file")
	defer os.Remove(testFile)

	// Test sending to storage
	err = service.SendToStorage(testFile, []string{"rsync_test"})
	assert.Error(t, err, "Should return error for non-existent server")
}

func TestGetProvider(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Storage: map[string]config.StorageConfig{
			"s3_test": {
				Kind:    "s3",
				Enabled: true,
			},
		},
	}

	// Create storage service
	service := NewStorageService(cfg)

	// Test getting existing provider
	provider, err := service.GetProvider("s3_test")
	assert.NoError(t, err, "Should not return error for existing provider")
	assert.NotNil(t, provider, "Provider should not be nil")

	// Test getting non-existent provider
	provider, err = service.GetProvider("non_existent")
	assert.Error(t, err, "Should return error for non-existent provider")
	assert.Nil(t, provider, "Provider should be nil")
}

func TestStorageProviderNames(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Storage: map[string]config.StorageConfig{
			"s3_test": {
				Kind:    "s3",
				Enabled: true,
			},
			"rsync_test": {
				Kind:    "rsync",
				Enabled: true,
			},
			"gdrive_test": {
				Kind:    "google_drive",
				Enabled: true,
			},
		},
	}

	// Create storage service
	service := NewStorageService(cfg)

	// Test provider names
	s3Provider, _ := service.GetProvider("s3_test")
	assert.Equal(t, "s3_test", s3Provider.Name(), "S3 provider name should match")

	rsyncProvider, _ := service.GetProvider("rsync_test")
	assert.Equal(t, "rsync_test", rsyncProvider.Name(), "Rsync provider name should match")

	gdriveProvider, _ := service.GetProvider("gdrive_test")
	assert.Equal(t, "gdrive_test", gdriveProvider.Name(), "Google Drive provider name should match")
} 