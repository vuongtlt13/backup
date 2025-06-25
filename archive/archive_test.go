package archive

import (
	"os"
	"path/filepath"
	"testing"

	"backupdb/config"

	"github.com/stretchr/testify/assert"
)

func TestCreateBackupArchive_Success(t *testing.T) {
	dir := "test_archive_data"
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "a.txt")
	os.WriteFile(file, []byte("hello"), 0644)

	backup := config.BackupConfig{
		Name:       "test-archive",
		SourcePath: dir,
	}
	archiveFile := "test-archive.tar.gz"
	defer os.Remove(archiveFile)

	service := NewArchiveService()
	err := service.CreateBackupArchive(backup, archiveFile)
	assert.NoError(t, err)
	info, err := os.Stat(archiveFile)
	assert.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestCreateBackupArchive_EmptyFolder(t *testing.T) {
	dir := "test_archive_empty"
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	backup := config.BackupConfig{
		Name:       "test-empty",
		SourcePath: dir,
	}
	archiveFile := "test-empty.tar.gz"
	defer os.Remove(archiveFile)

	service := NewArchiveService()
	err := service.CreateBackupArchive(backup, archiveFile)
	assert.NoError(t, err)
	info, err := os.Stat(archiveFile)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, info.Size(), int64(0))
}

func TestCreateBackupArchive_SourceNotExist(t *testing.T) {
	backup := config.BackupConfig{
		Name:       "not-exist",
		SourcePath: "not_exist_dir",
	}
	archiveFile := "not-exist.tar.gz"
	defer os.Remove(archiveFile)

	service := NewArchiveService()
	err := service.CreateBackupArchive(backup, archiveFile)
	assert.Error(t, err)
}

func TestCreateBackupArchive_CannotWriteArchive(t *testing.T) {
	dir := "test_archive_nowrite"
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "a.txt")
	os.WriteFile(file, []byte("hello"), 0644)

	backup := config.BackupConfig{
		Name:       "nowrite",
		SourcePath: dir,
	}
	// Path that cannot be written
	archiveFile := "/root/nowrite.tar.gz"

	service := NewArchiveService()
	err := service.CreateBackupArchive(backup, archiveFile)
	assert.Error(t, err)
}
