package file

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Evlushin/shorturl/internal/config"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"github.com/Evlushin/shorturl/internal/repository"
	"os"
	"strconv"
	"sync"
)

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Store struct {
	mux *sync.Mutex
	s   map[string]string
	cfg *config.Config
}

func NewStore(cfg *config.Config) (repository.Repository, error) {
	store := &Store{
		mux: &sync.Mutex{},
		s:   make(map[string]string),
		cfg: cfg,
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

	data, err := os.ReadFile(st.cfg.FileStorePath)
	if err != nil {
		return err
	}

	var arr []URLRecord
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	st.s = make(map[string]string)
	for _, rec := range arr {
		st.s[rec.ShortURL] = rec.OriginalURL
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
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		}
		arr = append(arr, rec)
		id++
	}

	data, err := json.Marshal(arr)
	if err != nil {
		return err
	}
	return os.WriteFile(st.cfg.FileStorePath, data, 0644)
}

func newErrGetShortenerNotFound(id string) error {
	return fmt.Errorf("%w for id = %s", myerrors.ErrGetShortenerNotFound, id)
}

func (st *Store) GetShortener(ctx context.Context, req *models.GetShortenerRequest) (*models.GetShortenerResponse, error) {
	st.mux.Lock()
	defer st.mux.Unlock()

	res, ok := st.s[req.ID]
	if !ok {
		return nil, newErrGetShortenerNotFound(req.ID)
	}
	return &models.GetShortenerResponse{
		URL: res,
	}, nil
}

func (st *Store) SetShortener(ctx context.Context, req *models.SetShortenerRequest) error {
	st.mux.Lock()
	st.s[req.ID] = req.URL
	st.mux.Unlock()

	return st.save()
}

func (st *Store) SetShortenerBatch(ctx context.Context, req []models.SetShortenerBatchRequest) error {
	st.mux.Lock()
	for _, r := range req {
		st.s[r.ID] = r.URL
	}
	st.mux.Unlock()

	return st.save()
}

func (st *Store) Close() error {
	return nil
}

func (st *Store) Ping(ctx context.Context) error {
	return nil
}
