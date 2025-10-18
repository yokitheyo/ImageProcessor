package domain

import (
	"time"
)

type ProcessingStatus string

const (
	StatusPending    ProcessingStatus = "pending"
	StatusProcessing ProcessingStatus = "processing"
	StatusCompleted  ProcessingStatus = "completed"
	StatusFailed     ProcessingStatus = "failed"
)

type ProcessingType string

const (
	ProcessingResize    ProcessingType = "resize"
	ProcessingThumbnail ProcessingType = "thumbnail"
	ProcessingWatermark ProcessingType = "watermark"
)

type Image struct {
	ID               string           `json:"id"`
	OriginalFilename string           `json:"original_filename"`
	OriginalPath     string           `json:"original_path"`
	ProcessedPath    string           `json:"processed_path,omitempty"`
	MimeType         string           `json:"mime_type"`
	Size             int64            `json:"size"`
	Width            int              `json:"width,omitempty"`
	Height           int              `json:"height,omitempty"`
	Status           ProcessingStatus `json:"status"`
	ProcessingType   ProcessingType   `json:"processing_type"`
	ErrorMessage     string           `json:"error_message,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	ProcessedAt      *time.Time       `json:"processed_at,omitempty"`
}

func (i *Image) IsProcessed() bool {
	return i.Status == StatusCompleted
}

func (i *Image) IsFailed() bool {
	return i.Status == StatusFailed
}

func (i *Image) CanBeProcessed() bool {
	return i.Status == StatusPending || i.Status == StatusFailed
}

func (i *Image) MarkAsProcessing() {
	i.Status = StatusProcessing
	i.UpdatedAt = time.Now()
}

func (i *Image) MarkAsCompleted(processedPath string, width, height int) {
	i.Status = StatusCompleted
	i.ProcessedPath = processedPath
	i.Width = width
	i.Height = height
	now := time.Now()
	i.ProcessedAt = &now
	i.UpdatedAt = now
	i.ErrorMessage = ""
}

func (i *Image) MarkAsFailed(errMsg string) {
	i.Status = StatusFailed
	i.ErrorMessage = errMsg
	i.UpdatedAt = time.Now()
}
