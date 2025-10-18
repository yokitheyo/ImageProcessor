package config

import (
	"fmt"
	"os"

	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/zlog"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Migrations MigrationsConfig `mapstructure:"migrations"`
	Kafka      KafkaConfig      `mapstructure:"kafka"`
	Storage    StorageConfig    `mapstructure:"storage"`
	Processing ProcessingConfig `mapstructure:"processing"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

type ServerConfig struct {
	Addr               string `mapstructure:"addr"`
	ShutdownTimeoutSec int    `mapstructure:"shutdown_timeout_sec"`
	ReadTimeoutSec     int    `mapstructure:"read_timeout_sec"`
	WriteTimeoutSec    int    `mapstructure:"write_timeout_sec"`
	MaxUploadSizeMB    int    `mapstructure:"max_upload_size_mb"`
}

type DatabaseConfig struct {
	DSN                  string `mapstructure:"dsn"`
	Slaves               string `mapstructure:"slaves"`
	MaxOpenConns         int    `mapstructure:"max_open_conns"`
	MaxIdleConns         int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeSec   int    `mapstructure:"conn_max_lifetime_sec"`
	ConnectRetries       int    `mapstructure:"connect_retries"`
	ConnectRetryDelaySec int    `mapstructure:"connect_retry_delay_sec"`
}

type MigrationsConfig struct {
	Path string `mapstructure:"path"`
}

type KafkaConfig struct {
	Brokers              []string `mapstructure:"brokers"`
	Topic                string   `mapstructure:"topic"`
	GroupID              string   `mapstructure:"group_id"`
	Partition            int      `mapstructure:"partition"`
	SessionTimeoutSec    int      `mapstructure:"session_timeout_sec"`
	HeartbeatIntervalSec int      `mapstructure:"heartbeat_interval_sec"`
}

type StorageConfig struct {
	Type         string `mapstructure:"type"`
	LocalPath    string `mapstructure:"local_path"`
	OriginalDir  string `mapstructure:"original_dir"`
	ProcessedDir string `mapstructure:"processed_dir"`

	S3Endpoint  string `mapstructure:"s3_endpoint"`
	S3AccessKey string `mapstructure:"s3_access_key"`
	S3SecretKey string `mapstructure:"s3_secret_key"`
	S3Bucket    string `mapstructure:"s3_bucket"`
	S3Region    string `mapstructure:"s3_region"`
	S3UseSSL    bool   `mapstructure:"s3_use_ssl"`
}

type ProcessingConfig struct {
	ResizeWidth      int      `mapstructure:"resize_width"`
	ResizeHeight     int      `mapstructure:"resize_height"`
	ThumbnailWidth   int      `mapstructure:"thumbnail_width"`
	ThumbnailHeight  int      `mapstructure:"thumbnail_height"`
	WatermarkText    string   `mapstructure:"watermark_text"`
	WatermarkOpacity int      `mapstructure:"watermark_opacity"`
	OutputQuality    int      `mapstructure:"output_quality"`
	SupportedFormats []string `mapstructure:"supported_formats"`
}

type LoggingConfig struct {
	Level string `mapstructure:"level"`
}

func Load(path string) (*Config, error) {
	cfg := config.New()

	configPath := path
	if configPath == "" {
		if _, err := os.Stat("config.yaml"); err == nil {
			configPath = "config.yaml"
		} else if _, err := os.Stat("/app/config.yaml"); err == nil {
			configPath = "/app/config.yaml"
		} else {
			return nil, fmt.Errorf("config.yaml not found")
		}
	}

	setDefaults(cfg)

	envPath := ".env"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		envPath = ""
	}

	if err := cfg.Load(configPath, envPath, "APP"); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	appConfig := &Config{}
	if err := cfg.Unmarshal(appConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validateConfig(appConfig); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	zlog.Logger.Info().
		Str("local_path", appConfig.Storage.LocalPath).
		Str("original_dir", appConfig.Storage.OriginalDir).
		Str("processed_dir", appConfig.Storage.ProcessedDir).
		Int("resize_width", appConfig.Processing.ResizeWidth).
		Int("resize_height", appConfig.Processing.ResizeHeight).
		Msg("Config loaded successfully via wbf")

	return appConfig, nil
}

func setDefaults(cfg *config.Config) {
	// Server
	cfg.SetDefault("server.addr", ":8080")
	cfg.SetDefault("server.shutdown_timeout_sec", 15)
	cfg.SetDefault("server.read_timeout_sec", 30)
	cfg.SetDefault("server.write_timeout_sec", 30)
	cfg.SetDefault("server.max_upload_size_mb", 10)

	// Database
	cfg.SetDefault("database.max_open_conns", 25)
	cfg.SetDefault("database.max_idle_conns", 5)
	cfg.SetDefault("database.conn_max_lifetime_sec", 1800)
	cfg.SetDefault("database.connect_retries", 10)
	cfg.SetDefault("database.connect_retry_delay_sec", 3)

	// Migrations
	cfg.SetDefault("migrations.path", "./migrations")

	// Kafka
	cfg.SetDefault("kafka.topic", "image-processing")
	cfg.SetDefault("kafka.group_id", "image-processor-workers")
	cfg.SetDefault("kafka.partition", 0)
	cfg.SetDefault("kafka.session_timeout_sec", 30)
	cfg.SetDefault("kafka.heartbeat_interval_sec", 3)

	// Storage
	cfg.SetDefault("storage.type", "local")
	cfg.SetDefault("storage.local_path", "./storage")
	cfg.SetDefault("storage.original_dir", "original")
	cfg.SetDefault("storage.processed_dir", "processed")

	// Processing
	cfg.SetDefault("processing.resize_width", 800)
	cfg.SetDefault("processing.resize_height", 600)
	cfg.SetDefault("processing.thumbnail_width", 200)
	cfg.SetDefault("processing.thumbnail_height", 150)
	cfg.SetDefault("processing.watermark_opacity", 128)
	cfg.SetDefault("processing.output_quality", 95)
	cfg.SetDefault("processing.supported_formats", []string{"jpg", "jpeg", "png", "gif"})

	// Logging
	cfg.SetDefault("logging.level", "info")
}

func validateConfig(cfg *Config) error {
	if cfg.Storage.Type == "local" && cfg.Storage.LocalPath == "" {
		return fmt.Errorf("storage.local_path is required for local storage")
	}
	if cfg.Processing.ResizeWidth <= 0 {
		return fmt.Errorf("processing.resize_width must be positive")
	}
	if cfg.Processing.ResizeHeight <= 0 {
		return fmt.Errorf("processing.resize_height must be positive")
	}
	if cfg.Processing.ThumbnailWidth <= 0 {
		return fmt.Errorf("processing.thumbnail_width must be positive")
	}
	if cfg.Processing.ThumbnailHeight <= 0 {
		return fmt.Errorf("processing.thumbnail_height must be positive")
	}
	return nil
}
