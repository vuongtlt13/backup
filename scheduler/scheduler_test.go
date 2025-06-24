package scheduler

import (
	"backupdb/backup"
	"backupdb/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSchedulerService(t *testing.T) {
	cfg := &config.Config{}
	s := NewSchedulerService(cfg)
	assert.NotNil(t, s)
}

func TestSchedulerService_Start_Stop(t *testing.T) {
	cfg := &config.Config{
		Backups: []config.BackupConfig{
			{
				Name:       "test-backup",
				SourcePath: "test_data",
				Scheduler: struct {
					Enabled    bool   `yaml:"enabled"`
					CronExpr   string `yaml:"cron_expr"`
					MaxBackups int    `yaml:"max_backups"`
				}{
					Enabled:    true,
					CronExpr:   "* * * * *",
					MaxBackups: 1,
				},
			},
		},
	}
	s := NewSchedulerService(cfg)
	backupService := backup.NewBackupService(cfg)
	// Start và Stop không panic
	s.Start(backupService)
	s.Stop()
}

func TestSchedulerService_DisabledOrNoCron(t *testing.T) {
	cfg := &config.Config{
		Backups: []config.BackupConfig{
			{
				Name:       "no-cron",
				SourcePath: "test_data",
				Scheduler: struct {
					Enabled    bool   `yaml:"enabled"`
					CronExpr   string `yaml:"cron_expr"`
					MaxBackups int    `yaml:"max_backups"`
				}{
					Enabled:    false,
					CronExpr:   "",
					MaxBackups: 1,
				},
			},
		},
	}
	s := NewSchedulerService(cfg)
	backupService := backup.NewBackupService(cfg)
	s.Start(backupService)
	s.Stop()
}
