package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/yokitheyo/imageprocessor/internal/helpers"

	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/config"
	infradatabase "github.com/yokitheyo/imageprocessor/internal/infrastructure/database"
	"github.com/yokitheyo/imageprocessor/internal/infrastructure/kafka"
	"github.com/yokitheyo/imageprocessor/internal/infrastructure/processor"
	"github.com/yokitheyo/imageprocessor/internal/infrastructure/storage"
	"github.com/yokitheyo/imageprocessor/internal/repository/postgres"
	"github.com/yokitheyo/imageprocessor/internal/retry"
	"github.com/yokitheyo/imageprocessor/internal/usecase"
	"github.com/yokitheyo/imageprocessor/internal/worker"
)

func main() {
	zlog.Init()
	zlog.Logger.Info().Msg("Starting Image Processor Worker")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load config (config.Load will look for default paths if empty)
	cfg, err := config.Load("")
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to load config")
	}

	connectRetries := cfg.Database.ConnectRetries
	connectDelay := cfg.Database.ConnectRetryDelaySec
	if connectRetries == 0 {
		connectRetries = 15
	}
	if connectDelay == 0 {
		connectDelay = 3
	}

	masterDSN := cfg.Database.DSN
	slaves := []string{}
	if strings.TrimSpace(cfg.Database.Slaves) != "" {
		slaves = helpers.SplitAndTrim(cfg.Database.Slaves, ",")
	}
	dbOpts := &dbpg.Options{
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: time.Duration(cfg.Database.ConnMaxLifetimeSec) * time.Second,
	}

	database, err := infradatabase.ConnectWithRetries(masterDSN, slaves, dbOpts, connectRetries, connectDelay)
	if err != nil || database == nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to connect to database after all retries")
	}

	// Run migrations
	zlog.Logger.Info().Msg("Running database migrations...")
	if err := infradatabase.RunMigrations(database, cfg.Migrations.Path); err != nil {
		zlog.Logger.Warn().Err(err).Msg("Migrations warning (might be already applied)")
	}

	// Setup Storage
	storageService, err := storage.New(&cfg.Storage)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("Failed to initialize storage")
	}

	// Setup Image Processor
	imageProcessor := processor.NewImageProcessor(&cfg.Processing)

	// Setup Repository and Usecase
	repo := postgres.NewImageRepository(database, retry.DefaultStrategy)
	processorUsecase := usecase.NewProcessorUsecase(repo, storageService, imageProcessor)
	imageWorker := worker.NewImageWorker(processorUsecase)

	// Kafka Consumer
	kafkaConsumer, err := kafka.NewConsumer(&cfg.Kafka, imageWorker.HandleProcessingTask)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("Failed to initialize Kafka consumer")
	}
	defer kafkaConsumer.Close()

	go func() {
		if err := kafkaConsumer.Start(ctx); err != nil {
			zlog.Logger.Error().Err(err).Msg("Kafka consumer error")
		}
	}()

	<-ctx.Done()
	zlog.Logger.Info().Msg("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	<-shutdownCtx.Done()

	if database != nil && database.Master != nil {
		database.Master.Close()
		for _, s := range database.Slaves {
			if s != nil {
				s.Close()
			}
		}
	}

	zlog.Logger.Info().Msg("Worker shutdown complete")
}
