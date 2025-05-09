package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"backupdb/config"
	"backupdb/logger"
	"backupdb/storage"
)

type BackupService struct {
	config         *config.Config
	log            *logger.Logger
	archiveService *ArchiveService
	storageService *storage.StorageService
}

func NewBackupService(cfg *config.Config) *BackupService {
	return &BackupService{
		config:         cfg,
		log:            logger.Get(),
		archiveService: NewArchiveService(),
		storageService: storage.NewStorageService(cfg),
	}
}

// shouldIgnoreFile checks if a file should be ignored based on the ignore patterns
func (s *BackupService) shouldIgnoreFile(path string, backup config.BackupConfig) bool {
	// Get the relative path from the source path
	relPath, err := filepath.Rel(backup.SourcePath, path)
	if err != nil {
		s.log.Error("Backup", "Failed to get relative path: %s (source: %s): %v", path, backup.SourcePath, err)
		return false
	}

	// Check file patterns (both full path and filename)
	for _, pattern := range backup.Ignore.Files {
		// Check against full path
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
		// Check against just the filename
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}

	// Check folder patterns (both full path and path segments)
	for _, pattern := range backup.Ignore.Folders {
		// Check if the pattern matches any part of the path
		pathParts := strings.Split(relPath, string(filepath.Separator))
		for _, part := range pathParts {
			if matched, _ := filepath.Match(pattern, part); matched {
				return true
			}
		}
		// Also check the full path
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
	}

	return false
}

// cleanupOldBackups removes old backups if they exceed the maximum number allowed
func (s *BackupService) cleanupOldBackups(backup config.BackupConfig) error {
	if backup.Scheduler.MaxBackups <= 0 {
		return nil
	}

	backupDir := filepath.Join("backups", backup.Name)
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %v", err)
	}

	// Sort backups by timestamp in filename (newest first)
	type backupInfo struct {
		path      string
		timestamp string
	}
	var backups []backupInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".tar.gz") {
			continue
		}
		// Extract timestamp from filename (format: YYYYMMDDHHMMSS.NNNNNN.tar.gz)
		timestamp := strings.TrimSuffix(name, ".tar.gz")
		backups = append(backups, backupInfo{
			path:      filepath.Join(backupDir, name),
			timestamp: timestamp,
		})
	}

	// Sort by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].timestamp > backups[j].timestamp
	})

	// Remove old backups
	if len(backups) > backup.Scheduler.MaxBackups {
		for i := backup.Scheduler.MaxBackups; i < len(backups); i++ {
			s.log.Info("Backup", "[%s] Removing old backup: %s", backup.Name, backups[i].path)
			if err := os.Remove(backups[i].path); err != nil {
				s.log.Error("Backup", "[%s] Failed to remove old backup: %s: %v", backup.Name, backups[i].path, err)
			}
		}
	}

	return nil
}

// CreateBackup creates a backup of the specified backup configuration
func (s *BackupService) CreateBackup(backup config.BackupConfig) error {
	s.log.Info("Backup", "[%s] Starting backup process for %s (source: %s)", backup.Name, backup.Name, backup.SourcePath)

	// Check if source directory exists and is accessible
	if _, err := os.Stat(backup.SourcePath); err != nil {
		return fmt.Errorf("failed to access source directory: %v", err)
	}

	// Create backup directory if it doesn't exist
	backupDir := filepath.Join("backups", backup.Name)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Generate backup filename with timestamp and microseconds for uniqueness
	timestamp := time.Now().Format("20060102150405.000000")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s_%s.tar.gz", backup.Name, timestamp))
	s.log.Info("Backup", "[%s] Creating backup file for %s: %s", backup.Name, backup.Name, backupFile)

	// Create backup archive
	if err := s.archiveService.CreateBackupArchive(backup, backupFile); err != nil {
		// Clean up the backup file if it exists
		os.Remove(backupFile)
		return fmt.Errorf("failed to create backup for %s: %s: %v", backup.Name, backupFile, err)
	}

	// Send backup to storage
	if len(backup.Storage) > 0 {
		if err := s.storageService.SendToStorage(backupFile, backup.Storage, backup.Name); err != nil {
			// Clean up the backup file if storage fails
			os.Remove(backupFile)
			return fmt.Errorf("failed to send backup to storage: %v", err)
		}
	}

	// Clean up old backups
	if err := s.cleanupOldBackups(backup); err != nil {
		s.log.Error("Backup", "[%s] Failed to clean up old backups: %v", backup.Name, err)
	}

	s.log.Info("Backup", "[%s] Backup completed successfully: %s", backup.Name, backup.Name)
	return nil
}

