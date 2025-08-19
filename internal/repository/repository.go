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
	Id string
}

type GetShortenerResponse struct {
	Url string
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

	res, ok := s.s[req.Id]
	if !ok {
		return nil, newErrGetShortenerNotFound(req.Id)
	}
	return &GetShortenerResponse{
		Url: res,
	}, nil
}

type SetShortenerRequest struct {
	Id  string
	Url string
}

func (s *Store) SetShortener(req *SetShortenerRequest) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.s[req.Id] = req.Url
}
