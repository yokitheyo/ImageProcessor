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
	WatermarkImage   string   `mapstructure:"watermark_image"`
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

func validateConfig(cfg *Config) error {
	// Server
	if cfg.Server.Addr == "" {
		return fmt.Errorf("server.addr is required")
	}
	if cfg.Server.ShutdownTimeoutSec <= 0 {
		return fmt.Errorf("server.shutdown_timeout_sec must be positive")
	}
	if cfg.Server.ReadTimeoutSec <= 0 {
		return fmt.Errorf("server.read_timeout_sec must be positive")
	}
	if cfg.Server.WriteTimeoutSec <= 0 {
		return fmt.Errorf("server.write_timeout_sec must be positive")
	}
	if cfg.Server.MaxUploadSizeMB <= 0 {
		return fmt.Errorf("server.max_upload_size_mb must be positive")
	}

	// Database
	if cfg.Database.DSN == "" {
		return fmt.Errorf("database.dsn is required")
	}
	if cfg.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("database.max_open_conns must be positive")
	}
	if cfg.Database.MaxIdleConns < 0 {
		return fmt.Errorf("database.max_idle_conns must be non-negative")
	}

	// Migrations
	if cfg.Migrations.Path == "" {
		return fmt.Errorf("migrations.path is required")
	}

	// Kafka
	if len(cfg.Kafka.Brokers) == 0 {
		return fmt.Errorf("kafka.brokers must contain at least one broker")
	}
	if cfg.Kafka.Topic == "" {
		return fmt.Errorf("kafka.topic is required")
	}
	if cfg.Kafka.GroupID == "" {
		return fmt.Errorf("kafka.group_id is required")
	}

	// Storage
	if cfg.Storage.Type == "" {
		return fmt.Errorf("storage.type is required (local|s3)")
	}
	if cfg.Storage.Type != "local" && cfg.Storage.Type != "s3" {
		return fmt.Errorf("storage.type must be 'local' or 's3'")
	}
	if cfg.Storage.Type == "local" && cfg.Storage.LocalPath == "" {
		return fmt.Errorf("storage.local_path is required for local storage")
	}

	// Processing
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
	if cfg.Storage.Type == "s3" {
		if cfg.Storage.S3Endpoint == "" {
			return fmt.Errorf("storage.s3_endpoint is required for s3 storage")
		}
		if cfg.Storage.S3Bucket == "" {
			return fmt.Errorf("storage.s3_bucket is required for s3 storage")
		}
		if cfg.Storage.S3AccessKey == "" || cfg.Storage.S3SecretKey == "" {
			return fmt.Errorf("storage.s3_access_key and storage.s3_secret_key are required for s3 storage")
		}
	}

	if len(cfg.Processing.SupportedFormats) == 0 {
		return fmt.Errorf("processing.supported_formats must contain at least one format")
	}
	if cfg.Logging.Level == "" {
		return fmt.Errorf("logging.level is required")
	}

	return nil
}
