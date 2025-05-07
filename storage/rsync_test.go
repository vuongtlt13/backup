package storage

import (
	"testing"

	"backupdb/config"
)

func TestRsyncProvider(t *testing.T) {
	// Create test config
	cfg := config.StorageConfig{
		Kind:         "rsync",
		Enabled:      true,
		TargetServer: "test-server",
		TargetPath:   "/backup/",
		User:         "test-user",
		Port:         22,
	}

	// Create provider
	provider := NewRsyncProvider("test_rsync", cfg)
	if provider == nil {
		t.Fatal("failed to create Rsync provider")
	}

	// Test provider name
	if name := provider.Name(); name != "test_rsync" {
		t.Errorf("expected provider name 'test_rsync', got '%s'", name)
	}

	// Test with invalid config
	invalidCfg := config.StorageConfig{
		Kind:    "rsync",
		Enabled: true,
		// Missing required fields
	}

	provider = NewRsyncProvider("invalid_rsync", invalidCfg)
	if provider != nil {
		t.Error("expected nil provider with invalid config")
	}
}

func TestRsyncProviderSend(t *testing.T) {
	// Create test config
	cfg := config.StorageConfig{
		Kind:         "rsync",
		Enabled:      true,
		TargetServer: "test-server",
		TargetPath:   "/backup/",
		User:         "test-user",
		Port:         22,
	}

	// Create provider
	provider := NewRsyncProvider("test_rsync", cfg)
	if provider == nil {
		t.Fatal("failed to create Rsync provider")
	}

	// Test sending file
	// Note: This is a mock test since we don't want to actually send via rsync
	// In a real test, you would use a mock rsync command
	err := provider.Send("test.txt")
	if err == nil {
		t.Error("expected error when sending via rsync without proper connection")
	}
} 