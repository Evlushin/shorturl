package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/Evlushin/shorturl/internal/repository"
	"net/url"
	"regexp"
)

type Repository interface {
	GetShortener(req *repository.GetShortenerRequest) (*repository.GetShortenerResponse, error)
	SetShortener(req *repository.SetShortenerRequest)
}

type Shortener struct {
	store Repository
}

func NewShortener(store Repository) *Shortener {
	return &Shortener{
		store: store,
	}
}

type GetShortenerRequest struct {
	ID string
}

type GetShortenerResponse struct {
	URL string
}

type SetShortenerRequest struct {
	URL string
}

type SetShortenerResponse struct {
	ID string
}

var (
	ErrGetShortenerInvalidRequest      = errors.New("invalid get shortener request")
	ErrValidateShortenerInvalidRequest = errors.New("invalid validate shortener request")
	ErrRepoFailed                      = errors.New("repo failed")
)

func (f *Shortener) GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error) {
	if err := getShortenerValidateRequest(req); err != nil {
		return nil, err
	}

	repositoryResp, err := f.store.GetShortener(&repository.GetShortenerRequest{
		ID: req.ID,
	})
	if err != nil {
		if !errors.Is(err, repository.ErrGetShortenerNotFound) {
			return nil, fmt.Errorf("failed to fetch the shortener result from the store: %w", err)
		}

		return nil, fmt.Errorf("not found: %w", err)
	}

	if repositoryResp != nil {
		return &GetShortenerResponse{
			URL: repositoryResp.URL,
		}, nil
	}

	return nil, fmt.Errorf("not found: %w", err)
}

func getShortenerValidateRequest(req *GetShortenerRequest) error {
	validPattern := regexp.MustCompile(`^[A-Za-z0-9]{8}$`)
	if !validPattern.MatchString(req.ID) {
		return ErrValidateShortenerInvalidRequest
	}

	return nil
}

func setShortenerValidateRequest(req *SetShortenerRequest) error {
	_, err := url.ParseRequestURI(req.URL)

	if err != nil {
		return ErrValidateShortenerInvalidRequest
	}

	return nil
}

func (f *Shortener) generateRandomString(length uint8) (string, error) {
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
	_, err := f.store.GetShortener(&repository.GetShortenerRequest{
		ID: id,
	})
	if err != nil {
		if errors.Is(err, repository.ErrGetShortenerNotFound) {
			return id, nil
		}

		return "", err
	}

	return f.generateRandomString(length)
}

func (f *Shortener) SetShortener(req *SetShortenerRequest) (*SetShortenerResponse, error) {
	if err := setShortenerValidateRequest(req); err != nil {
		return nil, err
	}

	id, err := f.generateRandomString(8)
	if err != nil {
		return nil, err
	}

	f.store.SetShortener(&repository.SetShortenerRequest{
		ID:  id,
		URL: req.URL,
	})

	return &SetShortenerResponse{
		ID: id,
	}, nil
}
