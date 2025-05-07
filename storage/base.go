package storage

import (
	"fmt"
	"os"

	"backupdb/config"
	"go.uber.org/zap"
)

// StorageProvider defines the interface for all storage implementations
type StorageProvider interface {
	// SendBackup sends a backup file to the storage destination
	SendBackup(filePath string) error
	// Name returns the name of the storage provider
	Name() string
}

// StorageService manages multiple storage providers
type StorageService struct {
	providers map[string]StorageProvider
	logger    *zap.Logger
}

// NewStorageService creates a new storage service with configured providers
func NewStorageService(cfg *config.Config) *StorageService {
	service := &StorageService{
		providers: make(map[string]StorageProvider),
		logger:    zap.L(),
	}

	// Initialize providers
	for name, storageCfg := range cfg.Storage {
		if !storageCfg.Enabled {
			service.logger.Info("Storage provider disabled",
				zap.String("provider", name),
				zap.String("kind", storageCfg.Kind),
			)
			continue
		}

		var provider StorageProvider
		switch storageCfg.Kind {
		case "s3":
			provider = NewS3Provider(name, storageCfg)
		case "rsync":
			provider = NewRsyncProvider(name, storageCfg)
		case "google_drive":
			provider = NewGoogleDriveProvider(name, storageCfg)
		default:
			service.logger.Error("Unknown storage provider kind",
				zap.String("provider", name),
				zap.String("kind", storageCfg.Kind),
			)
			continue
		}

		if provider == nil {
			service.logger.Error("Failed to initialize storage provider",
				zap.String("provider", name),
				zap.String("kind", storageCfg.Kind),
			)
			continue
		}

		service.providers[name] = provider
		service.logger.Info("Storage provider initialized",
			zap.String("provider", name),
			zap.String("kind", storageCfg.Kind),
		)
	}

	return service
}

// SendToStorage sends a backup file to all specified storage providers
func (s *StorageService) SendToStorage(filePath string, storageNames []string) error {
	s.logger.Info("Sending file to storage",
		zap.String("file", filePath),
		zap.Strings("storages", storageNames),
	)

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", filePath)
	}

	// Send to each specified storage provider
	for _, name := range storageNames {
		provider, err := s.GetProvider(name)
		if err != nil {
			s.logger.Error("Failed to get storage provider",
				zap.String("provider", name),
				zap.Error(err),
			)
			return err
		}

		s.logger.Info("Sending file to provider",
			zap.String("file", filePath),
			zap.String("provider", name),
		)

		if err := provider.SendBackup(filePath); err != nil {
			s.logger.Error("Failed to send file to provider",
				zap.String("file", filePath),
				zap.String("provider", name),
				zap.Error(err),
			)
			return fmt.Errorf("failed to send to %s: %v", name, err)
		}

		s.logger.Info("File sent successfully to provider",
			zap.String("file", filePath),
			zap.String("provider", name),
		)
	}

	s.logger.Info("File sent successfully to all providers",
		zap.String("file", filePath),
		zap.Strings("storages", storageNames),
	)
	return nil
}

// GetProvider returns a specific storage provider by name
func (s *StorageService) GetProvider(name string) (StorageProvider, error) {
	provider, exists := s.providers[name]
	if !exists {
		s.logger.Error("Storage provider not found",
			zap.String("provider", name),
		)
		return nil, fmt.Errorf("storage provider %s not found", name)
	}
	return provider, nil
}
