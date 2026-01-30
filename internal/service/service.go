package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Evlushin/shorturl/internal/config"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"github.com/Evlushin/shorturl/internal/repository"
	"github.com/Evlushin/shorturl/internal/repository/file"
	"github.com/Evlushin/shorturl/internal/repository/inmemory"
	"github.com/Evlushin/shorturl/internal/repository/pg"
	"github.com/google/uuid"
)

type Shortener struct {
	store repository.Repository
}

func NewShortener(cfg config.Config) (*Shortener, error) {
	store, err := NewRepository(&cfg)
	if err != nil {
		return nil, err
	}

	return &Shortener{
		store: store,
	}, nil
}

func NewRepository(cfg *config.Config) (repository.Repository, error) {
	if cfg.DatabaseDsn != "" {
		return pg.NewStore(cfg)
	}

	if cfg.FileStorePath != "" {
		return file.NewStore(cfg)
	}

	return inmemory.NewStore(cfg)
}

func (f *Shortener) Close() error {
	return f.store.Close()
}

func (f *Shortener) Ping(ctx context.Context) error {
	return f.store.Ping(ctx)
}

func (f *Shortener) GetStats(ctx context.Context) (*models.ResponseStats, error) {
	return f.store.GetStats(ctx)
}

func (f *Shortener) GetShortener(ctx context.Context, req *models.GetShortenerRequest) (*models.GetShortenerResponse, error) {
	if err := GetShortenerValidateRequest(req); err != nil {
		return nil, err
	}

	repositoryResp, err := f.store.GetShortener(ctx, req)
	if err != nil {
		if !errors.Is(err, myerrors.ErrGetShortenerNotFound) {
			return nil, fmt.Errorf("failed to fetch the shortener result from the store: %w", err)
		}

		return nil, fmt.Errorf("not found: %w", err)
	}

	if repositoryResp == nil {
		return nil, fmt.Errorf("not found: %w", err)
	}

	if repositoryResp.IsDeleted {
		return repositoryResp, myerrors.ErrGone410
	}

	return repositoryResp, nil
}

func (f *Shortener) DeleteShortenerUrls(ctx context.Context, req []models.RequestIDBatch, userID string) error {
	return f.store.DeleteShortenerUrls(ctx, req, userID)
}

func (f *Shortener) GetShortenerUrls(ctx context.Context, userID string) ([]models.GetShortenerUrls, error) {

	repositoryResp, err := f.store.GetShortenerUrls(ctx, userID)
	if err != nil {
		if !errors.Is(err, myerrors.ErrGetShortenerNotFound) {
			return nil, fmt.Errorf("failed to fetch the shortener result from the store: %w", err)
		}

		return nil, myerrors.ErrGetShortenerNotFound
	}

	if repositoryResp != nil {
		return repositoryResp, nil
	}

	return nil, myerrors.ErrGetShortenerNotFound
}

func (f *Shortener) generateRandomString() string {
	u := uuid.New()
	s := u.String()
	var builder strings.Builder
	builder.Grow(len(s) - 4)

	for i := 0; i < len(s); i++ {
		if s[i] != '-' {
			builder.WriteByte(s[i])
		}
	}
	return builder.String()
}

func (f *Shortener) SetShortener(ctx context.Context, req *models.SetShortenerRequest) (*models.SetShortenerResponse, error) {

	err := SetShortenerValidateRequest(req)
	if err != nil {
		return nil, err
	}

	req.ID = f.generateRandomString()

	err = f.store.SetShortener(ctx, req)
	if err != nil && !errors.Is(err, myerrors.ErrConflictURL) {
		return nil, err
	}

	return &models.SetShortenerResponse{
		ID: req.ID,
	}, err
}

func (f *Shortener) SetShortenerBatch(ctx context.Context, req []models.RequestBatch, userID string) ([]models.SetShortenerBatchRequest, error) {
	if err := SetShortenerBatchValidateRequest(req); err != nil {
		return nil, err
	}

	var r []models.SetShortenerBatchRequest
	for _, item := range req {
		id := f.generateRandomString()

		r = append(r, models.SetShortenerBatchRequest{
			CorrelationID: item.CorrelationID,
			ID:            id,
			URL:           item.OriginalURL,
			UserID:        userID,
		})
	}

	err := f.store.SetShortenerBatch(ctx, r)
	if err != nil && !errors.Is(err, myerrors.ErrConflictURL) {
		return nil, err
	}

	return r, err
}
