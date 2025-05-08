package storage

import (
	"fmt"
	"os"

	"backupdb/config"
	"backupdb/logger"
)

// StorageProvider defines the interface for all storage implementations
type StorageProvider interface {
	// SendFile sends a file to the storage destination
	SendFile(filePath string) error
	// GetName returns the name of the storage provider
	GetName() string
}

// StorageService manages multiple storage providers
type StorageService struct {
	providers map[string]StorageProvider
	log       *logger.Logger
}

// NewStorageService creates a new storage service with configured providers
func NewStorageService(cfg *config.Config) *StorageService {
	service := &StorageService{
		providers: make(map[string]StorageProvider),
		log:       logger.Get(),
	}

	// Initialize providers
	for name, storageCfg := range cfg.Storage {
		if !storageCfg.Enabled {
			service.log.Info("Storage", "[Storage] => Storage | %s (disabled)", name)
			continue
		}

		var provider StorageProvider
		var err error

		switch storageCfg.Kind {
		case "s3":
			service.log.Info("Storage", "[Storage] => Storage | s3")
			provider, err = NewS3Provider(storageCfg)
		case "rsync":
			service.log.Info("Storage", "[Storage] => Storage | rsync")
			provider, err = NewRsyncProvider(storageCfg)
		case "google_drive":
			service.log.Info("Storage", "[Storage] => Storage | google_drive")
			provider, err = NewGoogleDriveProvider(storageCfg)
		default:
			service.log.Error("Storage", "Unknown storage provider kind: %s", storageCfg.Kind)
			continue
		}

		if err != nil {
			service.log.Error("Storage", "Failed to initialize storage provider %s: %v", name, err)
			continue
		}

		if provider == nil {
			service.log.Error("Storage", "Failed to initialize storage provider %s (nil)", name)
			continue
		}

		service.providers[name] = provider
		service.log.Info("Storage", "[Storage] provider initialized: %s", name)
	}

	return service
}

// SendToStorage sends a backup file to all specified storage providers
func (s *StorageService) SendToStorage(filePath string, storageNames []string, backupName string) error {
	s.log.Info("Storage", "[%s] Sending file to storage: %s", backupName, filePath)

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", filePath)
	}

	// Track if any storage provider succeeded
	anySuccess := false
	var lastError error

	// Send to each specified storage provider
	for _, name := range storageNames {
		provider, err := s.GetProvider(name)
		if err != nil {
			s.log.Error("Storage", "[%s] Failed to get storage provider: %s", backupName, name)
			lastError = err
			continue
		}

		s.log.Info("Storage", "[%s] -> Sending file to provider: %s", backupName, name)

		if err := provider.SendFile(filePath); err != nil {
			s.log.Error("Storage", "[%s] Failed to send file to provider %s: %v", backupName, name, err)
			lastError = err
			continue
		}

		s.log.Info("Storage", "[%s] File sent successfully to provider: %s", backupName, name)
		anySuccess = true
	}

	if !anySuccess {
		return fmt.Errorf("failed to send file to any storage provider: %v", lastError)
	}

	s.log.Info("Storage", "[%s] File sent successfully to at least one provider: %s", backupName, filePath)
	return nil
}

// GetProvider returns a specific storage provider by name
func (s *StorageService) GetProvider(name string) (StorageProvider, error) {
	provider, exists := s.providers[name]
	if !exists {
		s.log.Error("Storage", "Storage provider not found: %s", name)
		return nil, fmt.Errorf("storage provider %s not found", name)
	}
	return provider, nil
}
