package worker

import (
	"context"
	"fmt"

	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/domain"
	"github.com/yokitheyo/imageprocessor/internal/dto"
)

// ImageWorker обрабатывает задачи из очереди
type ImageWorker struct {
	processorService domain.ProcessorService
}

// NewImageWorker создает нового воркера
func NewImageWorker(processorService domain.ProcessorService) *ImageWorker {
	return &ImageWorker{
		processorService: processorService,
	}
}

func (w *ImageWorker) HandleProcessingTask(ctx context.Context, task *dto.ProcessImageRequest) error {
	// Проверка валидности ProcessingType
	if task.ProcessingType != string(domain.ProcessingResize) &&
		task.ProcessingType != string(domain.ProcessingThumbnail) &&
		task.ProcessingType != string(domain.ProcessingWatermark) {
		zlog.Logger.Error().
			Str("image_id", task.ImageID).
			Str("processing_type", task.ProcessingType).
			Msg("invalid processing type")
		return fmt.Errorf("invalid processing type: %s", task.ProcessingType)
	}

	zlog.Logger.Info().
		Str("image_id", task.ImageID).
		Str("processing_type", task.ProcessingType).
		Msg("starting image processing task")

	// Вызов usecase, который уже обрабатывает и сохраняет изображение
	if err := w.processorService.ProcessImage(ctx, task.ImageID); err != nil {
		zlog.Logger.Error().
			Err(err).
			Str("image_id", task.ImageID).
			Str("processing_type", task.ProcessingType).
			Msg("failed to process image")
		return fmt.Errorf("process image %s: %w", task.ImageID, err)
	}

	zlog.Logger.Info().
		Str("image_id", task.ImageID).
		Msg("image processed successfully")

	return nil
}
