package storage

import (
	"testing"
	"time"

	"backupdb/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

func TestNewS3Provider(t *testing.T) {
	cfg := config.StorageConfig{
		Enabled:         true,
		Kind:            "s3",
		Bucket:          "test-bucket",
		Region:          "us-west-2",
		AccessKeyID:     "invalid-key",
		SecretAccessKey: "invalid-secret",
	}

	provider, err := NewS3Provider(cfg)
	assert.Error(t, err)
	assert.Nil(t, provider)

	disabledCfg := config.StorageConfig{
		Enabled: false,
		Kind:    "s3",
	}

	provider, err = NewS3Provider(disabledCfg)
	assert.Error(t, err)
	assert.Nil(t, provider)

	invalidCfg := config.StorageConfig{
		Enabled: true,
		Kind:    "s3",
		Bucket:  "test-bucket",
		Region:  "us-west-2",
	}

	provider, err = NewS3Provider(invalidCfg)
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func TestNewS3ClientOptions(t *testing.T) {
	client := newS3Client(aws.Config{Region: "us-east-1"}, config.StorageConfig{
		Endpoint:       "http://localhost:9000",
		ForcePathStyle: true,
	})

	options := client.Options()
	assert.Equal(t, "http://localhost:9000", *options.BaseEndpoint)
	assert.True(t, options.UsePathStyle)
}

func TestS3ObjectKey(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		prefix   string
		expected string
	}{
		{
			name:     "no prefix",
			filePath: "/backups/mysql_data_20260508.tar.gz",
			expected: "mysql_data_20260508.tar.gz",
		},
		{
			name:     "simple prefix",
			filePath: "/backups/mysql_data_20260508.tar.gz",
			prefix:   "mysql",
			expected: "mysql/mysql_data_20260508.tar.gz",
		},
		{
			name:     "trimmed prefix",
			filePath: "/backups/mysql_data_20260508.tar.gz",
			prefix:   "/prod/mysql/",
			expected: "prod/mysql/mysql_data_20260508.tar.gz",
		},
		{
			name:     "nested prefix",
			filePath: "/backups/postgres_data_20260508.tar.gz",
			prefix:   "prod/postgres",
			expected: "prod/postgres/postgres_data_20260508.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, s3ObjectKey(tt.filePath, tt.prefix))
		})
	}
}

func TestEffectiveS3ObjectKeyPrefix(t *testing.T) {
	storageCfg := config.StorageConfig{ObjectKeyPrefix: "storage-prefix"}

	assert.Equal(t, "job-prefix", effectiveS3ObjectKeyPrefix(config.BackupConfig{ObjectKeyPrefix: "job-prefix"}, storageCfg))
	assert.Equal(t, "storage-prefix", effectiveS3ObjectKeyPrefix(config.BackupConfig{}, storageCfg))
	assert.Equal(t, "storage-prefix", effectiveS3ObjectKeyPrefix(config.BackupConfig{ObjectKeyPrefix: "/"}, storageCfg))
}

func TestS3ListPrefix(t *testing.T) {
	assert.Equal(t, "", s3ListPrefix(""))
	assert.Equal(t, "backups/", s3ListPrefix("backups"))
	assert.Equal(t, "prod/mysql/", s3ListPrefix("/prod/mysql/"))
}

func TestParseS3BackupObject(t *testing.T) {
	object, ok := parseS3BackupObject("mysql/mysql_data_20260508010203_123456789.tar.gz", "mysql", "mysql_data")
	assert.True(t, ok)
	assert.Equal(t, "mysql/mysql_data_20260508010203_123456789.tar.gz", object.Key)
	assert.Equal(t, time.Date(2026, 5, 8, 1, 2, 3, 0, time.UTC), object.Timestamp)

	object, ok = parseS3BackupObject("mysql/mysql_data_20260508010203.tar.gz", "mysql", "mysql_data")
	assert.True(t, ok)
	assert.Equal(t, "mysql/mysql_data_20260508010203.tar.gz", object.Key)

	_, ok = parseS3BackupObject("postgres/mysql_data_20260508010203_123456789.tar.gz", "mysql", "mysql_data")
	assert.False(t, ok)

	_, ok = parseS3BackupObject("mysql/mysql_data_extra_20260508010203_123456789.tar.gz", "mysql", "mysql_data")
	assert.False(t, ok)

	_, ok = parseS3BackupObject("mysql/postgres_data_20260508010203_123456789.tar.gz", "mysql", "mysql_data")
	assert.False(t, ok)

	_, ok = parseS3BackupObject("mysql/mysql_data_20260508010203_123456789.zip", "mysql", "mysql_data")
	assert.False(t, ok)

	_, ok = parseS3BackupObject("mysql/nested/mysql_data_20260508010203_123456789.tar.gz", "mysql", "mysql_data")
	assert.False(t, ok)
}

