package usecase

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/disintegration/imaging"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/domain"
	"github.com/yokitheyo/imageprocessor/internal/infrastructure/processor"
	"github.com/yokitheyo/imageprocessor/internal/infrastructure/storage"
)

type ProcessorUsecase struct {
	repo      domain.ImageRepository
	storage   storage.Storage
	processor *processor.ImageProcessor
}

func NewProcessorUsecase(
	repo domain.ImageRepository,
	storage storage.Storage,
	processor *processor.ImageProcessor,
) *ProcessorUsecase {
	return &ProcessorUsecase{
		repo:      repo,
		storage:   storage,
		processor: processor,
	}
}

func (u *ProcessorUsecase) ProcessImage(ctx context.Context, imageID string) error {
	image, err := u.repo.FindByID(ctx, imageID)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("image_id", imageID).Msg("failed to find image")
		return fmt.Errorf("find image: %w", err)
	}

	if !image.CanBeProcessed() {
		zlog.Logger.Warn().
			Str("image_id", imageID).
			Str("status", string(image.Status)).
			Msg("image cannot be processed in current status")
		return nil
	}

	image.MarkAsProcessing()
	if err := u.repo.Update(ctx, image); err != nil {
		zlog.Logger.Error().Err(err).Str("image_id", imageID).Msg("failed to update status to processing")
		return fmt.Errorf("update status to processing: %w", err)
	}

	zlog.Logger.Info().
		Str("image_id", imageID).
		Str("processing_type", string(image.ProcessingType)).
		Msg("starting image processing")

	originalFile, err := u.storage.GetOriginal(ctx, image.OriginalPath)
	if err != nil {
		image.MarkAsFailed(fmt.Sprintf("failed to get original file: %v", err))
		_ = u.repo.Update(ctx, image)
		zlog.Logger.Error().Err(err).Str("image_id", imageID).Str("path", image.OriginalPath).Msg("failed to get original file")
		return fmt.Errorf("get original file: %w", err)
	}
	defer originalFile.Close()

	img, err := imaging.Decode(originalFile, imaging.AutoOrientation(true))
	if err != nil {
		image.MarkAsFailed(fmt.Sprintf("failed to decode original file: %v", err))
		_ = u.repo.Update(ctx, image)
		zlog.Logger.Error().Err(err).Str("image_id", imageID).Str("path", image.OriginalPath).Msg("failed to decode original image")
		return fmt.Errorf("decode original image: %w", err)
	}
	if img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
		image.MarkAsFailed("original image is empty")
		_ = u.repo.Update(ctx, image)
		zlog.Logger.Error().Str("image_id", imageID).Str("path", image.OriginalPath).Msg("original image is empty")
		return fmt.Errorf("original image is empty")
	}
	zlog.Logger.Info().
		Str("image_id", imageID).
		Int("original_width", img.Bounds().Dx()).
		Int("original_height", img.Bounds().Dy()).
		Msg("Original image decoded successfully")

	if seeker, ok := originalFile.(io.Seeker); ok {
		_, err = seeker.Seek(0, io.SeekStart)
		if err != nil {
			image.MarkAsFailed(fmt.Sprintf("failed to seek original file: %v", err))
			_ = u.repo.Update(ctx, image)
			zlog.Logger.Error().Err(err).Str("image_id", imageID).Msg("failed to seek original file")
			return fmt.Errorf("seek original file: %w", err)
		}
	}

	processedImg, err := u.processor.Process(originalFile, image.ProcessingType)
	if err != nil {
		image.MarkAsFailed(fmt.Sprintf("processing failed: %v", err))
		_ = u.repo.Update(ctx, image)
		zlog.Logger.Error().
			Err(err).
			Str("image_id", imageID).
			Str("processing_type", string(image.ProcessingType)).
			Msg("failed to process image")
		return fmt.Errorf("process image: %w", err)
	}

	width, height := processor.GetImageDimensions(processedImg)
	if width == 0 || height == 0 {
		image.MarkAsFailed("processed image is empty")
		_ = u.repo.Update(ctx, image)
		zlog.Logger.Error().
			Str("image_id", imageID).
			Str("processing_type", string(image.ProcessingType)).
			Int("resize_width", u.processor.ResizeWidth()).
			Int("resize_height", u.processor.ResizeHeight()).
			Int("thumbnail_width", u.processor.ThumbnailWidth()).
			Int("thumbnail_height", u.processor.ThumbnailHeight()).
			Msg("processed image is empty")
		return fmt.Errorf("processed image is empty")
	}

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, processedImg, imaging.JPEG, imaging.JPEGQuality(95)); err != nil {
		image.MarkAsFailed(fmt.Sprintf("encoding failed: %v", err))
		_ = u.repo.Update(ctx, image)
		zlog.Logger.Error().Err(err).Str("image_id", imageID).Msg("failed to encode image")
		return fmt.Errorf("encode image: %w", err)
	}

	if buf.Len() == 0 {
		image.MarkAsFailed("empty buffer after encoding")
		_ = u.repo.Update(ctx, image)
		zlog.Logger.Error().
			Str("image_id", imageID).
			Str("processing_type", string(image.ProcessingType)).
			Int("width", width).
			Int("height", height).
			Msg("empty buffer after encoding")
		return fmt.Errorf("empty buffer after encoding")
	}

	processedFilename := fmt.Sprintf("%s_%s.jpg", image.ID, image.ProcessingType)
	processedPath, err := u.storage.SaveProcessed(ctx, processedFilename, &buf)
	if err != nil {
		image.MarkAsFailed(fmt.Sprintf("failed to save processed file: %v", err))
		_ = u.repo.Update(ctx, image)
		zlog.Logger.Error().Err(err).Str("image_id", imageID).Str("path", processedFilename).Msg("failed to save processed file")
		return fmt.Errorf("save processed file: %w", err)
	}

	image.MarkAsCompleted(processedPath, width, height)
	if err := u.repo.Update(ctx, image); err != nil {
		zlog.Logger.Error().Err(err).Str("image_id", imageID).Msg("failed to update status to completed")
		return fmt.Errorf("update status to completed: %w", err)
	}

	zlog.Logger.Info().
		Str("image_id", imageID).
		Str("processed_path", processedPath).
		Int("width", width).
		Int("height", height).
		Int("buffer_size", buf.Len()).
		Msg("image processed successfully")

	return nil
}
