package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
)

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortUrl    string `json:"short_url"`
	OriginalUrl string `json:"original_url"`
}

type Repository interface {
	GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error)
	SetShortener(req *SetShortenerRequest) error
}

type Store struct {
	mux  *sync.Mutex
	s    map[string]string
	file string
}

func NewStore(fileStorePath string) (*Store, error) {
	store := &Store{
		mux:  &sync.Mutex{},
		s:    make(map[string]string),
		file: fileStorePath,
	}

	if err := store.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return store, nil
}

func (st *Store) load() error {
	st.mux.Lock()
	defer st.mux.Unlock()

	data, err := os.ReadFile(st.file)
	if err != nil {
		return err
	}

	var arr []URLRecord
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	st.s = make(map[string]string)
	for _, rec := range arr {
		st.s[rec.ShortUrl] = rec.OriginalUrl
	}

	return nil
}

func (st *Store) save() error {
	st.mux.Lock()
	defer st.mux.Unlock()

	arr := make([]URLRecord, 0, len(st.s))
	id := 1
	for shortURL, originalURL := range st.s {
		rec := URLRecord{
			UUID:        strconv.Itoa(id),
			ShortUrl:    shortURL,
			OriginalUrl: originalURL,
		}
		arr = append(arr, rec)
		id++
	}

	data, err := json.Marshal(arr)
	if err != nil {
		return err
	}
	return os.WriteFile(st.file, data, 0644)
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

func (st *Store) GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error) {
	st.mux.Lock()
	defer st.mux.Unlock()

	res, ok := st.s[req.ID]
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

func (st *Store) SetShortener(req *SetShortenerRequest) error {
	st.mux.Lock()
	st.s[req.ID] = req.URL
	st.mux.Unlock()

	return st.save()
}
