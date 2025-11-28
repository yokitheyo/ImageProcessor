package kafka

import (
	"context"
	"encoding/json"
	"time"

	wbfkafka "github.com/wb-go/wbf/kafka"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"

	"github.com/yokitheyo/imageprocessor/internal/config"
	"github.com/yokitheyo/imageprocessor/internal/dto"
)

type MessageHandler func(ctx context.Context, task *dto.ProcessImageRequest) error

type Consumer struct {
	client  *wbfkafka.Consumer
	handler MessageHandler
	topic   string
}

func NewConsumer(cfg *config.KafkaConfig, handler MessageHandler) (*Consumer, error) {
	client := wbfkafka.NewConsumer(cfg.Brokers, cfg.Topic, cfg.GroupID)

	zlog.Logger.Info().
		Strs("brokers", cfg.Brokers).
		Str("topic", cfg.Topic).
		Str("group_id", cfg.GroupID).
		Msg("Kafka consumer initialized (WB)")

	return &Consumer{
		client:  client,
		handler: handler,
		topic:   cfg.Topic,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	strategy := retry.Strategy{
		Attempts: 3,
		Delay:    2 * time.Second,
		Backoff:  2.0,
	}

	for {
		select {
		case <-ctx.Done():
			zlog.Logger.Info().Msg("Kafka consumer stopped")
			return nil
		default:
			msg, err := c.client.FetchWithRetry(ctx, strategy)
			if err != nil {
				zlog.Logger.Error().Err(err).Msg("Failed to fetch Kafka message")
				time.Sleep(time.Second)
				continue
			}

			var task dto.ProcessImageRequest
			if err := json.Unmarshal(msg.Value, &task); err != nil {
				zlog.Logger.Error().
					Err(err).
					Bytes("msg", msg.Value).
					Msg("Failed to unmarshal message")
				continue
			}

			if task.ImageID == "" || task.ProcessingType == "" {
				zlog.Logger.Error().
					Str("image_id", task.ImageID).
					Str("processing_type", task.ProcessingType).
					Msg("Invalid task: empty ImageID or ProcessingType")
				continue
			}

			zlog.Logger.Info().
				Str("image_id", task.ImageID).
				Str("processing_type", task.ProcessingType).
				Msg("Received new Kafka task")

			if err := c.handler(ctx, &task); err != nil {
				zlog.Logger.Error().
					Err(err).
					Str("image_id", task.ImageID).
					Str("processing_type", task.ProcessingType).
					Msg("Task processing failed")
				continue
			}

			if err := c.client.Commit(ctx, msg); err != nil {
				zlog.Logger.Error().
					Err(err).
					Str("image_id", task.ImageID).
					Msg("Failed to commit message")
				continue
			}

			zlog.Logger.Info().
				Str("image_id", task.ImageID).
				Str("processing_type", task.ProcessingType).
				Msg("Task processed and committed successfully")
		}
	}
}

func (c *Consumer) Close() error {
	if err := c.client.Close(); err != nil {
		zlog.Logger.Error().Err(err).Msg("Failed to close Kafka consumer")
		return err
	}
	zlog.Logger.Info().Msg("Kafka consumer closed successfully")
	return nil
}
