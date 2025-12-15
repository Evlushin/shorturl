package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/Evlushin/shorturl/internal/config"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"github.com/Evlushin/shorturl/internal/repository"
)

// generate:reset
type Store struct {
	mux *sync.RWMutex
	s   map[string]map[string]models.GetShortenerResponse
	cfg *config.Config
}

func NewStore(cfg *config.Config) (repository.Repository, error) {
	return &Store{
		mux: &sync.RWMutex{},
		s:   make(map[string]map[string]models.GetShortenerResponse),
		cfg: cfg,
	}, nil
}

func newErrGetShortenerNotFound(id string) error {
	return fmt.Errorf("%w for id = %s", myerrors.ErrGetShortenerNotFound, id)
}

func (s *Store) DeleteShortenerUrls(ctx context.Context, req []models.RequestIDBatch, userID string) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	res, ok := s.s[userID]
	if !ok {
		return myerrors.ErrGetShortenerNotFound
	}

	for _, id := range req {
		_, ok = res[string(id)]
		if ok {
			item := s.s[userID][string(id)]
			item.IsDeleted = true
			s.s[userID][string(id)] = item
		}
	}

	return nil
}

func (s *Store) GetShortener(ctx context.Context, req *models.GetShortenerRequest) (*models.GetShortenerResponse, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	for _, shortMap := range s.s {
		if url, exists := shortMap[req.ID]; exists {
			return &url, nil
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
		if !v.IsDeleted {
			urls = append(urls, models.GetShortenerUrls{
				ID:  id,
				URL: v.URL,
			})
		}
	}

	return urls, nil
}

func (s *Store) SetShortener(ctx context.Context, req *models.SetShortenerRequest) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	var errUniqueURL error
	for key, v := range s.s[req.UserID] {
		if v.URL == req.URL {
			req.ID = key
			errUniqueURL = myerrors.ErrConflictURL
		}
	}

	if s.s[req.UserID] == nil {
		s.s[req.UserID] = make(map[string]models.GetShortenerResponse)
	}

	s.s[req.UserID][req.ID] = models.GetShortenerResponse{
		URL:       req.URL,
		IsDeleted: false,
	}

	return errUniqueURL
}

func (s *Store) SetShortenerBatch(ctx context.Context, req []models.SetShortenerBatchRequest) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	var errUniqueURL error
	for i, r := range req {
		for key, v := range s.s[r.UserID] {
			if v.URL == r.URL {
				req[i].ID = key
				r.ID = key
				errUniqueURL = myerrors.ErrConflictURL
			}
		}
		if s.s[r.UserID] == nil {
			s.s[r.UserID] = make(map[string]models.GetShortenerResponse)
		}

		s.s[r.UserID][r.ID] = models.GetShortenerResponse{
			URL:       r.URL,
			IsDeleted: false,
		}
	}

	return errUniqueURL
}

func (s *Store) Close() error {
	return nil
}

func (s *Store) Ping(ctx context.Context) error {
	return nil
}