func (s *BackupService) backupFolder(backup config.BackupConfig, backupDir string) error {
	s.log.Info("Archive", "Starting folder backup for %s (source: %s, backup_dir: %s)", backup.Name, backup.SourcePath, backupDir)

	// Create timestamp for backup file
	timestamp := time.Now().Format("20060102150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s_%s.tar.gz", backup.Name, timestamp))

	// Create backup file
	file, err := os.Create(backupFile)
	if err != nil {
		s.log.Error("Archive", "Failed to create backup file for %s: %s: %v", backup.Name, backupFile, err)
		return fmt.Errorf("failed to create backup file: %v", err)
	}
	defer file.Close()

	// Create gzip writer with best compression
	gzipWriter, err := gzip.NewWriterLevel(file, gzip.BestCompression)
	if err != nil {
		s.log.Error("Archive", "Failed to create gzip writer for %s: %s: %v", backup.Name, backupFile, err)
		return fmt.Errorf("failed to create gzip writer: %v", err)
	}
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk through the source directory
	err = filepath.Walk(backup.SourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.log.Error("Archive", "Error accessing path: %s: %v", path, err)
			return fmt.Errorf("failed to access path %s: %v", path, err)
		}

		// Skip ignored files and folders
		if s.shouldIgnoreFile(path, backup) {
			s.log.Info("Ignore", "Skipping ignored file/folder: %s", path)
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Create header
		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			s.log.Error("Archive", "Failed to create tar header for %s: %v", path, err)
			return fmt.Errorf("failed to create tar header for %s: %v", path, err)
		}

		// Update header name to be relative to source path
		relPath, err := filepath.Rel(backup.SourcePath, path)
		if err != nil {
			s.log.Error("Archive", "Failed to get relative path for %s (source: %s): %v", path, backup.SourcePath, err)
			return fmt.Errorf("failed to get relative path for %s: %v", path, err)
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			s.log.Error("Archive", "Failed to write tar header for %s: %v", path, err)
			return fmt.Errorf("failed to write tar header for %s: %v", path, err)
		}

		// If it's a file, write its contents
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				s.log.Error("Archive", "Failed to open file for %s: %v", path, err)
				return fmt.Errorf("failed to open file %s: %v", path, err)
			}
			defer file.Close()

			// Use a buffer to copy file contents
			buffer := make([]byte, 32*1024) // 32KB buffer
			if _, err := io.CopyBuffer(tarWriter, file, buffer); err != nil {
				s.log.Error("Archive", "Failed to write file to tar for %s: %v", path, err)
				return fmt.Errorf("failed to write file %s to tar: %v", path, err)
			}
		}

		return nil
	})

	if err != nil {
		s.log.Error("Archive", "Failed to create backup archive for %s (source: %s): %v", backup.Name, backup.SourcePath, err)
		return fmt.Errorf("failed to create backup archive: %v", err)
	}

	// Ensure all data is written
	if err := tarWriter.Close(); err != nil {
		s.log.Error("Archive", "Failed to close tar writer for %s: %v", backup.Name, err)
		return fmt.Errorf("failed to close tar writer: %v", err)
	}

	if err := gzipWriter.Close(); err != nil {
		s.log.Error("Archive", "Failed to close gzip writer for %s: %v", backup.Name, err)
		return fmt.Errorf("failed to close gzip writer: %v", err)
	}

	s.log.Info("Archive", "Folder backup completed successfully for %s: %s", backup.Name, backupFile)
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
