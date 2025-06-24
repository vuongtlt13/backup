package backup

import (
	"backupdb/archive"
	"backupdb/config"
	"backupdb/logger"
	"fmt"
	"os"
)

// FolderBackup implements BackupTask for folder backup
// Handles backup of a local folder by archiving it
type FolderBackup struct {
	archiveService *archive.ArchiveService
}

// Run executes the folder backup logic
func (t *FolderBackup) Run(backup config.BackupConfig, backupDir, backupFile string, log *logger.Logger) error {
	if _, err := os.Stat(backup.SourcePath); err != nil {
		return fmt.Errorf("failed to access source directory: %v", err)
	}
	if err := t.archiveService.CreateBackupArchive(backup, backupFile); err != nil {
		os.Remove(backupFile)
		return fmt.Errorf("failed to create backup for %s: %s: %v", backup.Name, backupFile, err)
	}
	return nil
}

// Kind returns the type of backup
func (t *FolderBackup) Kind() string { return "folder" }
