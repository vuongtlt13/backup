package storage

import (
	"testing"

	"backupdb/config"
)

func TestGoogleDriveProvider(t *testing.T) {
	// Create test config
	cfg := config.StorageConfig{
		Kind:            "google_drive",
		Enabled:         true,
		CredentialsFile: "test-credentials.json",
		FolderID:        "test-folder",
	}

	// Create provider
	provider := NewGoogleDriveProvider("test_gdrive", cfg)
	if provider == nil {
		t.Fatal("failed to create Google Drive provider")
	}

	// Test provider name
	if name := provider.Name(); name != "test_gdrive" {
		t.Errorf("expected provider name 'test_gdrive', got '%s'", name)
	}

	// Test with invalid config
	invalidCfg := config.StorageConfig{
		Kind:    "google_drive",
		Enabled: true,
		// Missing required fields
	}

	provider = NewGoogleDriveProvider("invalid_gdrive", invalidCfg)
	if provider != nil {
		t.Error("expected nil provider with invalid config")
	}
}

func TestGoogleDriveProviderSend(t *testing.T) {
	// Create test config
	cfg := config.StorageConfig{
		Kind:            "google_drive",
		Enabled:         true,
		CredentialsFile: "test-credentials.json",
		FolderID:        "test-folder",
	}

	// Create provider
	provider := NewGoogleDriveProvider("test_gdrive", cfg)
	if provider == nil {
		t.Fatal("failed to create Google Drive provider")
	}

	// Test sending file
	// Note: This is a mock test since we don't want to actually send to Google Drive
	// In a real test, you would use a mock Google Drive client
	err := provider.Send("test.txt")
	if err == nil {
		t.Error("expected error when sending to Google Drive without proper credentials")
	}
} 