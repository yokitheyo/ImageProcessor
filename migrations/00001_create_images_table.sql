-- +goose Up
CREATE TABLE IF NOT EXISTS images (
    id VARCHAR(36) PRIMARY KEY,
    original_filename VARCHAR(255) NOT NULL,
    original_path TEXT NOT NULL,
    processed_path TEXT,
    mime_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL,
    width INTEGER,
    height INTEGER,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    processing_type VARCHAR(20) NOT NULL,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_images_status ON images(status);
CREATE INDEX IF NOT EXISTS idx_images_created_at ON images(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_images_processing_type ON images(processing_type);
CREATE INDEX IF NOT EXISTS idx_images_status_created ON images(status, created_at DESC);


-- +goose Down
DROP TRIGGER IF EXISTS update_images_updated_at ON images;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS images;
