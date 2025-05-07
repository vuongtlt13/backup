package storage

import (
	"fmt"
	"os/exec"

	"backupdb/config"
	"backupdb/logger"
)

// RsyncProvider implements StorageProvider for Rsync
type RsyncProvider struct {
	config config.StorageConfig
	logger *logger.Logger
}

// NewRsyncProvider creates a new Rsync storage provider
func NewRsyncProvider(cfg config.StorageConfig) (*RsyncProvider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("rsync provider is disabled")
	}

	return &RsyncProvider{
		config: cfg,
		logger: logger.Get(),
	}, nil
}

// SendFile implements StorageProvider interface
func (p *RsyncProvider) SendFile(filePath string) error {
	p.logger.Info("Starting file transfer via rsync",
		"file", filePath,
		"target", p.config.TargetServer,
		"path", p.config.TargetPath)

	args := []string{
		"-avz",
		"--progress",
		filePath,
	}

	if p.config.Port != 0 {
		args = append(args, "-e", fmt.Sprintf("ssh -p %d", p.config.Port))
	}

	if p.config.User != "" {
		args = append(args, fmt.Sprintf("%s@%s:%s", p.config.User, p.config.TargetServer, p.config.TargetPath))
	} else {
		args = append(args, fmt.Sprintf("%s:%s", p.config.TargetServer, p.config.TargetPath))
	}

	cmd := exec.Command("rsync", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		p.logger.Error("Failed to transfer file via rsync",
			"file", filePath,
			"target", p.config.TargetServer,
			"error", err,
			"output", string(output))
		return fmt.Errorf("rsync failed: %v", err)
	}

	p.logger.Info("File transferred successfully via rsync",
		"file", filePath,
		"target", p.config.TargetServer,
		"path", p.config.TargetPath)

	return nil
}

// GetName implements StorageProvider interface
func (p *RsyncProvider) GetName() string {
	return "rsync"
} 