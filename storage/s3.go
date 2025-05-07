package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"backupdb/config"
	"backupdb/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
)

// S3Provider implements StorageProvider for AWS S3
type S3Provider struct {
	name   string
	config config.StorageConfig
	client *s3.Client
}

// NewS3Provider creates a new S3 storage provider
func NewS3Provider(name string, cfg config.StorageConfig) *S3Provider {
	log := logger.Get().With(
		zap.String("provider", name),
		zap.String("kind", "s3"),
	)

	// Create AWS config
	awsCfg, err := s3config.LoadDefaultConfig(context.Background(),
		s3config.WithRegion(cfg.Region),
		s3config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		log.Error("Failed to create AWS config",
			zap.Error(err),
		)
		return nil
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg)

	return &S3Provider{
		name:   name,
		config: cfg,
		client: client,
	}
}

// SendBackup implements StorageProvider interface
func (p *S3Provider) SendBackup(filePath string) error {
	log := logger.Get().With(
		zap.String("provider", p.name),
		zap.String("kind", "s3"),
	)

	log.Info("Starting S3 upload",
		zap.String("file", filePath),
		zap.String("bucket", p.config.Bucket),
		zap.String("path", p.config.Path),
	)

	// Open file
	fileContent, err := os.Open(filePath)
	if err != nil {
		log.Error("Failed to open file",
			zap.String("file", filePath),
			zap.Error(err),
		)
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer fileContent.Close()

	// Get file info
	fileInfo, err := fileContent.Stat()
	if err != nil {
		log.Error("Failed to get file info",
			zap.String("file", filePath),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Create S3 key
	key := filepath.Join(p.config.Path, filepath.Base(filePath))

	// Upload file
	_, err = p.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:        aws.String(p.config.Bucket),
		Key:           aws.String(key),
		Body:          fileContent,
		ContentLength: aws.Int64(fileInfo.Size()),
	})
	if err != nil {
		log.Error("Failed to upload file to S3",
			zap.String("file", filePath),
			zap.String("bucket", p.config.Bucket),
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("failed to upload to S3: %v", err)
	}

	log.Info("File uploaded successfully to S3",
		zap.String("file", filePath),
		zap.String("bucket", p.config.Bucket),
		zap.String("key", key),
	)
	return nil
}

// Name implements StorageProvider interface
func (p *S3Provider) Name() string {
	return p.name
}
