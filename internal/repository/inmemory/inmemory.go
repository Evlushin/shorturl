package inmemory

import (
	"context"
	"fmt"
	"github.com/Evlushin/shorturl/internal/config"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"github.com/Evlushin/shorturl/internal/repository"
	"sync"
)

type Store struct {
	mux *sync.RWMutex
	s   map[string]map[string]string
	cfg *config.Config
}

func NewStore(cfg *config.Config) (repository.Repository, error) {
	return &Store{
		mux: &sync.RWMutex{},
		s:   make(map[string]map[string]string),
		cfg: cfg,
	}, nil
}

func newErrGetShortenerNotFound(id string) error {
	return fmt.Errorf("%w for id = %s", myerrors.ErrGetShortenerNotFound, id)
}

func (s *Store) GetShortener(ctx context.Context, req *models.GetShortenerRequest) (*models.GetShortenerResponse, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	for _, shortMap := range s.s {
		if url, exists := shortMap[req.ID]; exists {
			return &models.GetShortenerResponse{
				URL: url,
			}, nil
		}
	}
	return nil, newErrGetShortenerNotFound(req.ID)
}

func (s *Store) GetShortenerUrls(ctx context.Context, userID string) ([]models.GetShortenerUrls, error) {
	res, ok := s.s[userID]
	if !ok {
		return nil, myerrors.ErrGetShortenerNotFound
	}

	var urls []models.GetShortenerUrls
	for id, v := range res {
		urls = append(urls, models.GetShortenerUrls{
			ID:  id,
			URL: v,
		})
	}

	return urls, nil
}

func (s *Store) SetShortener(ctx context.Context, req *models.SetShortenerRequest) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	var errUniqueURL error
	for key, v := range s.s[req.UserID] {
		if v == req.URL {
			req.ID = key
			errUniqueURL = myerrors.ErrConflictURL
		}
	}

	if s.s[req.UserID] == nil {
		s.s[req.UserID] = make(map[string]string)
	}

	s.s[req.UserID][req.ID] = req.URL

	return errUniqueURL
}

func (s *Store) SetShortenerBatch(ctx context.Context, req []models.SetShortenerBatchRequest) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	var errUniqueURL error
	for i, r := range req {
		for key, v := range s.s[r.UserID] {
			if v == r.URL {
				req[i].ID = key
				r.ID = key
				errUniqueURL = myerrors.ErrConflictURL
			}
		}
		if s.s[r.UserID] == nil {
			s.s[r.UserID] = make(map[string]string)
		}

		s.s[r.UserID][r.ID] = r.URL
	}

	return errUniqueURL
}

func (s *Store) Close() error {
	return nil
}

func (s *Store) Ping(ctx context.Context) error {
	return nil
}