func TestSelectS3BackupsToDelete(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	objects := []s3BackupObject{
		{Key: "mysql/mysql_data_20260508030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 3, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260508020000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 2, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260508010000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 1, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260508000000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260507030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 7, 3, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260507020000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 7, 2, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260506030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 6, 3, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260402000000_000000001.tar.gz", Timestamp: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260401000000_000000001.tar.gz", Timestamp: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20250301000000_000000001.tar.gz", Timestamp: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20250201000000_000000001.tar.gz", Timestamp: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
	}

	toDelete := selectS3BackupsToDelete(objects, config.RemoteRetentionConfig{
		Enabled:     true,
		MaxPerDay:   3,
		MaxPerMonth: 1,
		MaxPerYear:  1,
	}, now)

	assert.ElementsMatch(t, []s3BackupObject{
		{Key: "mysql/mysql_data_20260508000000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260507020000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 7, 2, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260506030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 6, 3, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260401000000_000000001.tar.gz", Timestamp: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20250201000000_000000001.tar.gz", Timestamp: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
	}, toDelete)
}

func TestSelectS3BackupsToDeletePeriodTier(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	objects := []s3BackupObject{
		{Key: "mysql/mysql_data_20260508030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 3, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260508020000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 2, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260508010000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 1, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260508000000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260507030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 7, 3, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260506030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 6, 3, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260505030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 5, 3, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260504030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 4, 3, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260503030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 3, 3, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260402000000_000000001.tar.gz", Timestamp: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260401000000_000000001.tar.gz", Timestamp: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
	}

	toDelete := selectS3BackupsToDelete(objects, config.RemoteRetentionConfig{
		Enabled:      true,
		MaxPerDay:    3,
		PeriodDays:   3,
		MaxPerPeriod: 1,
		MaxPerMonth:  1,
		MaxPerYear:   1,
	}, now)

	assert.ElementsMatch(t, []s3BackupObject{
		{Key: "mysql/mysql_data_20260508000000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260506030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 6, 3, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260505030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 5, 3, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260503030000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 3, 3, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260401000000_000000001.tar.gz", Timestamp: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
	}, toDelete)
}

func TestSelectS3BackupsToDeleteZeroMaxKeepsTier(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	objects := []s3BackupObject{
		{Key: "mysql/mysql_data_20260508020000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 2, 0, 0, 0, time.UTC)},
		{Key: "mysql/mysql_data_20260508010000_000000001.tar.gz", Timestamp: time.Date(2026, 5, 8, 1, 0, 0, 0, time.UTC)},
	}

	toDelete := selectS3BackupsToDelete(objects, config.RemoteRetentionConfig{
		Enabled:     true,
		MaxPerDay:   0,
		MaxPerMonth: 1,
		MaxPerYear:  1,
	}, now)

	assert.Empty(t, toDelete)
}

func TestNewS3ProviderSkipsBucketValidation(t *testing.T) {
	cfg := config.StorageConfig{
		Enabled:              true,
		Kind:                 "s3",
		Bucket:               "test-bucket",
		Region:               "us-west-2",
		AccessKeyID:          "test-key",
		SecretAccessKey:      "test-secret",
		SkipBucketValidation: true,
	}

	provider, err := NewS3Provider(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestS3ProviderSendFile(t *testing.T) {
	cfg := config.StorageConfig{
		Enabled:         true,
		Kind:            "s3",
		Bucket:          "test-bucket",
		Region:          "us-west-2",
		AccessKeyID:     "invalid-key",
		SecretAccessKey: "invalid-secret",
	}

	provider, err := NewS3Provider(cfg)
	assert.Error(t, err)
	assert.Nil(t, provider)
}
