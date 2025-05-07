package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"backupdb/backup"
	"backupdb/config"
	"backupdb/logger"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	// Initialize logger
	log := logger.Get()
	defer logger.Sync()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatal("Failed to load configuration",
			zap.String("config_path", *configPath),
			zap.Error(err),
		)
	}

	// Create backup service
	backupService := backup.NewBackupService(cfg)

	// Process each backup configuration
	for _, backupCfg := range cfg.Backups {
		log.Info("Processing backup configuration",
			zap.String("name", backupCfg.Name),
			zap.String("source_path", backupCfg.SourcePath),
		)

		// Create backup
		if err := backupService.CreateBackup(); err != nil {
			log.Error("Failed to create backup",
				zap.String("name", backupCfg.Name),
				zap.Error(err),
			)
			continue
		}

		// Find the backup file
		backupDir := filepath.Join("backups", backupCfg.Name)
		backupFiles, err := filepath.Glob(filepath.Join(backupDir, "*.tar.gz"))
		if err != nil {
			log.Error("Failed to find backup files",
				zap.String("name", backupCfg.Name),
				zap.Error(err),
			)
			continue
		}

		if len(backupFiles) == 0 {
			log.Error("No backup files found",
				zap.String("name", backupCfg.Name),
			)
			continue
		}

		// Get the most recent backup file
		latestBackup := backupFiles[len(backupFiles)-1]

		// Send to storage
		if err := backupService.SendToStorage(latestBackup, backupCfg.Storages); err != nil {
			log.Error("Failed to send backup to storage",
				zap.String("name", backupCfg.Name),
				zap.String("file", latestBackup),
				zap.Error(err),
			)
			continue
		}

		log.Info("Backup completed successfully",
			zap.String("name", backupCfg.Name),
			zap.String("file", latestBackup),
		)
	}
}

func init() {
	// Create backups directory if it doesn't exist
	if err := os.MkdirAll("backups", 0755); err != nil {
		panic("Failed to create backups directory: " + err.Error())
	}
} 