package storage

import (
	"fmt"
	"os/exec"
	"strings"

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
func (p *RsyncProvider) SendFile(backupDir string) error {
	p.log.Info("Sending file via rsync",
		"file", backupDir,
		"server", p.config.Server,
		"path", p.config.Path,
	)

	// Construct rsync command
	p.log.Info("Rsync", "rsync with host: %s", fmt.Sprintf("%s@%s:%s", p.config.Username, p.config.Server, p.config.Path))
	//rsync -avzr -e "ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p 22" --delete --progress ./backups/plus500_db roo@194.233.71.140:/root/backups/
	cmd := exec.Command("rsync",
		"-avzr",
		"-e",
		fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p %d", p.config.Port),
		"--delete",
		"--progress",
		backupDir,
		fmt.Sprintf("%s@%s:%s", p.config.Username, p.config.Server, p.config.Path),
	)
	p.log.Info("Rsync", "Running command:")
	p.log.Info("Rsync", strings.Join(cmd.Args, " "))

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		p.log.Error("Rsync", string(output))
		p.log.Error("Failed to send file via rsync",
			"file", backupDir,
			"server", p.config.Server,
			"error", err,
			"output", string(output),
		)
		return fmt.Errorf("failed to send file via rsync: %v", err)
	}

	p.log.Info("File sent successfully via rsync",
		"file", backupDir,
		"server", p.config.Server,
		"output", string(output),
	)
	return nil
}

// GetName implements StorageProvider interface
func (p *RsyncProvider) GetName() string {
	return "rsync"
}
