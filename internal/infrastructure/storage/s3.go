package storage

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/config"
)

type s3Storage struct {
	client       *minio.Client
	bucket       string
	originalDir  string
	processedDir string
}

func NewS3Storage(cfg *config.StorageConfig) (Storage, error) {
	if cfg.S3Endpoint == "" {
		return nil, fmt.Errorf("s3 endpoint is required")
	}
	if cfg.S3Bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}
	if cfg.S3AccessKey == "" || cfg.S3SecretKey == "" {
		return nil, fmt.Errorf("s3 access key and secret key are required")
	}

	if cfg.OriginalDir == "" {
		cfg.OriginalDir = "original"
	}
	if cfg.ProcessedDir == "" {
		cfg.ProcessedDir = "processed"
	}

	creds := credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, "")
	client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  creds,
		Secure: cfg.S3UseSSL,
		Region: cfg.S3Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize s3 client: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.S3Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check s3 bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.S3Bucket, minio.MakeBucketOptions{Region: cfg.S3Region}); err != nil {
			zlog.Logger.Warn().Err(err).Str("bucket", cfg.S3Bucket).Msg("unable to create bucket, ensure it exists and credentials are correct")
		} else {
			zlog.Logger.Info().Str("bucket", cfg.S3Bucket).Msg("created s3 bucket")
		}
	}

	return &s3Storage{
		client:       client,
		bucket:       cfg.S3Bucket,
		originalDir:  cfg.OriginalDir,
		processedDir: cfg.ProcessedDir,
	}, nil
}

func (s *s3Storage) SaveOriginal(ctx context.Context, filename string, reader io.Reader) (string, error) {
	return s.saveObject(ctx, s.originalDir, filename, reader)
}

func (s *s3Storage) SaveProcessed(ctx context.Context, filename string, reader io.Reader) (string, error) {
	return s.saveObject(ctx, s.processedDir, filename, reader)
}

func (s *s3Storage) saveObject(ctx context.Context, dir, filename string, reader io.Reader) (string, error) {
	if reader == nil {
		zlog.Logger.Error().Str("filename", filename).Msg("reader is nil")
		return "", fmt.Errorf("reader is nil")
	}

	objectName := path.Join(dir, filename)

	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, -1, minio.PutObjectOptions{})
	if err != nil {
		zlog.Logger.Error().Err(err).Str("object", objectName).Msg("failed to put object to s3")
		return "", fmt.Errorf("put object %s: %w", objectName, err)
	}

	zlog.Logger.Info().Str("path", objectName).Msg("object saved to s3")
	return objectName, nil
}

func (s *s3Storage) GetOriginal(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.getObject(ctx, path)
}

func (s *s3Storage) GetProcessed(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.getObject(ctx, path)
}

func (s *s3Storage) getObject(ctx context.Context, objectPath string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, objectPath, minio.GetObjectOptions{})
	if err != nil {
		zlog.Logger.Error().Err(err).Str("object", objectPath).Msg("failed to get object")
		return nil, fmt.Errorf("get object %s: %w", objectPath, err)
	}

	if _, err := obj.Stat(); err != nil {
		zlog.Logger.Error().Err(err).Str("object", objectPath).Msg("object not found or inaccessible")
		return nil, fmt.Errorf("%w: %s", ErrObjectNotFound, objectPath)
	}

	zlog.Logger.Info().Str("path", objectPath).Msg("object opened from s3")
	return obj, nil
}

func (s *s3Storage) Delete(ctx context.Context, objectPath string) error {
	if objectPath == "" {
		return nil
	}
	if err := s.client.RemoveObject(ctx, s.bucket, objectPath, minio.RemoveObjectOptions{}); err != nil {
		zlog.Logger.Error().Err(err).Str("path", objectPath).Msg("failed to delete object from s3")
		return fmt.Errorf("remove object %s: %w", objectPath, err)
	}
	zlog.Logger.Info().Str("path", objectPath).Msg("object deleted from s3")
	return nil
}

func (s *s3Storage) DeleteAll(ctx context.Context, originalPath, processedPath string) error {
	var lastErr error

	if err := s.Delete(ctx, originalPath); err != nil {
		lastErr = err
	}

	if processedPath != "" {
		if err := s.Delete(ctx, processedPath); err != nil {
			lastErr = err
		}
	}

	return lastErr
}
