package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/config"
)

type localStorage struct {
	basePath     string
	originalDir  string
	processedDir string
}

func NewLocalStorage(cfg *config.StorageConfig) (Storage, error) {
	if cfg.LocalPath == "" {
		return nil, fmt.Errorf("LocalPath is empty, set storage.local_path in config or env")
	}
	if cfg.OriginalDir == "" {
		cfg.OriginalDir = "original"
	}
	if cfg.ProcessedDir == "" {
		cfg.ProcessedDir = "processed"
	}

	storage := &localStorage{
		basePath:     cfg.LocalPath,
		originalDir:  cfg.OriginalDir,
		processedDir: cfg.ProcessedDir,
	}

	originalPath := filepath.Join(storage.basePath, storage.originalDir)
	processedPath := filepath.Join(storage.basePath, storage.processedDir)

	if err := os.MkdirAll(originalPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create original directory: %w", err)
	}
	if err := os.MkdirAll(processedPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create processed directory: %w", err)
	}

	return storage, nil
}

func (s *localStorage) SaveOriginal(ctx context.Context, filename string, reader io.Reader) (string, error) {
	return s.saveFile(ctx, s.originalDir, filename, reader)
}

func (s *localStorage) SaveProcessed(ctx context.Context, filename string, reader io.Reader) (string, error) {
	return s.saveFile(ctx, s.processedDir, filename, reader)
}

func (s *localStorage) saveFile(ctx context.Context, dir, filename string, reader io.Reader) (string, error) {
	if reader == nil {
		zlog.Logger.Error().Str("filename", filename).Msg("reader is nil")
		return "", fmt.Errorf("reader is nil")
	}

	fullPath := filepath.Join(s.basePath, dir, filename)

	// Проверка существования файла (для отладки)
	if _, err := os.Stat(fullPath); err == nil {
		zlog.Logger.Warn().Str("path", fullPath).Msg("file already exists, will be overwritten")
	}

	file, err := os.Create(fullPath)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("path", fullPath).Msg("failed to create file")
		return "", fmt.Errorf("create file %s: %w", fullPath, err)
	}
	defer file.Close()

	written, err := io.Copy(file, reader)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("path", fullPath).Msg("failed to write file")
		return "", fmt.Errorf("write file %s: %w", fullPath, err)
	}
	if written == 0 {
		zlog.Logger.Error().Str("path", fullPath).Msg("no bytes written to file")
		return "", fmt.Errorf("no bytes written to file %s", fullPath)
	}

	relativePath := filepath.Join(dir, filename)
	zlog.Logger.Info().
		Str("path", relativePath).
		Str("ext", filepath.Ext(filename)).
		Int64("bytes", written).
		Msg("file saved successfully")

	return relativePath, nil
}

func (s *localStorage) GetOriginal(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.getFile(ctx, path)
}

func (s *localStorage) GetProcessed(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.getFile(ctx, path)
}

func (s *localStorage) getFile(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, path)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			zlog.Logger.Error().Str("path", fullPath).Msg("file not found")
			return nil, fmt.Errorf("file not found: %s", path)
		}
		zlog.Logger.Error().Err(err).Str("path", fullPath).Msg("failed to open file")
		return nil, fmt.Errorf("open file %s: %w", fullPath, err)
	}

	// Логируем размер файла для отладки
	if stat, err := file.Stat(); err == nil {
		zlog.Logger.Info().Str("path", fullPath).Int64("size", stat.Size()).Msg("file opened successfully")
	}

	return file, nil
}

func (s *localStorage) Delete(ctx context.Context, path string) error {
	if path == "" {
		return nil
	}

	fullPath := filepath.Join(s.basePath, path)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			zlog.Logger.Warn().Str("path", fullPath).Msg("file not found, skipping delete")
			return nil
		}
		zlog.Logger.Error().Err(err).Str("path", fullPath).Msg("failed to delete file")
		return fmt.Errorf("delete file %s: %w", fullPath, err)
	}

	zlog.Logger.Info().Str("path", path).Msg("file deleted successfully")
	return nil
}

func (s *localStorage) DeleteAll(ctx context.Context, originalPath, processedPath string) error {
	var lastErr error

	if err := s.Delete(ctx, originalPath); err != nil {
		lastErr = err
	}

	if processedPath != "" {
		if err := s.Delete(ctx, processedPath); err != nil {
			lastErr = err
		}
	}

	return lastErr
}
