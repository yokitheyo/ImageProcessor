package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/config"
)

type Storage interface {
	SaveOriginal(ctx context.Context, filename string, reader io.Reader) (string, error)
	SaveProcessed(ctx context.Context, filename string, reader io.Reader) (string, error)
	GetOriginal(ctx context.Context, path string) (io.ReadCloser, error)
	GetProcessed(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	DeleteAll(ctx context.Context, originalPath, processedPath string) error
}

func New(cfg *config.StorageConfig) (Storage, error) {
	switch cfg.Type {
	case "local":
		zlog.Logger.Info().Msg("Initializing local storage")
		return NewLocalStorage(cfg)
	/*	case "s3":
		zlog.Logger.Info().Msg("Initializing S3 storage")
		return NewS3Storage(cfg)*/
	default:
		zlog.Logger.Error().Str("type", cfg.Type).Msg("Unsupported storage type, use 'local' or 's3'")
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}
