package usecase

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"
	"errors"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/domain"
	"github.com/yokitheyo/imageprocessor/internal/infrastructure/storage"
)

type ImageUsecase struct {
	repo    domain.ImageRepository
	storage storage.Storage
	queue   domain.QueueService
}

func NewImageUsecase(
	repo domain.ImageRepository,
	storage storage.Storage,
	queue domain.QueueService,
) *ImageUsecase {
	return &ImageUsecase{
		repo:    repo,
		storage: storage,
		queue:   queue,
	}
}

func (u *ImageUsecase) UploadImage(
	ctx context.Context,
	filename string,
	mimeType string,
	size int64,
	reader io.Reader,
	processingType domain.ProcessingType,
) (*domain.Image, error) {
	imageID := uuid.New().String()
	ext := filepath.Ext(filename)
	uniqueFilename := fmt.Sprintf("%s%s", imageID, ext)

	originalPath, err := u.storage.SaveOriginal(ctx, uniqueFilename, reader)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("filename", filename).Msg("failed to save original file")
		return nil, fmt.Errorf("save original: %w", err)
	}

	now := time.Now()
	image := &domain.Image{
		ID:               imageID,
		OriginalFilename: filename,
		OriginalPath:     originalPath,
		MimeType:         mimeType,
		Size:             size,
		Status:           domain.StatusPending,
		ProcessingType:   processingType,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := u.repo.Create(ctx, image); err != nil {
		_ = u.storage.Delete(ctx, originalPath)
		zlog.Logger.Error().Err(err).Str("image_id", imageID).Msg("failed to create image record")
		return nil, fmt.Errorf("create image: %w", err)
	}

	if err := u.queue.PublishProcessingTask(ctx, imageID, processingType); err != nil {
		zlog.Logger.Error().Err(err).Str("image_id", imageID).Msg("failed to publish processing task")
	}

	zlog.Logger.Info().
		Str("image_id", imageID).
		Str("filename", filename).
		Str("processing_type", string(processingType)).
		Msg("image uploaded successfully")

	return image, nil
}

func (u *ImageUsecase) GetImage(ctx context.Context, id string) (*domain.Image, error) {
	return u.repo.FindByID(ctx, id)
}

func (u *ImageUsecase) GetImageFile(ctx context.Context, id string, useOriginal bool) (io.ReadCloser, string, error) {
	image, err := u.repo.FindByID(ctx, id)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("image_id", id).Msg("failed to find image by ID")
		return nil, "", err
	}

	var file io.ReadCloser
	var filename string

	if useOriginal {
		file, err = u.storage.GetOriginal(ctx, image.OriginalPath)
		filename = image.OriginalFilename
		if err != nil {
			zlog.Logger.Error().Err(err).Str("image_id", id).Str("path", image.OriginalPath).Msg("failed to get original file")
			if errors.Is(err, storage.ErrObjectNotFound) {
				return nil, "", domain.ErrImageNotFound
			}
		}
	} else {
		if !image.IsProcessed() {
			zlog.Logger.Warn().Str("image_id", id).Msg("image not processed yet")
			return nil, "", fmt.Errorf("image not processed yet")
		}
		file, err = u.storage.GetProcessed(ctx, image.ProcessedPath)
		if err != nil {
			zlog.Logger.Error().Err(err).Str("image_id", id).Str("path", image.ProcessedPath).Msg("failed to get processed file")
			if errors.Is(err, storage.ErrObjectNotFound) {
				return nil, "", domain.ErrImageNotFound
			}
			return nil, "", err
		}

		// Берем ext из реального ProcessedPath, чтобы избежать mismatch
		ext := filepath.Ext(image.ProcessedPath)
		baseName := image.OriginalFilename[:len(image.OriginalFilename)-len(filepath.Ext(image.OriginalFilename))]
		filename = fmt.Sprintf("%s_%s%s", baseName, image.ProcessingType, ext)
	}

	return file, filename, nil
}

func (u *ImageUsecase) DeleteImage(ctx context.Context, id string) error {
	image, err := u.repo.FindByID(ctx, id)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("image_id", id).Msg("failed to find image for delete")
		return err
	}

	if err := u.storage.DeleteAll(ctx, image.OriginalPath, image.ProcessedPath); err != nil {
		zlog.Logger.Error().Err(err).Str("image_id", id).Msg("failed to delete files")
	}

	if err := u.repo.Delete(ctx, id); err != nil {
		zlog.Logger.Error().Err(err).Str("image_id", id).Msg("failed to delete image record")
		return err
	}

	zlog.Logger.Info().Str("image_id", id).Msg("image deleted successfully")
	return nil
}

func (u *ImageUsecase) ListImages(ctx context.Context, limit, offset int) ([]*domain.Image, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	images, err := u.repo.List(ctx, limit, offset)
	if err != nil {
		zlog.Logger.Error().Err(err).Msg("failed to list images")
		return nil, err
	}
	return images, nil
}
