package backup

import (
	"backupdb/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMySQLBackup_ConfigValidation(t *testing.T) {
	backup := config.BackupConfig{
		Name: "mysql-test",
		SSH:  nil,
		DB:   nil,
	}
	task := &MySQLBackup{}
	err := task.Run(backup, "", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing SSH or DB config")
}
