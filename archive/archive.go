package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"backupdb/config"
	"backupdb/logger"
)

type ArchiveService struct {
	log *logger.Logger
}

func NewArchiveService() *ArchiveService {
	return &ArchiveService{
		log: logger.Get(),
	}
}

func (s *ArchiveService) CreateBackupArchive(backup config.BackupConfig, backupFile string) error {
	s.log.Info("Archive", "[%s] Starting folder backup for %s (source: %s, backup_dir: %s)",
		backup.Name, backup.Name, backup.SourcePath, filepath.Dir(backupFile))

	// Create tar.gz archive
	archive, err := os.Create(backupFile)
	if err != nil {
		s.log.Error("Archive", "[%s] Failed to create archive file: %v", backup.Name, err)
		return fmt.Errorf("failed to create archive file: %v", err)
	}
	defer archive.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(archive)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk through the source directory
	err = filepath.Walk(backup.SourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.log.Error("Archive", "[%s] Error accessing path: %s: %v", backup.Name, path, err)
			return fmt.Errorf("failed to access path %s: %v", path, err)
		}

		// Skip the root directory
		if path == backup.SourcePath {
			return nil
		}

		// Create header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			s.log.Error("Archive", "[%s] Failed to create tar header for %s: %v", backup.Name, path, err)
			return fmt.Errorf("failed to create tar header for %s: %v", path, err)
		}

		// Update header name to be relative to source path
		relPath, err := filepath.Rel(backup.SourcePath, path)
		if err != nil {
			s.log.Error("Archive", "[%s] Failed to get relative path for %s: %v", backup.Name, path, err)
			return fmt.Errorf("failed to get relative path for %s: %v", path, err)
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			s.log.Error("Archive", "[%s] Failed to write tar header for %s: %v", backup.Name, path, err)
			return fmt.Errorf("failed to write tar header for %s: %v", path, err)
		}

		// If it's a regular file, write its contents
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				s.log.Error("Archive", "[%s] Failed to open file for %s: %v", backup.Name, path, err)
				return fmt.Errorf("failed to open file for %s: %v", path, err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				s.log.Error("Archive", "[%s] Failed to write file contents for %s: %v", backup.Name, path, err)
				return fmt.Errorf("failed to write file contents for %s: %v", path, err)
			}
		}

		return nil
	})

	if err != nil {
		s.log.Error("Archive", "[%s] Failed to create backup archive for %s (source: %s): %v",
			backup.Name, backup.Name, backup.SourcePath, err)
		return fmt.Errorf("failed to create backup archive: %v", err)
	}

	s.log.Info("Archive", "[%s] Backup archive created successfully: %s", backup.Name, backupFile)
	return nil
}
