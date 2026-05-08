package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"backupdb/config"
	"backupdb/logger"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	googleDriveAuthModeServiceAccount = "service_account"
	googleDriveAuthModeOAuthUser      = "oauth_user"
	googleDriveFolderMimeType         = "application/vnd.google-apps.folder"
)

type googleDriveBackupFile struct {
	ID        string
	Name      string
	Timestamp time.Time
}

// GoogleDriveProvider implements StorageProvider for Google Drive
type GoogleDriveProvider struct {
	service *drive.Service
	config  config.StorageConfig
	log     *logger.Logger
}

func googleDriveAuthMode(cfg config.StorageConfig) string {
	if cfg.AuthMode == "" {
		return googleDriveAuthModeServiceAccount
	}
	return cfg.AuthMode
}

// NewGoogleDriveProvider creates a new Google Drive storage provider
func NewGoogleDriveProvider(cfg config.StorageConfig) (*GoogleDriveProvider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("google drive provider is disabled")
	}
	if cfg.FolderID == "" {
		return nil, fmt.Errorf("google drive folder ID is required")
	}

	ctx := context.Background()
	client, err := newGoogleDriveHTTPClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %v", err)
	}

	_, err = service.Files.Get(cfg.FolderID).SupportsAllDrives(true).Fields("id, name").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to validate Google Drive credentials: %v", err)
	}

	return &GoogleDriveProvider{
		service: service,
		config:  cfg,
		log:     logger.Get(),
	}, nil
}

func newGoogleDriveHTTPClient(ctx context.Context, cfg config.StorageConfig) (*http.Client, error) {
	switch googleDriveAuthMode(cfg) {
	case googleDriveAuthModeServiceAccount:
		return newServiceAccountHTTPClient(ctx, cfg)
	case googleDriveAuthModeOAuthUser:
		return newOAuthUserHTTPClient(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported google drive auth_mode: %s", cfg.AuthMode)
	}
}

func newServiceAccountHTTPClient(ctx context.Context, cfg config.StorageConfig) (*http.Client, error) {
	if cfg.CredentialsFile == "" {
		return nil, fmt.Errorf("google drive credentials file is required")
	}

	credentials, err := os.ReadFile(cfg.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %v", err)
	}

	jwtConfig, err := google.JWTConfigFromJSON(credentials, drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT config: %v", err)
	}

	return jwtConfig.Client(ctx), nil
}

func newOAuthUserHTTPClient(ctx context.Context, cfg config.StorageConfig) (*http.Client, error) {
	oauthConfig, err := NewGoogleDriveOAuthConfig(cfg)
	if err != nil {
		return nil, err
	}

	token, err := LoadOAuthToken(cfg.TokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load Google Drive OAuth token from %s: %v; run -gdrive-auth-init with this storage name", cfg.TokenFile, err)
	}

	return oauthConfig.Client(ctx, token), nil
}

func NewGoogleDriveOAuthConfig(cfg config.StorageConfig) (*oauth2.Config, error) {
	if cfg.ClientSecretFile == "" {
		return nil, fmt.Errorf("google drive OAuth client secret file is required")
	}
	if cfg.TokenFile == "" {
		return nil, fmt.Errorf("google drive OAuth token file is required")
	}

	clientSecret, err := os.ReadFile(cfg.ClientSecretFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read OAuth client secret file: %v", err)
	}

	oauthConfig, err := google.ConfigFromJSON(clientSecret, drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth config: %v", err)
	}

	return oauthConfig, nil
}

func LoadOAuthToken(path string) (*oauth2.Token, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var token oauth2.Token
	if err := json.NewDecoder(file).Decode(&token); err != nil {
		return nil, err
	}
	if !token.Valid() && token.RefreshToken == "" {
		return nil, fmt.Errorf("token is invalid and has no refresh token")
	}

	return &token, nil
}

func SaveOAuthToken(path string, token *oauth2.Token) error {
	if token == nil {
		return fmt.Errorf("OAuth token is nil")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %v", err)
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create token file: %v", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(token); err != nil {
		return fmt.Errorf("failed to write token file: %v", err)
	}

	return nil
}

func googleDriveQueryValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `'`, `\'`)
}

func googleDriveBackupListQuery(folderID, backupName string) string {
	return fmt.Sprintf("'%s' in parents and trashed = false and mimeType != '%s' and name contains '%s_'",
		googleDriveQueryValue(folderID),
		googleDriveFolderMimeType,
		googleDriveQueryValue(backupName),
	)
}

