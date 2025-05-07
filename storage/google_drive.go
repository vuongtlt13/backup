package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"backupdb/config"
	"backupdb/logger"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	driveChunkSize = 8 * 1024 * 1024 // 8MB chunks
)

// GoogleDriveProvider implements StorageProvider for Google Drive
type GoogleDriveProvider struct {
	service *drive.Service
	config  config.StorageConfig
	logger  *logger.Logger
}

// NewGoogleDriveProvider creates a new Google Drive storage provider
func NewGoogleDriveProvider(cfg config.StorageConfig) (*GoogleDriveProvider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("google drive provider is disabled")
	}

	ctx := context.Background()
	credentials, err := os.ReadFile(cfg.ServiceAccountFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account file: %v", err)
	}

	config, err := google.JWTConfigFromJSON(credentials, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT config: %v", err)
	}

	service, err := drive.NewService(ctx, option.WithHTTPClient(config.Client(ctx)))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %v", err)
	}

	return &GoogleDriveProvider{
		service: service,
		config:  cfg,
		logger:  logger.Get(),
	}, nil
}

// SendFile implements StorageProvider interface
func (p *GoogleDriveProvider) SendFile(filePath string) error {
	p.logger.Info("Starting file upload to Google Drive",
		"file", filePath,
		"folder_id", p.config.FolderID)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	driveFile := &drive.File{
		Name:    filepath.Base(filePath),
		Parents: []string{p.config.FolderID},
	}

	_, err = p.service.Files.Create(driveFile).
		Media(file, drive.ChunkSize(driveChunkSize)).
		ProgressUpdater(func(current, total int64) {
			p.logger.Info("Upload progress",
				"file", filePath,
				"current", current,
				"total", total,
				"percentage", float64(current)/float64(total)*100)
		}).
		Do()

	if err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}

	p.logger.Info("File uploaded successfully to Google Drive",
		"file", filePath,
		"folder_id", p.config.FolderID)

	return nil
}

// GetName implements StorageProvider interface
func (p *GoogleDriveProvider) GetName() string {
	return "google_drive"
}
