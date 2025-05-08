package main

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"backupdb/backup"
	"backupdb/config"
	"backupdb/logger"
	"backupdb/storage"
)

func main() {
	// Parse command line flags
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Initialize logger
	log := logger.Get()
	defer log.Sync()

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Error("Config", "Failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Create services
	backupService := backup.NewBackupService(cfg)
	storageService := storage.NewStorageService(cfg)
	schedulerService := backup.NewSchedulerService(cfg)

	// Run initial backups
	go func() {
		for _, backup := range cfg.Backups {
			if err := backupService.CreateBackup(backup); err != nil {
				log.Error("Backup", "Failed to create backup for %s: %v", backup.Name, err)
				continue
			}

			// Find the backup file
			backupDir := filepath.Join("backups", backup.Name)
			backupFiles, err := filepath.Glob(filepath.Join(backupDir, "*.tar.gz"))
			if err != nil {
				log.Error("Backup", "Failed to find backup files for %s: %v", backup.Name, err)
				continue
			}

			if len(backupFiles) == 0 {
				log.Error("Backup", "No backup files found for %s", backup.Name)
				continue
			}

			// Get the most recent backup file
			latestBackup := backupFiles[len(backupFiles)-1]

			// Send to storage if configured
			if len(backup.Storage) > 0 {
				if err := storageService.SendToStorage(latestBackup, backup.Storage, backup.Name); err != nil {
					log.Error("Backup", "[%s] Failed to send backup to storage: %v", backup.Name, err)
					continue
				}
			}

			log.Info("Backup", "Backup completed successfully for %s: %s", backup.Name, latestBackup)
		}

		// Start the scheduler after initial backups
		schedulerService.Start(backupService)
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Stop the scheduler gracefully
	schedulerService.Stop()
	log.Info("System", "Shutting down...")
}

func init() {
	// Create backups directory if it doesn't exist
	if err := os.MkdirAll("backups", 0755); err != nil {
		panic("Failed to create backups directory: " + err.Error())
	}
}
