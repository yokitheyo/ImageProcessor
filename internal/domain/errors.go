package domain

import "errors"

var (
	ErrImageNotFound         = errors.New("image not found")
	ErrInvalidFormat         = errors.New("invalid or unsupported image format")
	ErrFileTooLarge          = errors.New("file size exceeds maximum allowed")
	ErrInvalidImageData      = errors.New("invalid image data")
	ErrProcessingFailed      = errors.New("image processing failed")
	ErrStorageFailed         = errors.New("storage operation failed")
	ErrQueueFailed           = errors.New("queue operation failed")
	ErrAlreadyProcessing     = errors.New("image is already being processed")
	ErrInvalidProcessingType = errors.New("invalid processing type")
)
