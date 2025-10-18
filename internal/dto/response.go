package dto

import (
	"time"

	"github.com/yokitheyo/imageprocessor/internal/domain"
)

type ImageResponse struct {
	ID               string     `json:"id"`
	OriginalFilename string     `json:"original_filename"`
	MimeType         string     `json:"mime_type"`
	Size             int64      `json:"size"`
	Width            int        `json:"width,omitempty"`
	Height           int        `json:"height,omitempty"`
	Status           string     `json:"status"`
	ProcessingType   string     `json:"processing_type"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ProcessedAt      *time.Time `json:"processed_at,omitempty"`

	// URLs
	OriginalURL  string `json:"original_url"`
	ProcessedURL string `json:"processed_url,omitempty"`
}

type ImageListResponse struct {
	Images []*ImageResponse `json:"images"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

func MapImageToResponse(img *domain.Image, baseURL string) *ImageResponse {
	if img == nil {
		return nil
	}

	resp := &ImageResponse{
		ID:               img.ID,
		OriginalFilename: img.OriginalFilename,
		MimeType:         img.MimeType,
		Size:             img.Size,
		Width:            img.Width,
		Height:           img.Height,
		Status:           string(img.Status),
		ProcessingType:   string(img.ProcessingType),
		ErrorMessage:     img.ErrorMessage,
		CreatedAt:        img.CreatedAt,
		UpdatedAt:        img.UpdatedAt,
		ProcessedAt:      img.ProcessedAt,
		OriginalURL:      baseURL + "/image/" + img.ID + "/original",
	}

	if img.IsProcessed() {
		resp.ProcessedURL = baseURL + "/image/" + img.ID
	}

	return resp
}

func MapImagesToResponse(images []*domain.Image, baseURL string, limit, offset int) *ImageListResponse {
	responses := make([]*ImageResponse, 0, len(images))
	for _, img := range images {
		responses = append(responses, MapImageToResponse(img, baseURL))
	}

	return &ImageListResponse{
		Images: responses,
		Total:  len(responses),
		Limit:  limit,
		Offset: offset,
	}
}
