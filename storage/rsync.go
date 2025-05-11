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
	log    *logger.Logger
}

// NewRsyncProvider creates a new Rsync storage provider
func NewRsyncProvider(cfg config.StorageConfig) (*RsyncProvider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("rsync provider is disabled")
	}

	return &RsyncProvider{
		config: cfg,
		log:    logger.Get(),
	}, nil
}

// SendFile implements StorageProvider interface
func (p *RsyncProvider) SendFile(filePath string) error {
	p.log.Info("Sending file via rsync",
		"file", filePath,
		"server", p.config.Server,
		"path", p.config.Path,
	)

	// Construct rsync command
	p.log.Info("S3", "rsync with host: %s", fmt.Sprintf("%s@%s:%s", p.config.Username, p.config.Server, p.config.Path))
	cmd := exec.Command("rsync",
		"-avz",
		"--progress",
		filePath,
		fmt.Sprintf("%s@%s:%s", p.config.Username, p.config.Server, p.config.Path),
	)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		p.log.Error("S3", string(output))
		p.log.Error("Failed to send file via rsync",
			"file", filePath,
			"server", p.config.Server,
			"error", err,
			"output", string(output),
		)
		return fmt.Errorf("failed to send file via rsync: %v", err)
	}

	p.log.Info("File sent successfully via rsync",
		"file", filePath,
		"server", p.config.Server,
		"output", string(output),
	)
	return nil
}

// GetName implements StorageProvider interface
func (p *RsyncProvider) GetName() string {
	return "rsync"
}
