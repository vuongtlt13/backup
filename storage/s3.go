package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"backupdb/config"
	"backupdb/logger"
)

// S3Provider implements StorageProvider for AWS S3
type S3Provider struct {
	config config.StorageConfig
	client *s3.Client
	log    *logger.Logger
}

// NewS3Provider creates a new S3 storage provider
func NewS3Provider(cfg config.StorageConfig) (*S3Provider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("s3 provider is disabled")
	}

	// Validate required fields
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("s3 region is required")
	}
	if cfg.AccessKeyID == "" {
		return nil, fmt.Errorf("s3 access key ID is required")
	}
	if cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("s3 secret access key is required")
	}

	// Create AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg)

	// Validate credentials by trying to list buckets
	_, err = client.ListBuckets(context.Background(), &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to validate S3 credentials: %v", err)
	}

	return &S3Provider{
		config: cfg,
		client: client,
		log:    logger.Get(),
	}, nil
}

// SendFile implements StorageProvider interface
func (p *S3Provider) SendFile(filePath string) error {
	p.log.Info("Sending file to S3",
		"file", filePath,
		"bucket", p.config.Bucket,
	)

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		p.log.Error("Failed to open file",
			"file", filePath,
			"error", err,
		)
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		p.log.Error("Failed to get file info",
			"file", filePath,
			"error", err,
		)
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Upload file to S3
	_, err = p.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:        aws.String(p.config.Bucket),
		Key:           aws.String(filepath.Base(filePath)),
		Body:          file,
		ContentLength: aws.Int64(fileInfo.Size()),
	})
	if err != nil {
		p.log.Error("Failed to upload file to S3",
			"file", filePath,
			"bucket", p.config.Bucket,
			"error", err,
		)
		return fmt.Errorf("failed to upload file to S3: %v", err)
	}

	p.log.Info("File sent successfully to S3",
		"file", filePath,
		"bucket", p.config.Bucket,
	)
	return nil
}

// GetName implements StorageProvider interface
func (p *S3Provider) GetName() string {
	return "s3"
}
