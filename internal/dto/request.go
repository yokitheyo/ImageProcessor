package dto

import "github.com/yokitheyo/wb_level_3_04/internal/domain"

type UploadImageRequest struct {
	ProcessingType string `form:"processing_type" binding:"required,oneof=resize thumbnail watermark"`
}

func (r *UploadImageRequest) ToProcessingType() domain.ProcessingType {
	return domain.ProcessingType(r.ProcessingType)
}

type ProcessImageRequest struct {
	ImageID        string `json:"image_id"`
	ProcessingType string `json:"processing_type"`
}
