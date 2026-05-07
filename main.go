package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"backupdb/backup"
	"backupdb/config"
	"backupdb/logger"
	"backupdb/scheduler"
	"backupdb/storage"

	"golang.org/x/oauth2"
)

func main() {
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	googleDriveAuthInit := flag.String("gdrive-auth-init", "", "Initialize OAuth token for the named Google Drive storage")
	flag.Parse()

	log := logger.Get()
	defer log.Sync()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Error("Config", "Failed to load configuration: %v", err)
		os.Exit(1)
	}

	if *googleDriveAuthInit != "" {
		if err := initializeGoogleDriveOAuth(cfg, *googleDriveAuthInit); err != nil {
			log.Error("Google Drive OAuth", "%v", err)
			os.Exit(1)
		}
		return
	}

	backupService := backup.NewBackupService(cfg)
	schedulerService := scheduler.NewSchedulerService(cfg)

	go func() {
		for _, backup := range cfg.Backups {
			if err := backupService.CreateBackup(backup); err != nil {
				log.Error("Backup", "Failed to create backup for %s: %v", backup.Name, err)
				continue
			}

			log.Info("Backup", "Backup completed successfully for %s", backup.Name)
		}

		schedulerService.Start(backupService)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	schedulerService.Stop()
	log.Info("System", "Shutting down...")
}

func initializeGoogleDriveOAuth(cfg *config.Config, storageName string) error {
	storageCfg, exists := cfg.Storage[storageName]
	if !exists {
		return fmt.Errorf("storage %s not found", storageName)
	}
	if storageCfg.Kind != "google_drive" {
		return fmt.Errorf("storage %s is not a google_drive provider", storageName)
	}
	if storageCfg.AuthMode != "oauth_user" {
		return fmt.Errorf("storage %s must set auth_mode: oauth_user", storageName)
	}

	oauthConfig, err := storage.NewGoogleDriveOAuthConfig(storageCfg)
	if err != nil {
		return err
	}

	authURL := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Println("Open this URL in your browser:")
	fmt.Println(authURL)
	fmt.Println()
	fmt.Print("Paste the authorization code or full callback URL: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read authorization code: %v", err)
	}
	code, err := googleDriveOAuthCodeFromInput(input)
	if err != nil {
		return err
	}

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return fmt.Errorf("failed to exchange authorization code: %v", err)
	}
	if err := storage.SaveOAuthToken(storageCfg.TokenFile, token); err != nil {
		return err
	}

	fmt.Printf("Google Drive OAuth token saved to %s\n", storageCfg.TokenFile)
	return nil
}

func googleDriveOAuthCodeFromInput(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("authorization code or callback URL is required")
	}

	parsedURL, err := url.Parse(input)
	if err == nil && parsedURL.Query().Get("code") != "" {
		return parsedURL.Query().Get("code"), nil
	}

	return input, nil
}

func init() {
	if err := os.MkdirAll("backups", 0755); err != nil {
		panic("Failed to create backups directory: " + err.Error())
	}
}
