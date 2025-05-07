package storage

import (
	"os"
	"path/filepath"
	"testing"

	"backupdb/config"
)

func TestNewStorageService(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		Storage: map[string]config.StorageConfig{
			"s3_1": {
				Kind:      "s3",
				Enabled:   true,
				Bucket:    "test-bucket",
				Region:    "us-west-2",
				AccessKey: "test-key",
				SecretKey: "test-secret",
				Path:      "backups/",
			},
			"rsync_1": {
				Kind:         "rsync",
				Enabled:      true,
				TargetServer: "test-server",
				TargetPath:   "/backup/",
				User:         "test-user",
				Port:         22,
			},
			"disabled_storage": {
				Kind:    "s3",
				Enabled: false,
			},
		},
	}

	// Create storage service
	service := NewStorageService(cfg)

	// Test provider initialization
	tests := []struct {
		name     string
		expected bool
	}{
		{"s3_1", true},
		{"rsync_1", true},
		{"disabled_storage", false},
		{"non_existent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := service.GetProvider(tt.name)
			if tt.expected {
				if err != nil {
					t.Errorf("expected provider %s to exist, got error: %v", tt.name, err)
				}
				if provider == nil {
					t.Errorf("expected provider %s to be initialized", tt.name)
				}
			} else {
				if err == nil {
					t.Errorf("expected provider %s to not exist", tt.name)
				}
			}
		})
	}
}

func TestSendToStorage(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create test config with mock providers
	cfg := &config.Config{
		Storage: map[string]config.StorageConfig{
			"mock_s3": {
				Kind:    "s3",
				Enabled: true,
			},
			"mock_rsync": {
				Kind:    "rsync",
				Enabled: true,
			},
		},
	}

	// Create storage service with mock providers
	service := NewStorageService(cfg)

	// Test sending to multiple storages
	storageNames := []string{"mock_s3", "mock_rsync"}
	err := service.SendToStorage(testFile, storageNames)
	if err != nil {
		t.Errorf("SendToStorage failed: %v", err)
	}

	// Test sending to non-existent storage
	err = service.SendToStorage(testFile, []string{"non_existent"})
	if err == nil {
		t.Error("expected error when sending to non-existent storage")
	}
} 