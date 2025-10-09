package repository

import (
	"context"
	"github.com/Evlushin/shorturl/internal/config"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/repository/file"
	"github.com/Evlushin/shorturl/internal/repository/inmemory"
	"github.com/Evlushin/shorturl/internal/repository/pg"
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

func NewRepository(cfg *config.Config) (Repository, error) {
	if cfg.DatabaseDsn != "" {
		return pg.NewStore(cfg)
	}

	if cfg.FileStorePath != "" {
		return file.NewStore(cfg)
	}

	return inmemory.NewStore(cfg)
}
