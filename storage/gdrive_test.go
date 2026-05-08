package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"backupdb/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestNewGoogleDriveProvider(t *testing.T) {
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

	cfg := config.StorageConfig{
		Enabled:         true,
		Kind:            "google_drive",
		CredentialsFile: credentialsFile,
		FolderID:        "test-folder-id",
	}

	provider, err := NewGoogleDriveProvider(cfg)
	assert.Error(t, err)
	assert.Nil(t, provider)

	disabledCfg := config.StorageConfig{
		Enabled: false,
		Kind:    "google_drive",
	}

	provider, err = NewGoogleDriveProvider(disabledCfg)
	assert.Error(t, err)
	assert.Nil(t, provider)

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

func TestGoogleDriveProviderAuthValidation(t *testing.T) {
	tmpDir := t.TempDir()
	clientSecretFile := filepath.Join(tmpDir, "client-secret.json")
	tokenFile := filepath.Join(tmpDir, "token.json")
	require.NoError(t, os.WriteFile(clientSecretFile, []byte(`{"installed":{"client_id":"test-client","client_secret":"test-secret","redirect_uris":["http://localhost"]}}`), 0644))
	require.NoError(t, os.WriteFile(tokenFile, []byte(`not-json`), 0600))

	tests := []struct {
		name      string
		cfg       config.StorageConfig
		wantError string
	}{
		{
			name: "unknown auth mode",
			cfg: config.StorageConfig{
				Enabled:  true,
				Kind:     "google_drive",
				AuthMode: "unknown",
				FolderID: "folder-id",
			},
			wantError: "unsupported google drive auth_mode",
		},
		{
			name: "missing oauth client secret file",
			cfg: config.StorageConfig{
				Enabled:   true,
				Kind:      "google_drive",
				AuthMode:  "oauth_user",
				TokenFile: tokenFile,
				FolderID:  "folder-id",
			},
			wantError: "OAuth client secret file is required",
		},
		{
			name: "missing oauth token file",
			cfg: config.StorageConfig{
				Enabled:          true,
				Kind:             "google_drive",
				AuthMode:         "oauth_user",
				ClientSecretFile: clientSecretFile,
				FolderID:         "folder-id",
			},
			wantError: "OAuth token file is required",
		},
		{
			name: "invalid oauth token json",
			cfg: config.StorageConfig{
				Enabled:          true,
				Kind:             "google_drive",
				AuthMode:         "oauth_user",
				ClientSecretFile: clientSecretFile,
				TokenFile:        tokenFile,
				FolderID:         "folder-id",
			},
			wantError: "failed to load Google Drive OAuth token",
		},
		{
			name: "missing folder id",
			cfg: config.StorageConfig{
				Enabled:          true,
				Kind:             "google_drive",
				AuthMode:         "oauth_user",
				ClientSecretFile: clientSecretFile,
				TokenFile:        tokenFile,
			},
			wantError: "folder ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGoogleDriveProvider(tt.cfg)
			assert.Error(t, err)
			assert.Nil(t, provider)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestGoogleDriveOAuthTokenLoadSave(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "nested", "token.json")
	token := &oauth2.Token{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	require.NoError(t, SaveOAuthToken(tokenFile, token))

	info, err := os.Stat(tokenFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	loadedToken, err := LoadOAuthToken(tokenFile)
	require.NoError(t, err)
	assert.Equal(t, token.AccessToken, loadedToken.AccessToken)
	assert.Equal(t, token.RefreshToken, loadedToken.RefreshToken)
	assert.Equal(t, token.TokenType, loadedToken.TokenType)
}

func TestGoogleDriveBackupListQuery(t *testing.T) {
	query := googleDriveBackupListQuery(`folder'id\\x`, `mysql_data'prod`)

	assert.Contains(t, query, `'folder\'id\\\\x' in parents`)
	assert.Contains(t, query, "trashed = false")
	assert.Contains(t, query, "mimeType != 'application/vnd.google-apps.folder'")
	assert.Contains(t, query, `name contains 'mysql_data\'prod_'`)
}

func TestParseGoogleDriveBackupFile(t *testing.T) {
	file, ok := parseGoogleDriveBackupFile("file-id", "mysql_data_20260508010203_123456789.tar.gz", "mysql_data")
	assert.True(t, ok)
	assert.Equal(t, "file-id", file.ID)
	assert.Equal(t, "mysql_data_20260508010203_123456789.tar.gz", file.Name)
	assert.Equal(t, time.Date(2026, 5, 8, 1, 2, 3, 0, time.UTC), file.Timestamp)

	file, ok = parseGoogleDriveBackupFile("file-id", "mysql_data_20260508010203.tar.gz", "mysql_data")
	assert.True(t, ok)
	assert.Equal(t, "mysql_data_20260508010203.tar.gz", file.Name)

	_, ok = parseGoogleDriveBackupFile("file-id", "mysql_data_extra_20260508010203_123456789.tar.gz", "mysql_data")
	assert.False(t, ok)

	_, ok = parseGoogleDriveBackupFile("file-id", "postgres_data_20260508010203_123456789.tar.gz", "mysql_data")
	assert.False(t, ok)

	_, ok = parseGoogleDriveBackupFile("file-id", "mysql_data_20260508010203_123456789.zip", "mysql_data")
	assert.False(t, ok)

	_, ok = parseGoogleDriveBackupFile("file-id", "mysql_data_20260532010203_123456789.tar.gz", "mysql_data")
	assert.False(t, ok)
}

func TestSelectGoogleDriveBackupsToDelete(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	files := []googleDriveBackupFile{
		{ID: "day-3", Name: "mysql_data_20260508030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 3, 0, 0, 0, time.UTC)},
		{ID: "day-2", Name: "mysql_data_20260508020000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 2, 0, 0, 0, time.UTC)},
		{ID: "day-1", Name: "mysql_data_20260508010000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 1, 0, 0, 0, time.UTC)},
		{ID: "day-0", Name: "mysql_data_20260508000000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)},
		{ID: "month-new", Name: "mysql_data_20260402000000_000000001.tar.gz", Timestamp: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)},
		{ID: "month-old", Name: "mysql_data_20260401000000_000000001.tar.gz", Timestamp: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "year-new", Name: "mysql_data_20250301000000_000000001.tar.gz", Timestamp: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "year-old", Name: "mysql_data_20250201000000_000000001.tar.gz", Timestamp: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
	}

	toDelete := selectGoogleDriveBackupsToDelete(files, config.RemoteRetentionConfig{
		Enabled:     true,
		MaxPerDay:   3,
		MaxPerMonth: 1,
		MaxPerYear:  1,
	}, now)

	assert.ElementsMatch(t, []googleDriveBackupFile{
		{ID: "day-0", Name: "mysql_data_20260508000000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)},
		{ID: "month-old", Name: "mysql_data_20260401000000_000000001.tar.gz", Timestamp: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "year-old", Name: "mysql_data_20250201000000_000000001.tar.gz", Timestamp: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
	}, toDelete)
}

func TestSelectGoogleDriveBackupsToDeleteZeroMaxKeepsTier(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	files := []googleDriveBackupFile{
		{ID: "new", Name: "mysql_data_20260508020000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 2, 0, 0, 0, time.UTC)},
		{ID: "old", Name: "mysql_data_20260508010000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 1, 0, 0, 0, time.UTC)},
	}

	toDelete := selectGoogleDriveBackupsToDelete(files, config.RemoteRetentionConfig{
		Enabled:     true,
		MaxPerDay:   0,
		MaxPerMonth: 1,
		MaxPerYear:  1,
	}, now)

	assert.Empty(t, toDelete)
}

func TestGoogleDriveProviderSendFile(t *testing.T) {
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

	cfg := config.StorageConfig{
		Enabled:         true,
		Kind:            "google_drive",
		CredentialsFile: credentialsFile,
		FolderID:        "test-folder-id",
	}

	provider, err := NewGoogleDriveProvider(cfg)
	assert.Error(t, err)
	assert.Nil(t, provider)
}
