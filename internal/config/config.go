package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
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
	Type         string `mapstructure:"type"` // local or s3
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
	v := viper.New()
	v.SetConfigType("yaml")

	// Если путь не указан, ищем стандартные места
	if path == "" {
		if _, err := os.Stat("config.yaml"); err == nil {
			path = "config.yaml"
		} else if _, err := os.Stat("/app/config.yaml"); err == nil {
			path = "/app/config.yaml"
		} else {
			zlog.Logger.Fatal().Msg("No config.yaml found")
		}
	}

	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to read config")
	}
	fmt.Println("Loaded config from:", path)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to unmarshal config")
	}

	// Валидация
	if err := validateConfig(&cfg); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("config validation failed")
	}

	zlog.Logger.Info().
		Str("local_path", cfg.Storage.LocalPath).
		Str("original_dir", cfg.Storage.OriginalDir).
		Str("processed_dir", cfg.Storage.ProcessedDir).
		Int("resize_width", cfg.Processing.ResizeWidth).
		Int("resize_height", cfg.Processing.ResizeHeight).
		Msg("Config loaded successfully")

	return &cfg, nil
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
	// Добавьте другие проверки по необходимости
	return nil
}
