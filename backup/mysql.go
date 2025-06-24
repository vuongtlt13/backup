package backup

import (
	"backupdb/archive"
	"backupdb/config"
	"backupdb/logger"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// MySQLBackup implements BackupTask for MySQL database backup
// Handles backup of a MySQL database via SSH and mysqldump
type MySQLBackup struct {
	archiveService *archive.ArchiveService
}

// Run executes the MySQL backup logic
func (t *MySQLBackup) Run(backup config.BackupConfig, backupDir, backupFile string, log *logger.Logger) error {
	if backup.SSH == nil || backup.DB == nil {
		return fmt.Errorf("missing SSH or DB config for database backup")
	}
	dumpFile := filepath.Join(backupDir, fmt.Sprintf("%s.sql", backup.Name))
	args := []string{"-i", backup.SSH.KeyFile, fmt.Sprintf("%s@%s", backup.SSH.User, backup.SSH.Host), "mysqldump"}
	args = append(args, fmt.Sprintf("-u%s", backup.DB.User))
	if backup.DB.Password != "" {
		args = append(args, fmt.Sprintf("-p%s", backup.DB.Password))
	}
	args = append(args, backup.DB.DumpOptions...)
	args = append(args, backup.DB.Name)
	dumpCmd := exec.Command("ssh", args...)
	var out bytes.Buffer
	dumpCmd.Stdout = &out
	dumpCmd.Stderr = &out
	err := dumpCmd.Run()
	if err != nil {
		log.Error("Backup", "[%s] DB dump failed: %v, output: %s", backup.Name, err, out.String())
		return fmt.Errorf("failed to dump database: %v, output: %s", err, out.String())
	}
	if err := os.WriteFile(dumpFile, out.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write dump file: %v", err)
	}
	err = t.archiveService.CreateBackupArchive(config.BackupConfig{
		Name:       backup.Name,
		SourcePath: backupDir,
		Ignore:     backup.Ignore,
	}, backupFile)
	if err != nil {
		os.Remove(backupFile)
		return fmt.Errorf("failed to create archive for db backup: %v", err)
	}
	os.Remove(dumpFile)
	return nil
}

// Kind returns the type of backup
func (t *MySQLBackup) Kind() string { return "mysql" }
