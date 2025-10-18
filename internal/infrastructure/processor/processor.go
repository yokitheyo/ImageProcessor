package processor

import (
	"fmt"
	"image"
	"io"

	"github.com/disintegration/imaging"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/wb_level_3_04/internal/config"
	"github.com/yokitheyo/wb_level_3_04/internal/domain"
)

type ImageProcessor struct {
	cfg *config.ProcessingConfig
}

func NewImageProcessor(cfg *config.ProcessingConfig) *ImageProcessor {
	if cfg.ResizeWidth <= 0 || cfg.ResizeHeight <= 0 {
		zlog.Logger.Warn().
			Int("resize_width", cfg.ResizeWidth).
			Int("resize_height", cfg.ResizeHeight).
			Msg("Invalid resize dimensions, using defaults")
		cfg.ResizeWidth = 800
		cfg.ResizeHeight = 600
	}
	if cfg.ThumbnailWidth <= 0 || cfg.ThumbnailHeight <= 0 {
		zlog.Logger.Warn().
			Int("thumbnail_width", cfg.ThumbnailWidth).
			Int("thumbnail_height", cfg.ThumbnailHeight).
			Msg("Invalid thumbnail dimensions, using defaults")
		cfg.ThumbnailWidth = 200
		cfg.ThumbnailHeight = 150
	}
	zlog.Logger.Info().
		Int("resize_width", cfg.ResizeWidth).
		Int("resize_height", cfg.ResizeHeight).
		Int("thumbnail_width", cfg.ThumbnailWidth).
		Int("thumbnail_height", cfg.ThumbnailHeight).
		Int("output_quality", cfg.OutputQuality).
		Msg("ImageProcessor initialized")
	return &ImageProcessor{cfg: cfg}
}

// Геттеры
func (p *ImageProcessor) ResizeWidth() int {
	return p.cfg.ResizeWidth
}

func (p *ImageProcessor) ResizeHeight() int {
	return p.cfg.ResizeHeight
}

func (p *ImageProcessor) ThumbnailWidth() int {
	return p.cfg.ThumbnailWidth
}

func (p *ImageProcessor) ThumbnailHeight() int {
	return p.cfg.ThumbnailHeight
}

func (p *ImageProcessor) Process(r io.Reader, processingType domain.ProcessingType) (image.Image, error) {
	img, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		zlog.Logger.Error().Err(err).Msg("failed to decode image")
		return nil, fmt.Errorf("decode image: %w", err)
	}
	if img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
		zlog.Logger.Error().Msg("decoded image is empty")
		return nil, fmt.Errorf("decoded image is empty")
	}
	zlog.Logger.Info().
		Int("width", img.Bounds().Dx()).
		Int("height", img.Bounds().Dy()).
		Str("processing_type", string(processingType)).
		Msg("Image decoded successfully")
	switch processingType {
	case domain.ProcessingResize:
		return p.resize(img), nil
	case domain.ProcessingThumbnail:
		return p.thumbnail(img), nil
	case domain.ProcessingWatermark:
		return p.watermark(img), nil
	default:
		zlog.Logger.Error().Str("processing_type", string(processingType)).Msg("unknown processing type")
		return nil, fmt.Errorf("unknown processing type: %v", processingType)
	}
}

func (p *ImageProcessor) resize(img image.Image) image.Image {
	if p.cfg.ResizeWidth <= 0 || p.cfg.ResizeHeight <= 0 {
		zlog.Logger.Warn().
			Int("resize_width", p.cfg.ResizeWidth).
			Int("resize_height", p.cfg.ResizeHeight).
			Msg("Resize dimensions are invalid, returning original image")
		return img
	}
	zlog.Logger.Info().
		Int("resize_width", p.cfg.ResizeWidth).
		Int("resize_height", p.cfg.ResizeHeight).
		Msg("Starting resize")
	resized := imaging.Resize(img, p.cfg.ResizeWidth, p.cfg.ResizeHeight, imaging.Lanczos)
	if resized.Bounds().Dx() == 0 || resized.Bounds().Dy() == 0 {
		zlog.Logger.Error().
			Int("resize_width", p.cfg.ResizeWidth).
			Int("resize_height", p.cfg.ResizeHeight).
			Msg("Resize produced empty image")
		return img
	}
	zlog.Logger.Info().
		Int("width", resized.Bounds().Dx()).
		Int("height", resized.Bounds().Dy()).
		Msg("Image resized successfully")
	return resized
}

func (p *ImageProcessor) thumbnail(img image.Image) image.Image {
	if p.cfg.ThumbnailWidth <= 0 || p.cfg.ThumbnailHeight <= 0 {
		zlog.Logger.Warn().
			Int("thumbnail_width", p.cfg.ThumbnailWidth).
			Int("thumbnail_height", p.cfg.ThumbnailHeight).
			Msg("Thumbnail dimensions are invalid, returning original image")
		return img
	}
	zlog.Logger.Info().
		Int("thumbnail_width", p.cfg.ThumbnailWidth).
		Int("thumbnail_height", p.cfg.ThumbnailHeight).
		Msg("Starting thumbnail")
	thumb := imaging.Fit(img, p.cfg.ThumbnailWidth, p.cfg.ThumbnailHeight, imaging.Lanczos)
	if thumb.Bounds().Dx() == 0 || thumb.Bounds().Dy() == 0 {
		zlog.Logger.Error().
			Int("thumbnail_width", p.cfg.ThumbnailWidth).
			Int("thumbnail_height", p.cfg.ThumbnailHeight).
			Msg("Thumbnail produced empty image")
		return img
	}
	zlog.Logger.Info().
		Int("width", thumb.Bounds().Dx()).
		Int("height", thumb.Bounds().Dy()).
		Msg("Thumbnail created successfully")
	return thumb
}

func (p *ImageProcessor) watermark(img image.Image) image.Image {
	zlog.Logger.Warn().Msg("Watermark not implemented, returning original image")
	return img
}

func GetImageDimensions(img image.Image) (width, height int) {
	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy()
}
