package backup

import (
	"backupdb/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostgresBackup_ConfigValidation(t *testing.T) {
	backup := config.BackupConfig{
		Name: "postgres-test",
		SSH:  nil,
		DB:   nil,
	}
	task := &PostgresBackup{}
	err := task.Run(backup, "", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing SSH or DB config")
}
