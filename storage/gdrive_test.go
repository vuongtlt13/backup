package storage

import (
	"os"
	"path/filepath"
	"testing"

	"backupdb/config"

	"github.com/stretchr/testify/assert"
)

func TestNewGoogleDriveProvider(t *testing.T) {
	// Create temporary credentials file with invalid credentials
	tmpDir := t.TempDir()
	credentialsFile := filepath.Join(tmpDir, "credentials.json")
	err := os.WriteFile(credentialsFile, []byte(`{
		"type": "service_account",
		"project_id": "test-project",
		"private_key_id": "test-key-id",
		"private_key": "invalid-key",
		"client_email": "test@test.com",
		"client_id": "test-client-id"
	}`), 0644)
	assert.NoError(t, err)

	// Test with invalid credentials
	cfg := config.StorageConfig{
		Enabled:         true,
		Kind:            "google_drive",
		CredentialsFile: credentialsFile,
		FolderID:        "test-folder-id",
	}

	provider, err := NewGoogleDriveProvider(cfg)
	assert.Error(t, err) // Should error because credentials are invalid
	assert.Nil(t, provider)

	// Test with disabled config
	disabledCfg := config.StorageConfig{
		Enabled: false,
		Kind:    "google_drive",
	}

	provider, err = NewGoogleDriveProvider(disabledCfg)
	assert.Error(t, err)
	assert.Nil(t, provider)

	// Test with missing credentials file
	invalidCfg := config.StorageConfig{
		Enabled:         true,
		Kind:            "google_drive",
		CredentialsFile: "non-existent.json",
		FolderID:        "test-folder-id",
	}

	provider, err = NewGoogleDriveProvider(invalidCfg)
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func TestGoogleDriveProviderSendFile(t *testing.T) {
	// Create temporary credentials file with invalid credentials
	tmpDir := t.TempDir()
	credentialsFile := filepath.Join(tmpDir, "credentials.json")
	err := os.WriteFile(credentialsFile, []byte(`{
		"type": "service_account",
		"project_id": "test-project",
		"private_key_id": "test-key-id",
		"private_key": "invalid-key",
		"client_email": "test@test.com",
		"client_id": "test-client-id"
	}`), 0644)
	assert.NoError(t, err)

	// Create test config
	cfg := config.StorageConfig{
		Enabled:         true,
		Kind:            "google_drive",
		CredentialsFile: credentialsFile,
		FolderID:        "test-folder-id",
	}

	provider, err := NewGoogleDriveProvider(cfg)
	assert.Error(t, err) // Should error because credentials are invalid
	assert.Nil(t, provider)
}
