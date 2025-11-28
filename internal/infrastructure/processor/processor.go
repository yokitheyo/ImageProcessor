package processor

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"math"

	"github.com/disintegration/imaging"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/config"
	"github.com/yokitheyo/imageprocessor/internal/domain"
)

type ImageProcessor struct {
	cfg          *config.ProcessingConfig
	watermarkImg image.Image
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
		Str("watermark_text", cfg.WatermarkText).
		Str("watermark_image", cfg.WatermarkImage).
		Msg("ImageProcessor initialized")
	p := &ImageProcessor{cfg: cfg}

	if cfg.WatermarkImage != "" {
		img, err := imaging.Open(cfg.WatermarkImage)
		if err != nil {
			zlog.Logger.Warn().Err(err).Str("watermark_image", cfg.WatermarkImage).Msg("failed to load watermark image, falling back to text watermarking")
		} else {
			p.watermarkImg = img
			zlog.Logger.Info().Int("watermark_img_width", img.Bounds().Dx()).Int("watermark_img_height", img.Bounds().Dy()).Msg("Loaded watermark image")
		}
	}

	return p
}

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
		Msg("Starting resize with aspect ratio preservation")

	resized := imaging.Fit(img, p.cfg.ResizeWidth, p.cfg.ResizeHeight, imaging.Lanczos)

	if resized.Bounds().Dx() == 0 || resized.Bounds().Dy() == 0 {
		zlog.Logger.Error().
			Int("resize_width", p.cfg.ResizeWidth).
			Int("resize_height", p.cfg.ResizeHeight).
			Msg("Resize produced empty image")
		return img
	}

	zlog.Logger.Info().
		Int("original_width", img.Bounds().Dx()).
		Int("original_height", img.Bounds().Dy()).
		Int("resized_width", resized.Bounds().Dx()).
		Int("resized_height", resized.Bounds().Dy()).
		Msg("Image resized successfully with aspect ratio preserved")

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
		Msg("Starting thumbnail creation with aspect ratio preservation")

	thumb := imaging.Fit(img, p.cfg.ThumbnailWidth, p.cfg.ThumbnailHeight, imaging.Lanczos)

	if thumb.Bounds().Dx() == 0 || thumb.Bounds().Dy() == 0 {
		zlog.Logger.Error().
			Int("thumbnail_width", p.cfg.ThumbnailWidth).
			Int("thumbnail_height", p.cfg.ThumbnailHeight).
			Msg("Thumbnail produced empty image")
		return img
	}

	zlog.Logger.Info().
		Int("original_width", img.Bounds().Dx()).
		Int("original_height", img.Bounds().Dy()).
		Int("thumbnail_width", thumb.Bounds().Dx()).
		Int("thumbnail_height", thumb.Bounds().Dy()).
		Msg("Thumbnail created successfully with aspect ratio preserved")

	return thumb
}

func (p *ImageProcessor) watermark(img image.Image) image.Image {
	if p.watermarkImg != nil {
		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		out := imaging.Clone(img)

		wm := p.watermarkImg
		wmBounds := wm.Bounds()
		wmW := wmBounds.Dx()
		wmH := wmBounds.Dy()

		if wmW == 0 || wmH == 0 {
			zlog.Logger.Warn().Msg("watermark image has zero size, returning original image")
			return img
		}

		opacity := float64(p.cfg.WatermarkOpacity) / 255.0
		if opacity < 0 {
			opacity = 0
		}
		if opacity > 1 {
			opacity = 1
		}

		targetWidth := width / 4
		if targetWidth < 10 {
			targetWidth = 10
		}
		wmScaled := imaging.Resize(wm, targetWidth, 0, imaging.Lanczos)

		wmRot := imaging.Rotate(wmScaled, -45, color.NRGBA{0, 0, 0, 0})
		rotW := wmRot.Bounds().Dx()
		rotH := wmRot.Bounds().Dy()

		diagLen := int(math.Hypot(float64(width), float64(height))) + rotW
		spacing := rotW/2 + 20
		if spacing < 10 {
			spacing = 10
		}
		step := rotW + spacing
		count := diagLen/step + 2
		if count < 1 {
			count = 1
		}

		for i := 0; i <= count; i++ {
			t := float64(i) / float64(count)
			posX := int((1.0-t)*float64(-rotW) + t*float64(width))
			posY := int((1.0-t)*float64(-rotH) + t*float64(height))
			out = imaging.Overlay(out, wmRot, image.Pt(posX, posY), opacity)
		}

		zlog.Logger.Info().Str("watermark", p.cfg.WatermarkImage).Int("opacity", p.cfg.WatermarkOpacity).Msg("Image watermark applied (diagonal image-only)")

		return out
	}

	zlog.Logger.Warn().Msg("No image watermark configured — image watermarking is required. Returning original image (no text watermark)")
	return img
}

func GetImageDimensions(img image.Image) (width, height int) {
	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy()
}
