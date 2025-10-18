package kafka

import (
	"context"
	"encoding/json"
	"time"

	wbfkafka "github.com/wb-go/wbf/kafka"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/domain"

	"github.com/yokitheyo/imageprocessor/internal/config"
	"github.com/yokitheyo/imageprocessor/internal/dto"
)

type Producer struct {
	client *wbfkafka.Producer
	topic  string
}

// NewProducer создаёт Kafka producer через wbf.
func NewProducer(cfg *config.KafkaConfig) *Producer {
	client := wbfkafka.NewProducer(cfg.Brokers, cfg.Topic)
	zlog.Logger.Info().
		Strs("brokers", cfg.Brokers).
		Str("topic", cfg.Topic).
		Msg("Kafka producer initialized (wbf)")
	return &Producer{
		client: client,
		topic:  cfg.Topic,
	}
}

// Send отправляет сообщение без ретраев.
func (p *Producer) Send(ctx context.Context, task dto.ProcessImageRequest) error {
	data, err := json.Marshal(task)
	if err != nil {
		zlog.Logger.Error().
			Err(err).
			Str("image_id", task.ImageID).
			Str("processing_type", task.ProcessingType).
			Msg("Failed to marshal task")
		return err
	}
	if err := p.client.Send(ctx, nil, data); err != nil {
		zlog.Logger.Error().
			Err(err).
			Str("image_id", task.ImageID).
			Str("processing_type", task.ProcessingType).
			Msg("Failed to send Kafka message")
		return err
	}
	zlog.Logger.Info().
		Str("image_id", task.ImageID).
		Str("processing_type", task.ProcessingType).
		Msg("Message sent to Kafka")
	return nil
}

// SendWithRetry — с повторными попытками через стратегию.
func (p *Producer) SendWithRetry(ctx context.Context, task dto.ProcessImageRequest) error {
	data, err := json.Marshal(task)
	if err != nil {
		zlog.Logger.Error().
			Err(err).
			Str("image_id", task.ImageID).
			Str("processing_type", task.ProcessingType).
			Msg("Failed to marshal task")
		return err
	}
	strategy := retry.Strategy{
		Attempts: 3,
		Delay:    2 * time.Second,
		Backoff:  2.0, // Увеличен для экспоненциальной задержки
	}
	if err := p.client.SendWithRetry(ctx, strategy, nil, data); err != nil {
		zlog.Logger.Error().
			Err(err).
			Str("image_id", task.ImageID).
			Str("processing_type", task.ProcessingType).
			Msg("Failed to send Kafka message with retry")
		return err
	}
	zlog.Logger.Info().
		Str("image_id", task.ImageID).
		Str("processing_type", task.ProcessingType).
		Msg("Message sent to Kafka with retry")
	return nil
}

// Close закрывает продюсер.
func (p *Producer) Close() error {
	if err := p.client.Close(); err != nil {
		zlog.Logger.Error().Err(err).Msg("Failed to close Kafka producer")
		return err
	}
	zlog.Logger.Info().Msg("Kafka producer closed successfully")
	return nil
}

func (p *Producer) PublishProcessingTask(ctx context.Context, imageID string, processingType domain.ProcessingType) error {
	task := dto.ProcessImageRequest{
		ImageID:        imageID,
		ProcessingType: string(processingType),
	}
	return p.SendWithRetry(ctx, task) // Используем SendWithRetry для надёжности
}
