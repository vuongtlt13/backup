package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"backupdb/config"
	"backupdb/storage"
	"go.uber.org/zap"
)

type BackupService struct {
	config  *config.Config
	storage *storage.StorageService
	logger  *zap.Logger
}

func NewBackupService(cfg *config.Config) *BackupService {
	return &BackupService{
		config:  cfg,
		storage: storage.NewStorageService(cfg),
		logger:  zap.L(),
	}
}

func (s *BackupService) CreateBackup() error {
	for _, backup := range s.config.Backups {
		s.logger.Info("Starting backup process",
			zap.String("backup_name", backup.Name),
			zap.String("source_path", backup.SourcePath),
		)

		// Create backup directory
		backupDir := filepath.Join("backups", backup.Name)
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			s.logger.Error("Failed to create backup directory",
				zap.String("backup_name", backup.Name),
				zap.String("directory", backupDir),
				zap.Error(err),
			)
			return fmt.Errorf("failed to create backup directory: %v", err)
		}

		// Create timestamp for backup file
		timestamp := time.Now().Format("20060102150405")
		backupFile := filepath.Join(backupDir, fmt.Sprintf("%s.tar.gz", timestamp))

		s.logger.Info("Creating backup file",
			zap.String("backup_name", backup.Name),
			zap.String("file", backupFile),
		)

		// Create backup
		if err := s.backupFolder(backup, backupDir); err != nil {
			s.logger.Error("Failed to create backup",
				zap.String("backup_name", backup.Name),
				zap.String("file", backupFile),
				zap.Error(err),
			)
			return fmt.Errorf("failed to create backup: %v", err)
		}

		s.logger.Info("Backup created successfully",
			zap.String("backup_name", backup.Name),
			zap.String("file", backupFile),
		)
	}
	return nil
}

func (s *BackupService) backupFolder(backup config.BackupConfig, backupDir string) error {
	s.logger.Info("Starting folder backup",
		zap.String("backup_name", backup.Name),
		zap.String("source_path", backup.SourcePath),
		zap.String("backup_dir", backupDir),
	)

	// Create timestamp for backup file
	timestamp := time.Now().Format("20060102150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s.tar.gz", timestamp))

	// Create backup file
	file, err := os.Create(backupFile)
	if err != nil {
		s.logger.Error("Failed to create backup file",
			zap.String("backup_name", backup.Name),
			zap.String("file", backupFile),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create backup file: %v", err)
	}
	defer file.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk through the source directory
	err = filepath.Walk(backup.SourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.logger.Error("Error accessing path",
				zap.String("path", path),
				zap.Error(err),
			)
			return err
		}

		// Create header
		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			s.logger.Error("Failed to create tar header",
				zap.String("path", path),
				zap.Error(err),
			)
			return err
		}

		// Update header name to be relative to source path
		relPath, err := filepath.Rel(backup.SourcePath, path)
		if err != nil {
			s.logger.Error("Failed to get relative path",
				zap.String("path", path),
				zap.String("source_path", backup.SourcePath),
				zap.Error(err),
			)
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			s.logger.Error("Failed to write tar header",
				zap.String("path", path),
				zap.Error(err),
			)
			return err
		}

		// If it's a file, write its contents
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				s.logger.Error("Failed to open file",
					zap.String("path", path),
					zap.Error(err),
				)
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				s.logger.Error("Failed to write file to tar",
					zap.String("path", path),
					zap.Error(err),
				)
				return err
			}
		}

		return nil
	})

	if err != nil {
		s.logger.Error("Failed to create backup archive",
			zap.String("backup_name", backup.Name),
			zap.String("source_path", backup.SourcePath),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create backup archive: %v", err)
	}

	s.logger.Info("Folder backup completed successfully",
		zap.String("backup_name", backup.Name),
		zap.String("file", backupFile),
	)
	return nil
}

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath)
	})
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
} 