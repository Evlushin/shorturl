package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"github.com/Evlushin/shorturl/internal/repository"
	"net/url"
	"regexp"
	"strings"
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
	if err := getShortenerValidateRequest(req); err != nil {
		return nil, err
	}

	repositoryResp, err := f.store.GetShortener(ctx, &models.GetShortenerRequest{
		ID: req.ID,
	})
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

func getShortenerValidateRequest(req *models.GetShortenerRequest) error {
	validPattern := regexp.MustCompile(`^[A-Za-z0-9]{8}$`)
	if !validPattern.MatchString(req.ID) {
		return myerrors.ErrValidateShortenerInvalidRequest
	}

	return nil
}

func setShortenerValidateRequest(req *models.SetShortenerRequest) error {
	_, err := url.ParseRequestURI(req.URL)

	if err != nil {
		return fmt.Errorf("%w : URL : %s", myerrors.ErrValidateShortenerInvalidRequest, req.URL)
	}

	return nil
}

func setShortenerBatchValidateRequest(req []models.RequestBatch) error {
	notURL := make([]string, 0)
	for _, item := range req {
		err := setShortenerValidateRequest(&models.SetShortenerRequest{
			URL: item.OriginalURL,
		})

		if err != nil {
			notURL = append(notURL, item.OriginalURL)
		}
	}

	if len(notURL) > 0 {
		return fmt.Errorf("%w : URLs : %s", myerrors.ErrValidateShortenerInvalidRequest, strings.Join(notURL, ", "))
	}

	return nil
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

	err := setShortenerValidateRequest(req)
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

func (f *Shortener) SetShortenerBatch(ctx context.Context, req []models.RequestBatch) ([]models.SetShortenerBatchRequest, error) {
	if err := setShortenerBatchValidateRequest(req); err != nil {
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
		})
	}

	err := f.store.SetShortenerBatch(ctx, r)
	if err != nil && !errors.Is(err, myerrors.ErrConflictURL) {
		return nil, err
	}

	return r, err
}
