package repository

import (
	"errors"
	"fmt"
	"sync"
)

type Repository interface {
	GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error)
	SetShortener(req *SetShortenerRequest)
}

type Store struct {
	mux *sync.Mutex
	s   map[string]string
}

func NewStore() *Store {
	return &Store{
		mux: &sync.Mutex{},
		s:   make(map[string]string),
	}
}

type GetShortenerRequest struct {
	ID string
}

type GetShortenerResponse struct {
	URL string
}

var (
	ErrGetShortenerNotFound = errors.New("no url")
)

func newErrGetShortenerNotFound(id string) error {
	return fmt.Errorf("%w for id = %s", ErrGetShortenerNotFound, id)
}

func (s *Store) GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	res, ok := s.s[req.ID]
	if !ok {
		return nil, newErrGetShortenerNotFound(req.ID)
	}
	return &GetShortenerResponse{
		URL: res,
	}, nil
}

type SetShortenerRequest struct {
	ID  string
	URL string
}

func (s *Store) SetShortener(req *SetShortenerRequest) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.s[req.ID] = req.URL
}
