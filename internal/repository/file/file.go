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
	UserID      string `json:"user_id"`
}

type Store struct {
	mux *sync.RWMutex
	s   map[string]map[string]string
	cfg *config.Config
}

func NewStore(cfg *config.Config) (repository.Repository, error) {
	store := &Store{
		mux: &sync.RWMutex{},
		s:   make(map[string]map[string]string),
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

	st.s = make(map[string]map[string]string)
	for _, rec := range arr {
		st.s[rec.UserID][rec.ShortURL] = rec.OriginalURL
	}

	return nil
}

func (st *Store) save() error {
	st.mux.Lock()
	defer st.mux.Unlock()

	arr := make([]URLRecord, 0, len(st.s))
	id := 1
	for userID, urlRecords := range st.s {
		for shortURL, originalURL := range urlRecords {
			rec := URLRecord{
				UUID:        strconv.Itoa(id),
				ShortURL:    shortURL,
				OriginalURL: originalURL,
				UserID:      userID,
			}
			arr = append(arr, rec)
			id++
		}
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

	for _, shortMap := range st.s {
		if url, exists := shortMap[req.ID]; exists {
			return &models.GetShortenerResponse{
				URL: url,
			}, nil
		}
	}
	return nil, newErrGetShortenerNotFound(req.ID)
}

func (st *Store) GetShortenerUrls(ctx context.Context, userID string) ([]models.GetShortenerUrls, error) {
	res, ok := st.s[userID]
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

func (st *Store) SetShortener(ctx context.Context, req *models.SetShortenerRequest) error {
	st.mux.Lock()
	var errUniqueURL error
	for key, v := range st.s[req.UserID] {
		if v == req.URL {
			req.ID = key
			errUniqueURL = myerrors.ErrConflictURL
		}
	}

	if st.s[req.UserID] == nil {
		st.s[req.UserID] = make(map[string]string)
	}

	st.s[req.UserID][req.ID] = req.URL
	st.mux.Unlock()

	err := st.save()
	if err != nil {
		return err
	}

	return errUniqueURL
}

func (st *Store) SetShortenerBatch(ctx context.Context, req []models.SetShortenerBatchRequest) error {
	st.mux.Lock()
	var errUniqueURL error
	for i, r := range req {
		for key, v := range st.s[r.UserID] {
			if v == r.URL {
				req[i].ID = key
				r.ID = key
				errUniqueURL = myerrors.ErrConflictURL
			}
		}
		if st.s[r.UserID] == nil {
			st.s[r.UserID] = make(map[string]string)
		}

		st.s[r.UserID][r.ID] = r.URL
	}
	st.mux.Unlock()

	err := st.save()
	if err != nil {
		return err
	}

	return errUniqueURL
}

func (st *Store) Close() error {
	return nil
}

func (st *Store) Ping(ctx context.Context) error {
	return nil
}
