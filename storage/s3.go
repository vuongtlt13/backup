package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"backupdb/config"
	"backupdb/logger"
)

// S3Provider implements StorageProvider for AWS S3
type S3Provider struct {
	config config.StorageConfig
	client *s3.Client
	log    *logger.Logger
}

type s3BackupObject struct {
	Key       string
	Timestamp time.Time
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
	client := newS3Client(awsCfg, cfg)

	if !cfg.SkipBucketValidation {
		_, err = client.HeadBucket(context.Background(), &s3.HeadBucketInput{
			Bucket: aws.String(cfg.Bucket),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to validate S3 bucket access: %v", err)
		}
	}

	return &S3Provider{
		config: cfg,
		client: client,
		log:    logger.Get(),
	}, nil
}

func newS3Client(awsCfg aws.Config, cfg config.StorageConfig) *s3.Client {
	return s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		if cfg.Endpoint != "" {
			options.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		options.UsePathStyle = cfg.ForcePathStyle
	})
}

func effectiveS3ObjectKeyPrefix(backup config.BackupConfig, storageCfg config.StorageConfig) string {
	if strings.Trim(backup.ObjectKeyPrefix, "/") != "" {
		return backup.ObjectKeyPrefix
	}
	return storageCfg.ObjectKeyPrefix
}

func s3ObjectKey(filePath, prefix string) string {
	fileName := filepath.Base(filePath)
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return fileName
	}
	return prefix + "/" + fileName
}

func s3ListPrefix(prefix string) string {
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return ""
	}
	return prefix + "/"
}

func parseS3BackupObject(key, prefix, backupName string) (s3BackupObject, bool) {
	listPrefix := s3ListPrefix(prefix)
	if listPrefix != "" {
		if !strings.HasPrefix(key, listPrefix) {
			return s3BackupObject{}, false
		}
		key = strings.TrimPrefix(key, listPrefix)
	}
	if strings.Contains(key, "/") || strings.HasSuffix(key, "/") {
		return s3BackupObject{}, false
	}

	pattern := fmt.Sprintf(`^%s_(\d{14})(?:_\d{1,9})?\.tar\.gz$`, regexp.QuoteMeta(backupName))
	matches := regexp.MustCompile(pattern).FindStringSubmatch(key)
	if len(matches) != 2 {
		return s3BackupObject{}, false
	}

	timestamp, err := time.Parse("20060102150405", matches[1])
	if err != nil {
		return s3BackupObject{}, false
	}

	return s3BackupObject{Key: keyWithPrefix(key, listPrefix), Timestamp: timestamp}, true
}

func keyWithPrefix(key, prefix string) string {
	if prefix == "" {
		return key
	}
	return prefix + key
}

func selectS3BackupsToDelete(objects []s3BackupObject, retention config.RemoteRetentionConfig, now time.Time) []s3BackupObject {
	if len(objects) == 0 {
		return nil
	}

	sort.Slice(objects, func(i, j int) bool {
		return objects[i].Timestamp.After(objects[j].Timestamp)
	})

	keep := make(map[string]bool)
	currentYear, currentMonth, _ := now.Date()
	daily := make(map[string][]s3BackupObject)
	monthly := make(map[string][]s3BackupObject)
	yearly := make(map[string][]s3BackupObject)

	for _, object := range objects {
		year, month, _ := object.Timestamp.Date()
		switch {
		case year == currentYear && month == currentMonth:
			daily[object.Timestamp.Format("2006-01-02")] = append(daily[object.Timestamp.Format("2006-01-02")], object)
		case year == currentYear:
			monthly[object.Timestamp.Format("2006-01")] = append(monthly[object.Timestamp.Format("2006-01")], object)
		default:
			yearly[object.Timestamp.Format("2006")] = append(yearly[object.Timestamp.Format("2006")], object)
		}
	}

	markS3BackupsToKeep(daily, retention.MaxPerDay, keep)
	markS3BackupsToKeep(monthly, retention.MaxPerMonth, keep)
	markS3BackupsToKeep(yearly, retention.MaxPerYear, keep)

	var toDelete []s3BackupObject
	for _, object := range objects {
		if !keep[object.Key] {
			toDelete = append(toDelete, object)
		}
	}
	return toDelete
}

func markS3BackupsToKeep(groups map[string][]s3BackupObject, max int, keep map[string]bool) {
	for _, group := range groups {
		if max <= 0 {
			for _, object := range group {
				keep[object.Key] = true
			}
			continue
		}
		for i, object := range group {
			if i < max {
				keep[object.Key] = true
			}
		}
	}
}

// SendFile implements StorageProvider interface
func (p *S3Provider) SendFile(filePath string) error {
	return p.sendFileWithPrefix(filePath, p.config.ObjectKeyPrefix)
}

func (p *S3Provider) SendBackupFile(filePath string, backup config.BackupConfig) error {
	return p.sendFileWithPrefix(filePath, effectiveS3ObjectKeyPrefix(backup, p.config))
}

func (p *S3Provider) sendFileWithPrefix(filePath, prefix string) error {
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
		Key:           aws.String(s3ObjectKey(filePath, prefix)),
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

func (p *S3Provider) CleanupRemoteBackups(backup config.BackupConfig) error {
	if !backup.RemoteRetention.Enabled {
		return nil
	}

	prefix := effectiveS3ObjectKeyPrefix(backup, p.config)
	paginator := s3.NewListObjectsV2Paginator(p.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(p.config.Bucket),
		Prefix: aws.String(s3ListPrefix(prefix)),
	})

	var objects []s3BackupObject
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return fmt.Errorf("failed to list S3 objects for retention: %v", err)
		}

		for _, object := range page.Contents {
			if object.Key == nil {
				continue
			}
			backupObject, ok := parseS3BackupObject(*object.Key, prefix, backup.Name)
			if ok {
				objects = append(objects, backupObject)
			}
		}
	}

	toDelete := selectS3BackupsToDelete(objects, backup.RemoteRetention, time.Now())
	if len(toDelete) == 0 {
		return nil
	}

	for start := 0; start < len(toDelete); start += 1000 {
		end := start + 1000
		if end > len(toDelete) {
			end = len(toDelete)
		}
		var identifiers []types.ObjectIdentifier
		for _, object := range toDelete[start:end] {
			identifiers = append(identifiers, types.ObjectIdentifier{Key: aws.String(object.Key)})
		}

		output, err := p.client.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
			Bucket: aws.String(p.config.Bucket),
			Delete: &types.Delete{Objects: identifiers},
		})
		if err != nil {
			return fmt.Errorf("failed to delete S3 objects for retention: %v", err)
		}
		if len(output.Errors) > 0 {
			return fmt.Errorf("failed to delete %d S3 objects for retention", len(output.Errors))
		}
	}

	p.log.Info("S3 remote retention completed",
		"backup", backup.Name,
		"bucket", p.config.Bucket,
		"matched", len(objects),
		"deleted", len(toDelete),
	)
	return nil
}

// GetName implements StorageProvider interface
func (p *S3Provider) GetName() string {
	return "s3"
}
