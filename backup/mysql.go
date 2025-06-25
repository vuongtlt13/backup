package backup

import (
	"backupdb/archive"
	"backupdb/config"
	"backupdb/logger"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// MySQLBackup implements BackupTask for MySQL database backup
// Handles backup of a MySQL database via SSH and mysqldump
type MySQLBackup struct {
	archiveService *archive.ArchiveService
}

// getFreePort finds a free TCP port for local tunnel
func getFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// startSSHTunnel starts an SSH tunnel and returns the cmd process and local port
func startSSHTunnel(sshCfg *config.SSHConfig, remoteHost string, remotePort int) (*exec.Cmd, int, error) {
	localPort, err := getFreePort()
	if err != nil {
		return nil, 0, err
	}
	args := []string{"-N", "-L", fmt.Sprintf("%d:%s:%d", localPort, remoteHost, remotePort), "-p", fmt.Sprintf("%d", sshCfg.Port)}
	if sshCfg.KeyFile != "" {
		args = append(args, "-i", sshCfg.KeyFile)
	}
	args = append(args, fmt.Sprintf("%s@%s", sshCfg.User, sshCfg.Host))
	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, 0, err
	}
	// Wait a bit for tunnel to be ready
	for i := 0; i < 10; i++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", localPort), 300*time.Millisecond)
		if err == nil {
			conn.Close()
			return cmd, localPort, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	cmd.Process.Kill()
	return nil, 0, fmt.Errorf("failed to establish SSH tunnel on port %d", localPort)
}

// Run executes the MySQL backup logic
func (t *MySQLBackup) Run(backup config.BackupConfig, backupDir, backupFile string, log *logger.Logger) error {
	if backup.DB == nil {
		return fmt.Errorf("missing DB config for database backup")
	}

	var tunnelCmd *exec.Cmd
	var localPort int
	var err error
	useTunnel := backup.SSH != nil
	if useTunnel {
		tunnelCmd, localPort, err = startSSHTunnel(backup.SSH, "127.0.0.1", 3306)
		if err != nil {
			return fmt.Errorf("failed to start SSH tunnel: %v", err)
		}
		defer func() {
			if tunnelCmd != nil && tunnelCmd.Process != nil {
				tunnelCmd.Process.Kill()
			}
		}()
	}

	// Determine which databases to backup
	var databases []string
	if len(backup.DB.Databases) > 0 {
		databases = backup.DB.Databases
	} else if backup.DB.Name == "__ALL__" {
		allDBs, err := t.getAllDatabases(backup, log, localPort, useTunnel)
		if err != nil {
			return fmt.Errorf("failed to get databases list: %v", err)
		}
		databases = allDBs
	} else if backup.DB.Name != "" {
		databases = []string{backup.DB.Name}
	} else {
		return fmt.Errorf("no database specified for backup")
	}

	if len(backup.DB.ExcludeDatabases) > 0 {
		databases = t.filterExcludedDatabases(databases, backup.DB.ExcludeDatabases)
	}
	if len(databases) == 0 {
		return fmt.Errorf("no databases to backup after filtering")
	}

	log.Info("Backup", "[%s] Backing up %d databases: %v", backup.Name, len(databases), databases)

	tempDir := filepath.Join(backupDir, "temp_dumps")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	for _, dbName := range databases {
		dumpFile := filepath.Join(tempDir, fmt.Sprintf("%s.sql", dbName))
		err := t.dumpDatabase(backup, dbName, dumpFile, log, localPort, useTunnel)
		if err != nil {
			log.Error("Backup", "[%s] Failed to dump database %s: %v", backup.Name, dbName, err)
			continue
		}
		log.Info("Backup", "[%s] Successfully dumped database: %s", backup.Name, dbName)
	}

	err = t.archiveService.CreateBackupArchive(config.BackupConfig{
		Name:       backup.Name,
		SourcePath: tempDir,
		Ignore:     backup.Ignore,
	}, backupFile)
	if err != nil {
		os.Remove(backupFile)
		return fmt.Errorf("failed to create archive for db backup: %v", err)
	}

	log.Info("Backup", "[%s] Successfully created backup archive with %d databases", backup.Name, len(databases))
	return nil
}

// getAllDatabases gets list of all databases from MySQL server
func (t *MySQLBackup) getAllDatabases(backup config.BackupConfig, log *logger.Logger, localPort int, useTunnel bool) ([]string, error) {
	bin := "mysql"
	if backup.DB.MySQLPath != "" {
		bin = backup.DB.MySQLPath
	}
	args := []string{"-u", backup.DB.User}
	if backup.DB.Password != "" {
		args = append(args, fmt.Sprintf("-p%s", backup.DB.Password))
	}
	args = append(args, "-e", "SHOW DATABASES;")
	if useTunnel {
		args = append([]string{"-h", "127.0.0.1", "-P", fmt.Sprintf("%d", localPort)}, args...)
	} else if backup.SSH != nil {
		sshArgs := []string{}
		if backup.SSH.KeyFile != "" {
			sshArgs = append(sshArgs, "-i", backup.SSH.KeyFile)
		}
		sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", backup.SSH.User, backup.SSH.Host), bin)
		sshArgs = append(sshArgs, args...)
		cmd := exec.Command("ssh", sshArgs...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			log.Error("Backup", "[%s] Failed to get databases list: %v, output: %s", backup.Name, err, out.String())
			return nil, fmt.Errorf("failed to get databases list: %v, output: %s", err, out.String())
		}
		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		var databases []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && line != "Database" {
				databases = append(databases, line)
			}
		}
		return databases, nil
	}
	cmd := exec.Command(bin, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		log.Error("Backup", "[%s] Failed to get databases list: %v, output: %s", backup.Name, err, out.String())
		return nil, fmt.Errorf("failed to get databases list: %v, output: %s", err, out.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	var databases []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != "Database" {
			databases = append(databases, line)
		}
	}
	return databases, nil
}

// filterExcludedDatabases removes excluded databases from the list
func (t *MySQLBackup) filterExcludedDatabases(databases, excluded []string) []string {
	excludedMap := make(map[string]bool)
	for _, db := range excluded {
		excludedMap[db] = true
	}

	var filtered []string
	for _, db := range databases {
		if !excludedMap[db] {
			filtered = append(filtered, db)
		}
	}
	return filtered
}

// dumpDatabase dumps a single database
func (t *MySQLBackup) dumpDatabase(backup config.BackupConfig, dbName, dumpFile string, log *logger.Logger, localPort int, useTunnel bool) error {
	bin := "mysqldump"
	if backup.DB.MysqldumpPath != "" {
		bin = backup.DB.MysqldumpPath
	}
	args := []string{"-u", backup.DB.User}
	if backup.DB.Password != "" {
		args = append(args, fmt.Sprintf("-p%s", backup.DB.Password))
	}
	args = append(args, backup.DB.DumpOptions...)
	args = append(args, dbName)
	if useTunnel {
		args = append([]string{"-h", "127.0.0.1", "-P", fmt.Sprintf("%d", localPort)}, args...)
	} else if backup.SSH != nil {
		sshArgs := []string{}
		if backup.SSH.KeyFile != "" {
			sshArgs = append(sshArgs, "-i", backup.SSH.KeyFile)
		}
		sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", backup.SSH.User, backup.SSH.Host), bin)
		sshArgs = append(sshArgs, args...)
		dumpCmd := exec.Command("ssh", sshArgs...)
		var out bytes.Buffer
		dumpCmd.Stdout = &out
		dumpCmd.Stderr = &out
		if err := dumpCmd.Run(); err != nil {
			log.Error("Backup", "[%s] DB dump failed for %s: %v, output: %s", backup.Name, dbName, err, out.String())
			return fmt.Errorf("failed to dump database %s: %v, output: %s", dbName, err, out.String())
		}
		if err := os.WriteFile(dumpFile, out.Bytes(), 0644); err != nil {
			return fmt.Errorf("failed to write dump file for %s: %v", dbName, err)
		}
		return nil
	}
	dumpCmd := exec.Command(bin, args...)
	var out bytes.Buffer
	dumpCmd.Stdout = &out
	dumpCmd.Stderr = &out
	if err := dumpCmd.Run(); err != nil {
		log.Error("Backup", "[%s] DB dump failed for %s: %v, output: %s", backup.Name, dbName, err, out.String())
		return fmt.Errorf("failed to dump database %s: %v, output: %s", dbName, err, out.String())
	}
	if err := os.WriteFile(dumpFile, out.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write dump file for %s: %v", dbName, err)
	}
	return nil
}

// Kind returns the type of backup
func (t *MySQLBackup) Kind() string { return "mysql" }
