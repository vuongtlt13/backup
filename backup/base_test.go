package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"backupdb/config"

	"github.com/stretchr/testify/assert"
)

func TestNewBackupService(t *testing.T) {
	cfg := &config.Config{}
	service := NewBackupService(cfg)
	assert.NotNil(t, service)
}

func TestCreateBackup(t *testing.T) {
	// Create test directory structure
	testDir := "test_data"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		Backups: []config.BackupConfig{
			{
				Name:       "test-backup",
				SourcePath: testDir,
				Storage:    []string{}, // No storage for basic backup test
				Scheduler: struct {
					Enabled    bool   `yaml:"enabled"`
					CronExpr   string `yaml:"cron_expr"`
					MaxBackups int    `yaml:"max_backups"`
				}{
					Enabled:    true,
					CronExpr:   "*/5 * * * *",
					MaxBackups: 3,
				},
				Ignore: struct {
					Files   []string `yaml:"files"`
					Folders []string `yaml:"folders"`
				}{
					Files:   []string{"*.tmp"},
					Folders: []string{"temp"},
				},
			},
		},
		Storage: map[string]config.StorageConfig{
			"rsync": {
				Enabled:  true,
				Kind:     "rsync",
				Server:   "localhost",
				Username: "",
				Path:     "./backups/rsync",
				Port:     22,
			},
		},
	}

	// Create service
	service := NewBackupService(cfg)
	assert.NotNil(t, service)

	// Create backup directory
	backupDir := filepath.Join("backups", "test-backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("failed to create backup directory: %v", err)
	}
	defer os.RemoveAll("backups")

	// Test case 1: Successful backup without storage
	t.Run("Successful backup without storage", func(t *testing.T) {
		err := service.CreateBackup(cfg.Backups[0])
		assert.NoError(t, err)

		// Verify backup file exists
		entries, err := os.ReadDir(backupDir)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(entries))

		// Verify backup file name format and size
		backupFile := filepath.Join(backupDir, entries[0].Name())
		assert.Regexp(t, fmt.Sprintf("^%s_\\d{14}(_\\d+)?\\.tar\\.gz$", cfg.Backups[0].Name), entries[0].Name(), "Backup filename should match the expected format")
		info, err := os.Stat(backupFile)
		assert.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	})

	// Test case 2: Failed storage but successful backup
	t.Run("Failed storage but successful backup", func(t *testing.T) {
		backupWithStorage := cfg.Backups[0]
		backupWithStorage.Storage = []string{"rsync"}
		err := service.CreateBackup(backupWithStorage)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send backup to storage")

		// Verify backup file was cleaned up after storage failure
		entries, err := os.ReadDir(backupDir)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(entries)) // Should still have only the previous successful backup
	})

	// Test case 3: Backup with non-existent source
	t.Run("Non-existent source", func(t *testing.T) {
		invalidBackup := cfg.Backups[0]
		invalidBackup.SourcePath = "non_existent_dir"
		err := service.CreateBackup(invalidBackup)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to access source directory")

		// Verify no new backup file was created
		entries, err := os.ReadDir(backupDir)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(entries)) // Should still have only the previous successful backup
	})

	// Test case 4: Backup with read-only source
	t.Run("Read-only source", func(t *testing.T) {
		// Create a read-only directory
		readOnlyDir := filepath.Join(testDir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0755); err != nil {
			t.Skip("Skipping read-only test: cannot create test directory")
		}

		// Create a file in the read-only directory first
		testFile := filepath.Join(readOnlyDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			os.RemoveAll(readOnlyDir)
			t.Skip("Skipping read-only test: cannot create test file")
		}

		// Try to make directory completely inaccessible
		if err := os.Chmod(readOnlyDir, 0); err != nil {
			os.RemoveAll(readOnlyDir)
			t.Skip("Skipping read-only test: cannot set directory permissions")
		}
		defer func() {
			// Restore permissions for cleanup
			os.Chmod(readOnlyDir, 0755)
			os.RemoveAll(readOnlyDir)
		}()

		// Create a backup config with the read-only directory
		readOnlyBackup := cfg.Backups[0]
		readOnlyBackup.SourcePath = readOnlyDir

		// Attempt to create backup
		err := service.CreateBackup(readOnlyBackup)
		if err == nil {
			// If backup succeeds, try to verify if we can actually read the directory
			if _, err := os.ReadDir(readOnlyDir); err == nil {
				t.Skip("Skipping read-only test: directory is still accessible")
			}
		}
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "permission denied")
		}

		// Verify no new backup file was created
		entries, err := os.ReadDir(backupDir)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(entries)) // Should still have only the previous successful backup
	})

	// Test case 5: Backup with max backups limit
	t.Run("Max backups limit", func(t *testing.T) {
		// Reset backup directory
		os.RemoveAll(backupDir)
		os.MkdirAll(backupDir, 0755)

		// Create multiple backups
		for i := 0; i < 5; i++ {
			err := service.CreateBackup(cfg.Backups[0])
			assert.NoError(t, err)
			// Add a small delay to ensure different timestamps
			time.Sleep(100 * time.Millisecond)
		}

		// Verify only max_backups files exist
		entries, err := os.ReadDir(backupDir)
		assert.NoError(t, err)
		assert.Equal(t, cfg.Backups[0].Scheduler.MaxBackups, len(entries))

		// Get filenames and verify they are sorted by timestamp (newest first)
		var filenames []string
		for _, entry := range entries {
			filenames = append(filenames, entry.Name())
			// Verify filename format
			assert.Regexp(t, fmt.Sprintf("^%s_\\d{14}(_\\d+)?\\.tar\\.gz$", cfg.Backups[0].Name), entry.Name(), "Backup filename should match the expected format")
		}
		sort.Strings(filenames)
		sort.Sort(sort.Reverse(sort.StringSlice(filenames)))

		// Verify files are sorted by timestamp (newest first)
		for i := 0; i < len(filenames)-1; i++ {
			assert.True(t, filenames[i] > filenames[i+1], "Backup files should be sorted by timestamp (newest first)")
		}
	})
}

