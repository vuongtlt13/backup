package backup

import (
	"os"
	"path/filepath"
	"testing"

	"backupdb/config"
	"backupdb/logger"

	"github.com/stretchr/testify/assert"
)

func TestNewBackupService(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Backups: []config.BackupConfig{
			{
				Name:       "test_backup",
				SourcePath: "test_source",
				Storages:   []string{"s3_test", "rsync_test"},
			},
		},
	}

	// Create backup service
	service := NewBackupService(cfg)
	assert.NotNil(t, service, "Backup service should not be nil")
	assert.NotNil(t, service.config, "Config should not be nil")
	assert.NotNil(t, service.storage, "Storage service should not be nil")
}

func TestCreateBackup(t *testing.T) {
	// Create test directories
	testSource := "test_source"
	testFile := "test_file.txt"
	testContent := []byte("test content")

	err := os.MkdirAll(testSource, 0755)
	assert.NoError(t, err, "Failed to create test source directory")
	defer os.RemoveAll(testSource)

	err = os.WriteFile(filepath.Join(testSource, testFile), testContent, 0644)
	assert.NoError(t, err, "Failed to create test file")

	// Create test configuration
	cfg := &config.Config{
		Backups: []config.BackupConfig{
			{
				Name:       "test_backup",
				SourcePath: testSource,
				Storages:   []string{"s3_test"},
			},
		},
	}

	// Create backup service
	service := NewBackupService(cfg)

	// Create backup
	err = service.CreateBackup()
	assert.NoError(t, err, "Failed to create backup")

	// Verify backup file exists
	backupDir := filepath.Join("backups", "test_backup")
	backupFiles, err := filepath.Glob(filepath.Join(backupDir, "*.tar.gz"))
	assert.NoError(t, err, "Failed to find backup files")
	assert.NotEmpty(t, backupFiles, "Backup files should exist")

	// Cleanup
	os.RemoveAll("backups")
}

func TestBackupFolder(t *testing.T) {
	// Create test directories
	testSource := "test_source"
	testFile := "test_file.txt"
	testContent := []byte("test content")

	err := os.MkdirAll(testSource, 0755)
	assert.NoError(t, err, "Failed to create test source directory")
	defer os.RemoveAll(testSource)

	err = os.WriteFile(filepath.Join(testSource, testFile), testContent, 0644)
	assert.NoError(t, err, "Failed to create test file")

	// Create test configuration
	backupCfg := config.BackupConfig{
		Name:       "test_backup",
		SourcePath: testSource,
		Storages:   []string{"s3_test"},
	}

	// Create backup service
	service := NewBackupService(&config.Config{})

	// Create backup directory
	backupDir := filepath.Join("backups", backupCfg.Name)
	err = os.MkdirAll(backupDir, 0755)
	assert.NoError(t, err, "Failed to create backup directory")
	defer os.RemoveAll("backups")

	// Test backup folder
	err = service.backupFolder(backupCfg, backupDir)
	assert.NoError(t, err, "Failed to backup folder")

	// Verify backup file exists
	backupFiles, err := filepath.Glob(filepath.Join(backupDir, "*.tar.gz"))
	assert.NoError(t, err, "Failed to find backup files")
	assert.NotEmpty(t, backupFiles, "Backup files should exist")
}

func TestCopyDirectory(t *testing.T) {
	// Create test directories
	srcDir := "test_source"
	dstDir := "test_destination"
	testFile := "test_file.txt"
	testContent := []byte("test content")

	err := os.MkdirAll(srcDir, 0755)
	assert.NoError(t, err, "Failed to create source directory")
	defer os.RemoveAll(srcDir)

	err = os.WriteFile(filepath.Join(srcDir, testFile), testContent, 0644)
	assert.NoError(t, err, "Failed to create test file")

	// Test copy directory
	err = copyDirectory(srcDir, dstDir)
	assert.NoError(t, err, "Failed to copy directory")
	defer os.RemoveAll(dstDir)

	// Verify copied file exists
	copiedFile := filepath.Join(dstDir, testFile)
	content, err := os.ReadFile(copiedFile)
	assert.NoError(t, err, "Failed to read copied file")
	assert.Equal(t, testContent, content, "Copied file content should match")
}

func TestCopyFile(t *testing.T) {
	// Create test file
	srcFile := "test_source.txt"
	dstFile := "test_destination.txt"
	testContent := []byte("test content")

	err := os.WriteFile(srcFile, testContent, 0644)
	assert.NoError(t, err, "Failed to create source file")
	defer os.Remove(srcFile)
	defer os.Remove(dstFile)

	// Test copy file
	err = copyFile(srcFile, dstFile)
	assert.NoError(t, err, "Failed to copy file")

	// Verify copied file exists
	content, err := os.ReadFile(dstFile)
	assert.NoError(t, err, "Failed to read copied file")
	assert.Equal(t, testContent, content, "Copied file content should match")
} 