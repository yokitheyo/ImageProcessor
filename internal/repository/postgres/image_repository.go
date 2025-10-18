package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/domain"
)

type imageRepository struct {
	db       *dbpg.DB
	strategy retry.Strategy
}

func NewImageRepository(db *dbpg.DB, strategy retry.Strategy) domain.ImageRepository {
	return &imageRepository{
		db:       db,
		strategy: strategy,
	}
}

func (r *imageRepository) Create(ctx context.Context, image *domain.Image) error {
	query := `
		INSERT INTO images (
			id, original_filename, original_path, processed_path,
			mime_type, size, width, height, status, processing_type,
			error_message, created_at, updated_at, processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.ExecWithRetry(ctx, r.strategy, query,
		image.ID,
		image.OriginalFilename,
		image.OriginalPath,
		nullString(image.ProcessedPath),
		image.MimeType,
		image.Size,
		nullInt(image.Width),
		nullInt(image.Height),
		image.Status,
		image.ProcessingType,
		nullString(image.ErrorMessage),
		image.CreatedAt,
		image.UpdatedAt,
		image.ProcessedAt,
	)

	if err != nil {
		zlog.Logger.Error().Err(err).Str("image_id", image.ID).Msg("failed to create image")
		return fmt.Errorf("create image: %w", err)
	}

	zlog.Logger.Info().Str("image_id", image.ID).Msg("image created successfully")
	return nil
}

func (r *imageRepository) FindByID(ctx context.Context, id string) (*domain.Image, error) {
	query := `
		SELECT id, original_filename, original_path, processed_path,
			   mime_type, size, width, height, status, processing_type,
			   error_message, created_at, updated_at, processed_at
		FROM images
		WHERE id = $1
	`

	var img domain.Image
	var processedPath, errorMsg sql.NullString
	var width, height sql.NullInt32
	var processedAt sql.NullTime

	row := r.db.Master.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&img.ID,
		&img.OriginalFilename,
		&img.OriginalPath,
		&processedPath,
		&img.MimeType,
		&img.Size,
		&width,
		&height,
		&img.Status,
		&img.ProcessingType,
		&errorMsg,
		&img.CreatedAt,
		&img.UpdatedAt,
		&processedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrImageNotFound
	}
	if err != nil {
		zlog.Logger.Error().Err(err).Str("image_id", id).Msg("failed to find image")
		return nil, fmt.Errorf("find image: %w", err)
	}

	if processedPath.Valid {
		img.ProcessedPath = processedPath.String
	}
	if errorMsg.Valid {
		img.ErrorMessage = errorMsg.String
	}
	if width.Valid {
		img.Width = int(width.Int32)
	}
	if height.Valid {
		img.Height = int(height.Int32)
	}
	if processedAt.Valid {
		img.ProcessedAt = &processedAt.Time
	}

	return &img, nil
}

func (r *imageRepository) Update(ctx context.Context, image *domain.Image) error {
	query := `
		UPDATE images
		SET original_filename = $2,
		    original_path = $3,
		    processed_path = $4,
		    mime_type = $5,
		    size = $6,
		    width = $7,
		    height = $8,
		    status = $9,
		    processing_type = $10,
		    error_message = $11,
		    processed_at = $12,
		    updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecWithRetry(ctx, r.strategy, query,
		image.ID,
		image.OriginalFilename,
		image.OriginalPath,
		nullString(image.ProcessedPath),
		image.MimeType,
		image.Size,
		nullInt(image.Width),
		nullInt(image.Height),
		image.Status,
		image.ProcessingType,
		nullString(image.ErrorMessage),
		image.ProcessedAt,
	)

	if err != nil {
		zlog.Logger.Error().Err(err).Str("image_id", image.ID).Msg("failed to update image")
		return fmt.Errorf("update image: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return domain.ErrImageNotFound
	}

	zlog.Logger.Info().Str("image_id", image.ID).Msg("image updated successfully")
	return nil
}

func (r *imageRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM images WHERE id = $1`

	result, err := r.db.ExecWithRetry(ctx, r.strategy, query, id)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("image_id", id).Msg("failed to delete image")
		return fmt.Errorf("delete image: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return domain.ErrImageNotFound
	}

	zlog.Logger.Info().Str("image_id", id).Msg("image deleted successfully")
	return nil
}

func (r *imageRepository) FindByStatus(ctx context.Context, status domain.ProcessingStatus, limit, offset int) ([]*domain.Image, error) {
	query := `
		SELECT id, original_filename, original_path, processed_path,
			   mime_type, size, width, height, status, processing_type,
			   error_message, created_at, updated_at, processed_at
		FROM images
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryWithRetry(ctx, r.strategy, query, status, limit, offset)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("status", string(status)).Msg("failed to find images by status")
		return nil, fmt.Errorf("find images by status: %w", err)
	}
	defer rows.Close()

	return r.scanImages(rows)
}

func (r *imageRepository) List(ctx context.Context, limit, offset int) ([]*domain.Image, error) {
	query := `
		SELECT id, original_filename, original_path, processed_path,
			   mime_type, size, width, height, status, processing_type,
			   error_message, created_at, updated_at, processed_at
		FROM images
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryWithRetry(ctx, r.strategy, query, limit, offset)
	if err != nil {
		zlog.Logger.Error().Err(err).Msg("failed to list images")
		return nil, fmt.Errorf("list images: %w", err)
	}
	defer rows.Close()

	return r.scanImages(rows)
}

func (r *imageRepository) UpdateStatus(ctx context.Context, id string, status domain.ProcessingStatus) error {
	query := `
		UPDATE images
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecWithRetry(ctx, r.strategy, query, id, status)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("image_id", id).Msg("failed to update status")
		return fmt.Errorf("update status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return domain.ErrImageNotFound
	}

	return nil
}

func (r *imageRepository) scanImages(rows *sql.Rows) ([]*domain.Image, error) {
	var images []*domain.Image

	for rows.Next() {
		var img domain.Image
		var processedPath, errorMsg sql.NullString
		var width, height sql.NullInt32
		var processedAt sql.NullTime

		err := rows.Scan(
			&img.ID,
			&img.OriginalFilename,
			&img.OriginalPath,
			&processedPath,
			&img.MimeType,
			&img.Size,
			&width,
			&height,
			&img.Status,
			&img.ProcessingType,
			&errorMsg,
			&img.CreatedAt,
			&img.UpdatedAt,
			&processedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan image: %w", err)
		}

		if processedPath.Valid {
			img.ProcessedPath = processedPath.String
		}
		if errorMsg.Valid {
			img.ErrorMessage = errorMsg.String
		}
		if width.Valid {
			img.Width = int(width.Int32)
		}
		if height.Valid {
			img.Height = int(height.Int32)
		}
		if processedAt.Valid {
			img.ProcessedAt = &processedAt.Time
		}

		images = append(images, &img)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return images, nil
}

// Helper functions
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt(i int) sql.NullInt32 {
	if i == 0 {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: int32(i), Valid: true}
}