func TestBackupFolder(t *testing.T) {
	// Create test directory structure
	testDir := "test_data_backup"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		Backups: []config.BackupConfig{
			{
				Name:       "test-backup",
				SourcePath: testDir,
				Storage:    []string{"s3"},
				Ignore: struct {
					Files   []string `yaml:"files"`
					Folders []string `yaml:"folders"`
				}{
					Files:   []string{"*.tmp"},
					Folders: []string{"temp"},
				},
			},
		},
		Storage: map[string]config.StorageConfig{
			"s3": {
				Enabled:         true,
				Kind:            "s3",
				Bucket:          "test-bucket",
				Region:          "us-west-2",
				AccessKeyID:     "test-access-key",
				SecretAccessKey: "test-secret-key",
			},
		},
	}

	// Create service
	service := NewBackupService(cfg)
	assert.NotNil(t, service)

	// Create backup directory
	backupDir := filepath.Join("backups", "test-backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("failed to create backup directory: %v", err)
	}
	defer os.RemoveAll("backups")

	// Create a temporary file that should be ignored
	tmpFile := filepath.Join(testDir, "test.tmp")
	if err := os.WriteFile(tmpFile, []byte("temp content"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Create a directory that should be ignored
	tempDir := filepath.Join(testDir, "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}

	// Test backing up folder
	timestamp := time.Now().Format("20060102150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s_%s.tar.gz", cfg.Backups[0].Name, timestamp))
	err := service.archiveService.CreateBackupArchive(cfg.Backups[0], backupFile)
	assert.NoError(t, err)

	// Verify backup file exists
	_, err = os.Stat(backupFile)
	assert.NoError(t, err)
	// Verify backup file name format
	assert.Regexp(t, fmt.Sprintf("^%s_\\d{14}(_\\d+)?\\.tar\\.gz$", cfg.Backups[0].Name), filepath.Base(backupFile), "Backup filename should match the expected format")
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

func TestShouldIgnoreFile(t *testing.T) {
	testDir := "test_data_ignore"
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	tempFile := filepath.Join(testDir, "test.tmp")
	tempDir := filepath.Join(testDir, "temp")

	os.WriteFile(testFile, []byte("test content"), 0644)
	os.WriteFile(tempFile, []byte("temp content"), 0644)
	os.MkdirAll(tempDir, 0755)

	cfg := &config.Config{
		Backups: []config.BackupConfig{
			{
				Name:       "test-backup",
				SourcePath: testDir,
				Ignore: struct {
					Files   []string `yaml:"files"`
					Folders []string `yaml:"folders"`
				}{
					Files:   []string{"*.tmp"},
					Folders: []string{"temp"},
				},
			},
		},
	}
	service := NewBackupService(cfg)

	// Test cases
	assert.True(t, service.shouldIgnoreFile(tempFile, cfg.Backups[0]))
	assert.False(t, service.shouldIgnoreFile(testFile, cfg.Backups[0]))
	assert.True(t, service.shouldIgnoreFile(tempDir, cfg.Backups[0]))
	assert.False(t, service.shouldIgnoreFile(testDir, cfg.Backups[0]))
}
