package storage

import (
	"testing"

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
