package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Backups []BackupConfig           `yaml:"backups"`
	Storage map[string]StorageConfig `yaml:"storage"`
}

// BackupConfig represents a single backup configuration
type BackupConfig struct {
	Name       string   `yaml:"name"`
	SourcePath string   `yaml:"source_path"`
	Storage    []string `yaml:"storage"`

	// New fields for DB backup
	Type string     `yaml:"type"` // folder, mysql, postgres
	SSH  *SSHConfig `yaml:"ssh,omitempty"`
	DB   *DBConfig  `yaml:"db,omitempty"`

	// Scheduler configuration
	Scheduler struct {
		Enabled    bool   `yaml:"enabled"`
		CronExpr   string `yaml:"cron_expr"`   // e.g. "0 2 * * *" for 2 AM daily
		MaxBackups int    `yaml:"max_backups"` // Maximum number of backups to keep
	} `yaml:"scheduler"`
	// Ignore patterns for files and folders
	Ignore struct {
		Files   []string `yaml:"files"`   // e.g. ["*.log", "*.tmp", "temp.txt"]
		Folders []string `yaml:"folders"` // e.g. ["node_modules", ".git", "temp"]
	} `yaml:"ignore"`
}

// SSHConfig holds SSH connection info
type SSHConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	User    string `yaml:"user"`
	KeyFile string `yaml:"key_file"`
}

// DBConfig holds database info for dump
type DBConfig struct {
	Name             string   `yaml:"name"`
	Databases        []string `yaml:"databases"`
	ExcludeDatabases []string `yaml:"exclude_databases"`
	User             string   `yaml:"user"`
	Password         string   `yaml:"password"`
	DumpOptions      []string `yaml:"dump_options"`
	MySQLPath        string   `yaml:"mysql_path"`     // Path to mysql binary
	MysqldumpPath    string   `yaml:"mysqldump_path"` // Path to mysqldump binary
	PSQLPath         string   `yaml:"psql_path"`      // Path to psql binary (for Postgres)
	PGDumpPath       string   `yaml:"pg_dump_path"`   // Path to pg_dump binary (for Postgres)
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Enabled bool   `yaml:"enabled"`
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
	Port     int    `yaml:"port"`
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
