package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Backups []BackupConfig `yaml:"backups"`
	Storage map[string]StorageConfig `yaml:"storage"` // Map of storage configurations
}

type BackupConfig struct {
	Name       string   `yaml:"name"`
	SourcePath string   `yaml:"source_path"`
	Storages   []string `yaml:"storages"` // List of storage names to use
}

type StorageConfig struct {
	Kind            string `yaml:"kind"` // Type of storage: "s3", "google_drive", "rsync"
	Enabled         bool   `yaml:"enabled"`
	// S3 specific fields
	Bucket          string `yaml:"bucket,omitempty"`
	Region          string `yaml:"region,omitempty"`
	AccessKey       string `yaml:"access_key,omitempty"`
	SecretKey       string `yaml:"secret_key,omitempty"`
	Path            string `yaml:"path,omitempty"`
	// Google Drive specific fields
	CredentialsFile string `yaml:"credentials_file,omitempty"`
	FolderID        string `yaml:"folder_id,omitempty"`
	// Rsync specific fields
	TargetServer    string `yaml:"target_server,omitempty"`
	TargetPath      string `yaml:"target_path,omitempty"`
	User            string `yaml:"user,omitempty"`
	Port            int    `yaml:"port,omitempty"`
}

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