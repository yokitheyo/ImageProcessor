package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/config"
	httpHandler "github.com/yokitheyo/imageprocessor/internal/handler/http"
	"github.com/yokitheyo/imageprocessor/internal/handler/middleware"
	"github.com/yokitheyo/imageprocessor/internal/helpers"
	infradatabase "github.com/yokitheyo/imageprocessor/internal/infrastructure/database"
	"github.com/yokitheyo/imageprocessor/internal/infrastructure/kafka"
	"github.com/yokitheyo/imageprocessor/internal/infrastructure/storage"
	"github.com/yokitheyo/imageprocessor/internal/repository/postgres"
	"github.com/yokitheyo/imageprocessor/internal/retry"
	"github.com/yokitheyo/imageprocessor/internal/usecase"
)

func main() {
	zlog.Init()
	zlog.Logger.Info().Msg("Starting Image Processor API Server")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to load config")
	}
	zlog.Logger.Info().
		Int("max_upload_size_mb", cfg.Server.MaxUploadSizeMB).
		Msg("Loaded server config")

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
		zlog.Logger.Fatal().Err(err).Msg("Migrations failed")
	}

	// Setup Storage
	storageService, err := storage.New(&cfg.Storage)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("Failed to initialize storage")
	}

	// Kafka Producer
	kafkaProducer := kafka.NewProducer(&cfg.Kafka)
	defer kafkaProducer.Close()

	// Repository + Usecase
	repo := postgres.NewImageRepository(database, retry.DefaultStrategy)
	imageUsecase := usecase.NewImageUsecase(repo, storageService, kafkaProducer)

	// Gin engine + middleware
	engine := ginext.New("api")
	engine.Use(
		middleware.ErrorHandlerMiddleware(),
		middleware.LoggerMiddleware(),
		middleware.CORSMiddleware(),
	)

	engine.GET("/health", func(c *ginext.Context) {
		c.JSON(http.StatusOK, ginext.H{"status": "ok"})
	})

	imageHandler := httpHandler.NewImageHandler(
		imageUsecase,
		cfg.Server.MaxUploadSizeMB,
		cfg.Processing.SupportedFormats,
	)
	imageHandler.RegisterRoutes(engine)

	engine.GET("/", func(c *ginext.Context) {
		c.File("./static/index.html")
	})
	engine.Static("/static", "./static")

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      engine,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSec) * time.Second,
	}

	go func() {
		zlog.Logger.Info().Str("addr", cfg.Server.Addr).Msg("Starting HTTP server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zlog.Logger.Fatal().Err(err).Msg("Failed to start API server")
		}
	}()

	<-ctx.Done()
	zlog.Logger.Info().Msg("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeoutSec)*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		zlog.Logger.Error().Err(err).Msg("HTTP server shutdown failed")
	} else {
		zlog.Logger.Info().Msg("HTTP server stopped gracefully")
	}

	if database != nil && database.Master != nil {
		if err := database.Master.Close(); err != nil {
			zlog.Logger.Error().Err(err).Msg("closing db master failed")
		} else {
			zlog.Logger.Info().Msg("db master closed")
		}
		for i, s := range database.Slaves {
			if err := s.Close(); err != nil {
				zlog.Logger.Error().Err(err).Int("slave_index", i).Msg("closing db slave failed")
			}
		}
	}

	zlog.Logger.Info().Msg("API shutdown complete")
}
