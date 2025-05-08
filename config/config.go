package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Backups []BackupConfig            `yaml:"backups"`
	Storage map[string]StorageConfig  `yaml:"storage"`
}

// BackupConfig represents a single backup configuration
type BackupConfig struct {
	Name       string   `yaml:"name"`
	SourcePath string   `yaml:"source_path"`
	Storage    []string `yaml:"storage"`
	// Scheduler configuration
	Scheduler struct {
		Enabled    bool   `yaml:"enabled"`
		CronExpr   string `yaml:"cron_expr"`   // e.g. "0 2 * * *" for 2 AM daily
		MaxBackups int    `yaml:"max_backups"` // Maximum number of backups to keep
	} `yaml:"scheduler"`
	// Ignore patterns for files and folders
	Ignore struct {
		Files   []string `yaml:"files"`    // e.g. ["*.log", "*.tmp", "temp.txt"]
		Folders []string `yaml:"folders"`   // e.g. ["node_modules", ".git", "temp"]
	} `yaml:"ignore"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Enabled bool `yaml:"enabled"`
	Kind    string `yaml:"kind"` // s3, rsync, google_drive

	// S3 specific fields
	Bucket          string `yaml:"bucket"`
	Region          string `yaml:"region"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`

	// Google Drive specific fields
	CredentialsFile string `yaml:"credentials_file"`
	FolderID        string `yaml:"folder_id"`

	// Rsync specific fields
	Server   string `yaml:"server"`
	Username string `yaml:"username"`
	Path     string `yaml:"path"`
}

// LoadConfig loads the configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
} 