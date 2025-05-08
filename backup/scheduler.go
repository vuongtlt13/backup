package backup

import (
	"strings"
	"sync"

	"github.com/robfig/cron/v3"

	"backupdb/config"
	"backupdb/logger"
)

type SchedulerService struct {
	config *config.Config
	log    *logger.Logger
	cron   *cron.Cron
	jobs   map[string]cron.EntryID
	mu     sync.Mutex
}

func NewSchedulerService(cfg *config.Config) *SchedulerService {
	return &SchedulerService{
		config: cfg,
		log:    logger.Get(),
		jobs:   make(map[string]cron.EntryID),
	}
}

// getCronInstance returns a cron.Cron instance appropriate for the cron expression
func getCronInstance(expr string) *cron.Cron {
	fields := strings.Fields(expr)
	if len(fields) == 6 {
		return cron.New(cron.WithSeconds())
	}
	return cron.New()
}

// Start starts the scheduler service
func (s *SchedulerService) Start(backupService *BackupService) {
	s.log.Info("Scheduler", "Starting scheduler service")

	// Schedule each backup
	for _, backup := range s.config.Backups {
		if !backup.Scheduler.Enabled {
			s.log.Info("Scheduler", "Scheduler disabled for backup: %s", backup.Name)
			continue
		}

		if backup.Scheduler.CronExpr == "" {
			s.log.Error("Scheduler", "No cron expression provided for backup: %s", backup.Name)
			continue
		}

		// Use the correct cron instance for this backup
		cronInstance := getCronInstance(backup.Scheduler.CronExpr)

		// Create a copy of the backup config for the closure
		backupCfg := backup
		jobID, err := cronInstance.AddFunc(backup.Scheduler.CronExpr, func() {
			s.log.Info("Scheduler", "Running scheduled backup: %s (cron: %s)", backupCfg.Name, backupCfg.Scheduler.CronExpr)

			if err := backupService.CreateBackup(backupCfg); err != nil {
				s.log.Error("Scheduler", "Failed to run scheduled backup: %s: %v", backupCfg.Name, err)
			} else {
				s.log.Info("Scheduler", "Backup completed successfully: %s", backupCfg.Name)
			}
		})

		if err != nil {
			s.log.Error("Scheduler", "Failed to schedule backup: %s (cron: %s): %v", backup.Name, backup.Scheduler.CronExpr, err)
			continue
		}

		s.mu.Lock()
		s.jobs[backup.Name] = jobID
		s.mu.Unlock()

		s.log.Info("Scheduler", "Backup scheduled successfully: %s (cron: %s)", backup.Name, backup.Scheduler.CronExpr)

		// Start the cron scheduler for this backup
		cronInstance.Start()
	}
}

// Stop stops the scheduler service
func (s *SchedulerService) Stop() {
	s.log.Info("Scheduler", "Stopping scheduler service")
	if s.cron != nil {
		s.cron.Stop()
	}
}
