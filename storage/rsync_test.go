package storage

import (
	"testing"

	"backupdb/config"

	"github.com/stretchr/testify/assert"
)

func TestNewRsyncProvider(t *testing.T) {
	// Test with valid config
	cfg := config.StorageConfig{
		Enabled:  true,
		Kind:     "rsync",
		Server:   "test-server",
		Username: "test-user",
		Path:     "/backup",
	}

	provider, err := NewRsyncProvider(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "rsync", provider.GetName())

	// Test with disabled config
	disabledCfg := config.StorageConfig{
		Enabled: false,
		Kind:    "rsync",
	}

	provider, err = NewRsyncProvider(disabledCfg)
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func TestRsyncProviderSendFile(t *testing.T) {
	// Create test config
	cfg := config.StorageConfig{
		Enabled:  true,
		Kind:     "rsync",
		Server:   "test-server",
		Username: "test-user",
		Path:     "/backup",
	}

	provider, err := NewRsyncProvider(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, provider)

	// Test sending non-existent file
	err = provider.SendFile("non-existent.txt")
	assert.Error(t, err)
}
