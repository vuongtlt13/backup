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

type BackupFileSender interface {
	SendBackupFile(filePath string, backup config.BackupConfig) error
}

type RemoteRetentionProvider interface {
	CleanupRemoteBackups(backup config.BackupConfig) error
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
func (s *StorageService) SendToStorage(filePath string, backup config.BackupConfig) error {
	s.log.Info("Storage", "[%s] Sending file to storage: %s", backup.Name, filePath)

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", filePath)
	}

	// Track if any storage provider succeeded
	anySuccess := false
	var lastError error

	// Send to each specified storage provider
	for _, name := range backup.Storage {
		provider, err := s.GetProvider(name)
		if err != nil {
			s.log.Error("Storage", "[%s] Failed to get storage provider: %s", backup.Name, name)
			lastError = err
			continue
		}

		s.log.Info("Storage", "[%s] -> Sending file to provider: %s", backup.Name, name)

		if sender, ok := provider.(BackupFileSender); ok {
			err = sender.SendBackupFile(filePath, backup)
		} else {
			err = provider.SendFile(filePath)
		}
		if err != nil {
			s.log.Error("Storage", "[%s] Failed to send file to provider %s: %v", backup.Name, name, err)
			lastError = err
			continue
		}

		s.log.Info("Storage", "[%s] File sent successfully to provider: %s", backup.Name, name)
		anySuccess = true
	}

	if !anySuccess {
		return fmt.Errorf("failed to send file to any storage provider: %v", lastError)
	}

	s.log.Info("Storage", "[%s] File sent successfully to at least one provider: %s", backup.Name, filePath)
	return nil
}

func (s *StorageService) CleanupRemoteRetention(backup config.BackupConfig) error {
	if !backup.RemoteRetention.Enabled {
		return nil
	}

	var lastError error
	for _, name := range backup.Storage {
		provider, err := s.GetProvider(name)
		if err != nil {
			lastError = err
			continue
		}

		retentionProvider, ok := provider.(RemoteRetentionProvider)
		if !ok {
			s.log.Info("Storage", "[%s] Provider %s does not support remote retention", backup.Name, name)
			continue
		}

		if err := retentionProvider.CleanupRemoteBackups(backup); err != nil {
			s.log.Error("Storage", "[%s] Failed to clean up remote backups for provider %s: %v", backup.Name, name, err)
			lastError = err
		}
	}

	return lastError
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
