package backup

import (
	"backupdb/archive"
	"backupdb/config"
	"backupdb/logger"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMySQLBackup_ConfigValidation(t *testing.T) {
	backup := config.BackupConfig{
		Name: "mysql-test",
		SSH:  nil,
		DB:   nil,
	}
	task := &MySQLBackup{archiveService: archive.NewArchiveService()}
	err := task.Run(backup, "", "", logger.Get())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing DB config")
}

func TestMySQLBackup_DumpLocalSingleDB(t *testing.T) {
	backupDir := "backups/mysql-local"
	os.MkdirAll(backupDir, 0755)
	defer os.RemoveAll("backups")
	backupFile := filepath.Join(backupDir, "mysql-local.tar.gz")
	cfg := config.BackupConfig{
		Name: "mysql-local",
		DB: &config.DBConfig{
			Name:          "testdb",
			User:          "root",
			Password:      "wrongpass", // intentionally wrong to trigger error
			MysqldumpPath: "/home/vuongtlt13/mysqldump",
			DumpOptions:   []string{"--no-data"},
		},
	}
	task := &MySQLBackup{archiveService: archive.NewArchiveService()}
	err := task.Run(cfg, backupDir, backupFile, logger.Get())
	// Should succeed in creating archive even if dump fails
	assert.NoError(t, err)
	// Check that archive was created
	assert.FileExists(t, backupFile)
}

func TestMySQLBackup_DumpMultipleDBs_Exclude(t *testing.T) {
	backupDir := "backups/mysql-multi"
	os.MkdirAll(backupDir, 0755)
	defer os.RemoveAll("backups")
	backupFile := filepath.Join(backupDir, "mysql-multi.tar.gz")
	cfg := config.BackupConfig{
		Name: "mysql-multi",
		DB: &config.DBConfig{
			Databases:        []string{"db1", "db2", "db3"},
			ExcludeDatabases: []string{"db2"},
			User:             "root",
			Password:         "wrongpass", // intentionally wrong
			MysqldumpPath:    "/home/vuongtlt13/mysqldump",
			DumpOptions:      []string{"--no-data"},
		},
	}
	task := &MySQLBackup{archiveService: archive.NewArchiveService()}
	err := task.Run(cfg, backupDir, backupFile, logger.Get())
	// Should succeed in creating archive even if dumps fail
	assert.NoError(t, err)
	// Check that archive was created
	assert.FileExists(t, backupFile)
}

func TestMySQLBackup_CleanupMaxBackups(t *testing.T) {
	backupDir := "backups/mysql-cleanup"
	os.MkdirAll(backupDir, 0755)
	defer os.RemoveAll("backups")
	cfg := &config.Config{
		Backups: []config.BackupConfig{
			{
				Name: "mysql-cleanup",
				DB: &config.DBConfig{
					Name:          "testdb",
					User:          "root",
					Password:      "wrongpass",
					MysqldumpPath: "/home/vuongtlt13/mysqldump",
					DumpOptions:   []string{"--no-data"},
				},
				Scheduler: struct {
					Enabled    bool   `yaml:"enabled"`
					CronExpr   string `yaml:"cron_expr"`
					MaxBackups int    `yaml:"max_backups"`
				}{
					Enabled:    true,
					CronExpr:   "* * * * *",
					MaxBackups: 2,
				},
			},
		},
	}
	service := NewBackupService(cfg)
	for i := 0; i < 4; i++ {
		_ = service.CreateBackup(cfg.Backups[0])
	}
	entries, _ := os.ReadDir(backupDir)
	assert.LessOrEqual(t, len(entries), 2)
}
