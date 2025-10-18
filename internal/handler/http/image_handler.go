package http

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/domain"
	"github.com/yokitheyo/imageprocessor/internal/dto"
)

type ImageHandler struct {
	service        domain.ImageService
	maxUploadSize  int64
	allowedFormats []string
}

func NewImageHandler(service domain.ImageService, maxUploadSizeMB int, allowedFormats []string) *ImageHandler {
	return &ImageHandler{
		service:        service,
		maxUploadSize:  int64(maxUploadSizeMB) * 1024 * 1024,
		allowedFormats: allowedFormats,
	}
}

func (h *ImageHandler) RegisterRoutes(engine *ginext.Engine) {
	engine.POST("/upload", h.UploadImage)
	engine.GET("/image/:id", h.GetProcessedImage)
	engine.GET("/image/:id/original", h.GetOriginalImage)
	engine.DELETE("/image/:id", h.DeleteImage)
	engine.GET("/images", h.ListImages)
}

// UploadImage POST /upload
func (h *ImageHandler) UploadImage(c *ginext.Context) {
	// Получаем файл из формы
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		zlog.Logger.Warn().Err(err).Msg("failed to get file from request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "No image file provided",
		})
		return
	}
	defer file.Close()

	if header.Size > h.maxUploadSize {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "file_too_large",
			Message: fmt.Sprintf("File size exceeds maximum allowed (%d MB)", h.maxUploadSize/(1024*1024)),
		})
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !h.isAllowedFormat(ext) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_format",
			Message: fmt.Sprintf("Unsupported file format. Allowed: %v", h.allowedFormats),
		})
		return
	}

	processingType := c.PostForm("processing_type")
	if processingType == "" {
		processingType = "resize"
	}

	var pt domain.ProcessingType
	switch processingType {
	case "resize":
		pt = domain.ProcessingResize
	case "thumbnail":
		pt = domain.ProcessingThumbnail
	case "watermark":
		pt = domain.ProcessingWatermark
	default:
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_processing_type",
			Message: "Processing type must be one of: resize, thumbnail, watermark",
		})
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	image, err := h.service.UploadImage(
		c.Request.Context(),
		header.Filename,
		mimeType,
		header.Size,
		file,
		pt,
	)

	if err != nil {
		zlog.Logger.Error().Err(err).Msg("failed to upload image")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "upload_failed",
			Message: "Failed to upload image",
		})
		return
	}

	baseURL := h.getBaseURL(c)
	response := dto.MapImageToResponse(image, baseURL)

	c.JSON(http.StatusCreated, response)
}

// GetProcessedImage GET /image/:id
func (h *ImageHandler) GetProcessedImage(c *ginext.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "Image ID is required",
		})
		return
	}

	file, filename, err := h.service.GetImageFile(c.Request.Context(), id, false)
	if err != nil {
		if err == domain.ErrImageNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: "Image not found",
			})
			return
		}
		zlog.Logger.Error().Err(err).Str("image_id", id).Msg("failed to get processed image")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to retrieve image",
		})
		return
	}
	defer file.Close()

	contentType := h.getContentType(filename)

	if stat, err := file.(interface{ Stat() (os.FileInfo, error) }).Stat(); err == nil {
		c.Header("Content-Length", strconv.FormatInt(stat.Size(), 10))
	} else {
		zlog.Logger.Warn().Err(err).Str("image_id", id).Str("filename", filename).Msg("failed to get file size")
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", filename))

	written, err := io.Copy(c.Writer, file)
	if err != nil {
		zlog.Logger.Error().
			Err(err).
			Str("image_id", id).
			Str("filename", filename).
			Int64("bytes_written", written).
			Msg("failed to write image to response")
		return
	}
	zlog.Logger.Info().
		Str("image_id", id).
		Str("filename", filename).
		Int64("bytes_written", written).
		Msg("processed image sent successfully")
}

// GetOriginalImage GET /image/:id/original
func (h *ImageHandler) GetOriginalImage(c *ginext.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "Image ID is required",
		})
		return
	}

	file, filename, err := h.service.GetImageFile(c.Request.Context(), id, true)
	if err != nil {
		if err == domain.ErrImageNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: "Image not found",
			})
			return
		}
		zlog.Logger.Error().Err(err).Str("image_id", id).Msg("failed to get original image")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to retrieve image",
		})
		return
	}
	defer file.Close()

	contentType := h.getContentType(filename)

	if stat, err := file.(interface{ Stat() (os.FileInfo, error) }).Stat(); err == nil {
		c.Header("Content-Length", strconv.FormatInt(stat.Size(), 10))
	} else {
		zlog.Logger.Warn().Err(err).Str("image_id", id).Str("filename", filename).Msg("failed to get file size")
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", filename))

	written, err := io.Copy(c.Writer, file)
	if err != nil {
		zlog.Logger.Error().
			Err(err).
			Str("image_id", id).
			Str("filename", filename).
			Int64("bytes_written", written).
			Msg("failed to write original image to response")
		return
	}
	zlog.Logger.Info().
		Str("image_id", id).
		Str("filename", filename).
		Int64("bytes_written", written).
		Msg("original image sent successfully")
}

// DeleteImage DELETE /image/:id
func (h *ImageHandler) DeleteImage(c *ginext.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "Image ID is required",
		})
		return
	}

	if err := h.service.DeleteImage(c.Request.Context(), id); err != nil {
		if err == domain.ErrImageNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: "Image not found",
			})
			return
		}
		zlog.Logger.Error().Err(err).Str("image_id", id).Msg("failed to delete image")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to delete image",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListImages GET /images
func (h *ImageHandler) ListImages(c *ginext.Context) {
	// Получаем параметры пагинации
	limit := 10
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	images, err := h.service.ListImages(c.Request.Context(), limit, offset)
	if err != nil {
		zlog.Logger.Error().Err(err).Msg("failed to list images")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to retrieve images",
		})
		return
	}

	baseURL := h.getBaseURL(c)
	response := dto.MapImagesToResponse(images, baseURL, limit, offset)

	c.JSON(http.StatusOK, response)
}

// Helper methods

func (h *ImageHandler) isAllowedFormat(ext string) bool {
	ext = strings.TrimPrefix(ext, ".")
	for _, allowed := range h.allowedFormats {
		if strings.EqualFold(ext, allowed) {
			return true
		}
	}
	return false
}

func (h *ImageHandler) getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

func (h *ImageHandler) getBaseURL(c *ginext.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}
