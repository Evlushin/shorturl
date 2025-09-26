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
	s   map[string]string
	cfg *config.Config
}

func NewStore(cfg *config.Config) (repository.Repository, error) {
	return &Store{
		mux: &sync.RWMutex{},
		s:   make(map[string]string),
		cfg: cfg,
	}, nil
}

func newErrGetShortenerNotFound(id string) error {
	return fmt.Errorf("%w for id = %s", myerrors.ErrGetShortenerNotFound, id)
}

func (s *Store) GetShortener(ctx context.Context, req *models.GetShortenerRequest) (*models.GetShortenerResponse, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	res, ok := s.s[req.ID]
	if !ok {
		return nil, newErrGetShortenerNotFound(req.ID)
	}
	return &models.GetShortenerResponse{
		URL: res,
	}, nil
}

func (s *Store) SetShortener(ctx context.Context, req *models.SetShortenerRequest) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	var errUniqueURL error
	for key, v := range s.s {
		if v == req.URL {
			req.ID = key
			errUniqueURL = myerrors.ErrConflictURL
		}
	}

	s.s[req.ID] = req.URL

	return errUniqueURL
}

func (s *Store) SetShortenerBatch(ctx context.Context, req []models.SetShortenerBatchRequest) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	var errUniqueURL error
	for i, r := range req {
		for key, v := range s.s {
			if v == r.URL {
				req[i].ID = key
				r.ID = key
				errUniqueURL = myerrors.ErrConflictURL
			}
		}
		s.s[r.ID] = r.URL
	}

	return errUniqueURL
}

func (s *Store) Close() error {
	return nil
}

func (s *Store) Ping(ctx context.Context) error {
	return nil
}
