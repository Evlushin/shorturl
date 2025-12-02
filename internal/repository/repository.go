package repository

import (
	"context"

	"github.com/Evlushin/shorturl/internal/models"
)

type Repository interface {
	GetShortener(ctx context.Context, req *models.GetShortenerRequest) (*models.GetShortenerResponse, error)
	SetShortener(ctx context.Context, req *models.SetShortenerRequest) error
	SetShortenerBatch(ctx context.Context, req []models.SetShortenerBatchRequest) error
	GetShortenerUrls(ctx context.Context, userID string) ([]models.GetShortenerUrls, error)
	DeleteShortenerUrls(ctx context.Context, req []models.RequestIDBatch, userID string) error
	Close() error
	Ping(ctx context.Context) error
}
