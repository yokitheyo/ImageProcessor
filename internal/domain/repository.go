package domain

import "context"

type ImageRepository interface {
	Create(ctx context.Context, image *Image) error
	FindByID(ctx context.Context, id string) (*Image, error)
	Update(ctx context.Context, image *Image) error
	Delete(ctx context.Context, id string) error
	FindByStatus(ctx context.Context, status ProcessingStatus, limit, offset int) ([]*Image, error)
	List(ctx context.Context, limit, offset int) ([]*Image, error)
	UpdateStatus(ctx context.Context, id string, status ProcessingStatus) error
}
