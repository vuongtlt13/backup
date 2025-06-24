package backup

import (
	"backupdb/archive"
	"backupdb/config"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFolderBackup_Run_Success(t *testing.T) {
	testDir := "test_data_folder"
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	backupDir := "backups/folder-test"
	os.MkdirAll(backupDir, 0755)
	defer os.RemoveAll("backups")

	backupFile := filepath.Join(backupDir, "folder-test.tar.gz")
	cfg := config.BackupConfig{
		Name:       "folder-test",
		SourcePath: testDir,
	}
	task := &FolderBackup{archiveService: archive.NewArchiveService()}
	err := task.Run(cfg, backupDir, backupFile, nil)
	assert.NoError(t, err)
	_, err = os.Stat(backupFile)
	assert.NoError(t, err)
}

func TestFolderBackup_Run_NonExistentSource(t *testing.T) {
	backupDir := "backups/folder-test-nonexistent"
	os.MkdirAll(backupDir, 0755)
	defer os.RemoveAll("backups")
	backupFile := filepath.Join(backupDir, "folder-test.tar.gz")
	cfg := config.BackupConfig{
		Name:       "folder-test-nonexistent",
		SourcePath: "non_existent_dir",
	}
	task := &FolderBackup{archiveService: archive.NewArchiveService()}
	err := task.Run(cfg, backupDir, backupFile, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to access source directory")
}

func TestFolderBackup_Run_ReadOnlySource(t *testing.T) {
	testDir := "test_data_folder_readonly"
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)
	testFile := filepath.Join(testDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	os.Chmod(testDir, 0)
	defer os.Chmod(testDir, 0755)
	backupDir := "backups/folder-test-readonly"
	os.MkdirAll(backupDir, 0755)
	defer os.RemoveAll("backups")
	backupFile := filepath.Join(backupDir, "folder-test.tar.gz")
	cfg := config.BackupConfig{
		Name:       "folder-test-readonly",
		SourcePath: testDir,
	}
	task := &FolderBackup{archiveService: archive.NewArchiveService()}
	err := task.Run(cfg, backupDir, backupFile, nil)
	assert.Error(t, err)
}

func TestFolderBackup_Run_IgnorePattern(t *testing.T) {
	testDir := "test_data_folder_ignore"
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	tmpFile := filepath.Join(testDir, "test.tmp")
	tmpDir := filepath.Join(testDir, "temp")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.WriteFile(tmpFile, []byte("temp content"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	backupDir := "backups/folder-test-ignore"
	os.MkdirAll(backupDir, 0755)
	defer os.RemoveAll("backups")
	backupFile := filepath.Join(backupDir, "folder-test.tar.gz")
	cfg := config.BackupConfig{
		Name:       "folder-test-ignore",
		SourcePath: testDir,
		Ignore: struct {
			Files   []string `yaml:"files"`
			Folders []string `yaml:"folders"`
		}{
			Files:   []string{"*.tmp"},
			Folders: []string{"temp"},
		},
	}
	task := &FolderBackup{archiveService: archive.NewArchiveService()}
	err := task.Run(cfg, backupDir, backupFile, nil)
	assert.NoError(t, err)
	// Optionally: check tarball content to ensure ignored files/folders are not included
}

func TestFolderBackup_Run_MultiLevel(t *testing.T) {
	testDir := "test_data_folder_multi"
	os.MkdirAll(filepath.Join(testDir, "subdir1", "subdir2"), 0755)
	defer os.RemoveAll(testDir)
	if err := os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("file1"), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "subdir1", "file2.txt"), []byte("file2"), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "subdir1", "subdir2", "file3.txt"), []byte("file3"), 0644); err != nil {
		t.Fatalf("failed to create file3: %v", err)
	}
	backupDir := "backups/folder-test-multi"
	os.MkdirAll(backupDir, 0755)
	defer os.RemoveAll("backups")
	backupFile := filepath.Join(backupDir, "folder-test.tar.gz")
	cfg := config.BackupConfig{
		Name:       "folder-test-multi",
		SourcePath: testDir,
	}
	task := &FolderBackup{archiveService: archive.NewArchiveService()}
	err := task.Run(cfg, backupDir, backupFile, nil)
	assert.NoError(t, err)
	_, err = os.Stat(backupFile)
	assert.NoError(t, err)
}
