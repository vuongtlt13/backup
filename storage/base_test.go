package storage

import (
	"testing"

	"backupdb/config"

	"github.com/stretchr/testify/assert"
)

func TestNewStorageService(t *testing.T) {
	// Test with valid config
	cfg := &config.Config{
		Storage: map[string]config.StorageConfig{
			"s3": {
				Enabled:         true,
				Kind:            "s3",
				Bucket:          "test-bucket",
				Region:          "us-west-2",
				AccessKeyID:     "test-access-key",
				SecretAccessKey: "test-secret-key",
			},
			"rsync": {
				Enabled:  true,
				Kind:     "rsync",
				Server:   "test-server",
				Username: "test-user",
				Path:     "/backup",
			},
		},
	}

	service := NewStorageService(cfg)
	assert.NotNil(t, service)
	assert.Equal(t, 1, len(service.providers)) // Only rsync provider should be initialized since S3 credentials are invalid

	// Test with invalid config
	invalidCfg := &config.Config{
		Storage: map[string]config.StorageConfig{
			"invalid": {
				Enabled: true,
				Kind:    "invalid",
			},
		},
	}

	service = NewStorageService(invalidCfg)
	assert.NotNil(t, service)
	assert.Equal(t, 0, len(service.providers))
}

func TestSendToStorage(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		Storage: map[string]config.StorageConfig{
			"s3": {
				Enabled:         true,
				Kind:            "s3",
				Bucket:          "test-bucket",
				Region:          "us-west-2",
				AccessKeyID:     "test-access-key",
				SecretAccessKey: "test-secret-key",
			},
		},
	}

	// Create service
	service := NewStorageService(cfg)
	assert.NotNil(t, service)

	// Test sending to non-existent storage
	err := service.SendToStorage("test.txt", []string{"non-existent"}, "backup-name")
	assert.Error(t, err)

	// Test sending to valid storage
	err = service.SendToStorage("test.txt", []string{"s3"}, "backup-name")
	assert.Error(t, err) // Should error because we can't actually connect to S3
}
