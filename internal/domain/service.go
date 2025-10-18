package domain

import (
	"context"
	"io"
)

type ImageService interface {
	UploadImage(ctx context.Context, filename string, mimeType string, size int64, reader io.Reader, processingType ProcessingType) (*Image, error)
	GetImage(ctx context.Context, id string) (*Image, error)
	GetImageFile(ctx context.Context, id string, useOriginal bool) (io.ReadCloser, string, error)
	DeleteImage(ctx context.Context, id string) error
	ListImages(ctx context.Context, limit, offset int) ([]*Image, error)
}

type ProcessorService interface {
	ProcessImage(ctx context.Context, imageID string) error
}

type StorageService interface {
	SaveOriginal(ctx context.Context, filename string, reader io.Reader) (string, error)
	SaveProcessed(ctx context.Context, filename string, reader io.Reader) (string, error)
	GetOriginal(ctx context.Context, path string) (io.ReadCloser, error)
	GetProcessed(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	DeleteAll(ctx context.Context, originalPath, processedPath string) error
}

type QueueService interface {
	PublishProcessingTask(ctx context.Context, imageID string, processingType ProcessingType) error
	Close() error
}
