package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"github.com/Evlushin/shorturl/internal/repository"
)

type Shortener struct {
	store repository.Repository
}

func NewShortener(store repository.Repository) *Shortener {
	return &Shortener{
		store: store,
	}
}

func (f *Shortener) Ping(ctx context.Context) error {
	return f.store.Ping(ctx)
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

	if repositoryResp != nil {
		return &models.GetShortenerResponse{
			URL: repositoryResp.URL,
		}, nil
	}

	return nil, fmt.Errorf("not found: %w", err)
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

func (f *Shortener) generateRandomString(ctx context.Context, length uint8, limit uint16) (string, error) {
	if limit <= 0 {
		return "", myerrors.ErrEndRandomStrings
	}

	const (
		charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	)

	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	result := make([]byte, length)
	for i := uint8(0); i < length; i++ {
		result[i] = charset[int(b[i])%len(charset)]
	}

	id := string(result)
	_, err := f.store.GetShortener(ctx, &models.GetShortenerRequest{
		ID: id,
	})
	if err != nil {
		if errors.Is(err, myerrors.ErrGetShortenerNotFound) {
			return id, nil
		}

		return "", err
	}

	return f.generateRandomString(ctx, length, limit-1)
}

func (f *Shortener) SetShortener(ctx context.Context, req *models.SetShortenerRequest) (*models.SetShortenerResponse, error) {

	err := SetShortenerValidateRequest(req)
	if err != nil {
		return nil, err
	}

	req.ID, err = f.generateRandomString(ctx, 8, 10000)
	if err != nil {
		return nil, err
	}

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
		id, err := f.generateRandomString(ctx, 8, 100)

		if err != nil {
			return nil, err
		}

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