func parseGoogleDriveBackupFile(id, name, backupName string) (googleDriveBackupFile, bool) {
	pattern := fmt.Sprintf(`^%s_(\d{14})(?:_\d{1,9})?\.tar\.gz$`, regexp.QuoteMeta(backupName))
	matches := regexp.MustCompile(pattern).FindStringSubmatch(name)
	if len(matches) != 2 {
		return googleDriveBackupFile{}, false
	}

	timestamp, err := time.Parse("20060102150405", matches[1])
	if err != nil {
		return googleDriveBackupFile{}, false
	}

	return googleDriveBackupFile{ID: id, Name: name, Timestamp: timestamp}, true
}

func selectGoogleDriveBackupsToDelete(files []googleDriveBackupFile, retention config.RemoteRetentionConfig, now time.Time) []googleDriveBackupFile {
	if len(files) == 0 {
		return nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Timestamp.After(files[j].Timestamp)
	})

	keep := make(map[string]bool)
	currentYear, currentMonth, _ := now.Date()
	daily := make(map[string][]googleDriveBackupFile)
	monthly := make(map[string][]googleDriveBackupFile)
	yearly := make(map[string][]googleDriveBackupFile)

	for _, file := range files {
		year, month, _ := file.Timestamp.Date()
		switch {
		case year == currentYear && month == currentMonth:
			dayKey := file.Timestamp.Format("2006-01-02")
			daily[dayKey] = append(daily[dayKey], file)
		case year == currentYear:
			monthKey := file.Timestamp.Format("2006-01")
			monthly[monthKey] = append(monthly[monthKey], file)
		default:
			yearKey := file.Timestamp.Format("2006")
			yearly[yearKey] = append(yearly[yearKey], file)
		}
	}

	markGoogleDriveBackupsToKeep(daily, retention.MaxPerDay, keep)
	markGoogleDriveBackupsToKeep(monthly, retention.MaxPerMonth, keep)
	markGoogleDriveBackupsToKeep(yearly, retention.MaxPerYear, keep)

	var toDelete []googleDriveBackupFile
	for _, file := range files {
		if !keep[file.ID] {
			toDelete = append(toDelete, file)
		}
	}
	return toDelete
}

func markGoogleDriveBackupsToKeep(groups map[string][]googleDriveBackupFile, max int, keep map[string]bool) {
	for _, group := range groups {
		if max <= 0 {
			for _, file := range group {
				keep[file.ID] = true
			}
			continue
		}
		for i, file := range group {
			if i < max {
				keep[file.ID] = true
			}
		}
	}
}

// SendFile implements StorageProvider interface
func (p *GoogleDriveProvider) SendFile(filePath string) error {
	p.log.Info("Starting file upload to Google Drive",
		"file", filePath,
		"folder_id", p.config.FolderID)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	driveFile := &drive.File{
		Name:    filepath.Base(filePath),
		Parents: []string{p.config.FolderID},
	}

	_, err = p.service.Files.Create(driveFile).
		SupportsAllDrives(true).
		Media(file).
		ProgressUpdater(func(current, total int64) {
			p.log.Info("Upload progress",
				"file", filePath,
				"current", current,
				"total", total,
				"percentage", float64(current)/float64(total)*100)
		}).
		Do()

	if err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}

	p.log.Info("File uploaded successfully to Google Drive",
		"file", filePath,
		"folder_id", p.config.FolderID)

	return nil
}

func (p *GoogleDriveProvider) CleanupRemoteBackups(backup config.BackupConfig) error {
	if !backup.RemoteRetention.Enabled {
		return nil
	}

	query := googleDriveBackupListQuery(p.config.FolderID, backup.Name)
	call := p.service.Files.List().
		Q(query).
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
		PageSize(1000).
		Fields("nextPageToken, files(id, name, mimeType)")

	var files []googleDriveBackupFile
	for {
		result, err := call.Do()
		if err != nil {
			return fmt.Errorf("failed to list Google Drive files for retention: %v", err)
		}

		for _, driveFile := range result.Files {
			if driveFile.MimeType == googleDriveFolderMimeType {
				continue
			}
			backupFile, ok := parseGoogleDriveBackupFile(driveFile.Id, driveFile.Name, backup.Name)
			if ok {
				files = append(files, backupFile)
			}
		}

		if result.NextPageToken == "" {
			break
		}
		call.PageToken(result.NextPageToken)
	}

	toDelete := selectGoogleDriveBackupsToDelete(files, backup.RemoteRetention, time.Now())
	for _, file := range toDelete {
		if err := p.service.Files.Delete(file.ID).SupportsAllDrives(true).Do(); err != nil {
			return fmt.Errorf("failed to delete Google Drive file %s (%s): %v", file.Name, file.ID, err)
		}
	}

	p.log.Info("Google Drive remote retention completed",
		"backup", backup.Name,
		"folder_id", p.config.FolderID,
		"matched", len(files),
		"deleted", len(toDelete),
	)
	return nil
}

// GetName implements StorageProvider interface
func (p *GoogleDriveProvider) GetName() string {
	return "google_drive"
}
