package storage

import (
	"testing"

	"backupdb/config"

	"github.com/stretchr/testify/assert"
)

func TestNewS3Provider(t *testing.T) {
	// Test with valid config but invalid credentials
	cfg := config.StorageConfig{
		Enabled:         true,
		Kind:            "s3",
		Bucket:          "test-bucket",
		Region:          "us-west-2",
		AccessKeyID:     "invalid-key",
		SecretAccessKey: "invalid-secret",
	}

	provider, err := NewS3Provider(cfg)
	assert.Error(t, err) // Should error because credentials are invalid
	assert.Nil(t, provider)

	// Test with disabled config
	disabledCfg := config.StorageConfig{
		Enabled: false,
		Kind:    "s3",
	}

	provider, err = NewS3Provider(disabledCfg)
	assert.Error(t, err)
	assert.Nil(t, provider)

	// Test with missing credentials
	invalidCfg := config.StorageConfig{
		Enabled: true,
		Kind:    "s3",
		Bucket:  "test-bucket",
		Region:  "us-west-2",
	}

	provider, err = NewS3Provider(invalidCfg)
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func TestS3ProviderSendFile(t *testing.T) {
	// Create test config with invalid credentials
	cfg := config.StorageConfig{
		Enabled:         true,
		Kind:            "s3",
		Bucket:          "test-bucket",
		Region:          "us-west-2",
		AccessKeyID:     "invalid-key",
		SecretAccessKey: "invalid-secret",
	}

	provider, err := NewS3Provider(cfg)
	assert.Error(t, err) // Should error because credentials are invalid
	assert.Nil(t, provider)
}
