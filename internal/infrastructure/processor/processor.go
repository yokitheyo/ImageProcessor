package processor

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"

	"github.com/disintegration/imaging"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/config"
	"github.com/yokitheyo/imageprocessor/internal/domain"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
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
		Str("watermark_text", cfg.WatermarkText).
		Msg("ImageProcessor initialized")
	return &ImageProcessor{cfg: cfg}
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
	if p.cfg.WatermarkText == "" {
		zlog.Logger.Warn().Msg("Watermark text is empty, returning original image")
		return img
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	// КРАСНЫЙ цвет с прозрачностью
	opacity := uint8(float64(p.cfg.WatermarkOpacity) * 255.0 / 100.0)
	red := color.RGBA{255, 0, 0, opacity}

	face := basicfont.Face7x13

	// ОГРОМНЫЙ масштаб
	scale := 10 // каждая буква будет 10x10 пикселей вместо 1x1

	textLen := len(p.cfg.WatermarkText)
	scaledWidth := textLen * 7 * scale
	scaledHeight := 13 * scale

	// Расстояние между водяными знаками
	stepX := scaledWidth + 200
	stepY := scaledHeight + 150

	// Рисуем по всему изображению
	for row := -1; row*stepY < height+scaledHeight; row++ {
		for col := -1; col*stepX < width+scaledWidth; col++ {
			x := col * stepX
			y := row * stepY

			// Шахматный порядок
			if row%2 == 1 {
				x += stepX / 2
			}

			// Рисуем текст увеличенным
			drawLargeText(rgba, p.cfg.WatermarkText, x, y, scale, red, face)
		}
	}

	zlog.Logger.Info().
		Str("text", p.cfg.WatermarkText).
		Int("opacity", p.cfg.WatermarkOpacity).
		Int("scale", scale).
		Str("color", "RED").
		Msg("HUGE RED watermark applied")

	return rgba
}

func drawLargeText(dst *image.RGBA, text string, x, y, scale int, col color.Color, face font.Face) {
	tempWidth := len(text) * 10
	tempHeight := 20
	temp := image.NewRGBA(image.Rect(0, 0, tempWidth, tempHeight))

	drawer := &font.Drawer{
		Dst:  temp,
		Src:  image.NewUniform(color.White),
		Face: face,
		Dot:  fixed.Point26_6{X: 0, Y: fixed.Int26_6(13 * 64)},
	}
	drawer.DrawString(text)

	bounds := dst.Bounds()
	for sy := 0; sy < tempHeight; sy++ {
		for sx := 0; sx < tempWidth; sx++ {
			c := temp.At(sx, sy)
			if c != (color.RGBA{0, 0, 0, 0}) {
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						px := x + sx*scale + dx
						py := y + sy*scale + dy
						if px >= 0 && px < bounds.Dx() && py >= 0 && py < bounds.Dy() {
							dst.Set(px, py, col)
						}
					}
				}
			}
		}
	}
}

func GetImageDimensions(img image.Image) (width, height int) {
	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy()
}
