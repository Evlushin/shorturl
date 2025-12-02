package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/Evlushin/shorturl/internal/config"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"github.com/Evlushin/shorturl/internal/repository"
)

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	IsDeleted   bool   `json:"is_deleted"`
}

type Store struct {
	mux *sync.RWMutex
	s   map[string]map[string]models.GetShortenerResponse
	cfg *config.Config
}

func NewStore(cfg *config.Config) (repository.Repository, error) {
	store := &Store{
		mux: &sync.RWMutex{},
		s:   make(map[string]map[string]models.GetShortenerResponse),
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

	st.s = make(map[string]map[string]models.GetShortenerResponse)
	for _, rec := range arr {
		st.s[rec.UserID][rec.ShortURL] = models.GetShortenerResponse{
			URL:       rec.OriginalURL,
			IsDeleted: rec.IsDeleted,
		}
	}

	return nil
}

func (st *Store) save() error {

	arr := make([]URLRecord, 0, len(st.s))
	id := 1
	for userID, urlRecords := range st.s {
		for shortURL, originalURL := range urlRecords {
			rec := URLRecord{
				UUID:        strconv.Itoa(id),
				ShortURL:    shortURL,
				OriginalURL: originalURL.URL,
				UserID:      userID,
				IsDeleted:   originalURL.IsDeleted,
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

func (st *Store) DeleteShortenerUrls(ctx context.Context, req []models.RequestIDBatch, userID string) error {
	st.mux.Lock()
	defer st.mux.Unlock()

	res, ok := st.s[userID]
	if !ok {
		return myerrors.ErrGetShortenerNotFound
	}

	for _, id := range req {
		_, ok = res[string(id)]
		if ok {
			item := st.s[userID][string(id)]
			item.IsDeleted = true
			st.s[userID][string(id)] = item
		}
	}

	err := st.save()
	if err != nil {
		return err
	}

	return nil
}

func (st *Store) GetShortener(ctx context.Context, req *models.GetShortenerRequest) (*models.GetShortenerResponse, error) {
	st.mux.Lock()
	defer st.mux.Unlock()

	for _, shortMap := range st.s {
		if url, exists := shortMap[req.ID]; exists {
			return &url, nil
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
		if !v.IsDeleted {
			urls = append(urls, models.GetShortenerUrls{
				ID:  id,
				URL: v.URL,
			})
		}
	}

	return urls, nil
}

func (st *Store) SetShortener(ctx context.Context, req *models.SetShortenerRequest) error {
	st.mux.Lock()
	defer st.mux.Unlock()

	var errUniqueURL error
	for key, v := range st.s[req.UserID] {
		if v.URL == req.URL {
			req.ID = key
			errUniqueURL = myerrors.ErrConflictURL
		}
	}

	if st.s[req.UserID] == nil {
		st.s[req.UserID] = make(map[string]models.GetShortenerResponse)
	}

	st.s[req.UserID][req.ID] = models.GetShortenerResponse{
		URL:       req.URL,
		IsDeleted: false,
	}

	err := st.save()
	if err != nil {
		return err
	}

	return errUniqueURL
}

func (st *Store) SetShortenerBatch(ctx context.Context, req []models.SetShortenerBatchRequest) error {
	st.mux.Lock()
	defer st.mux.Unlock()

	var errUniqueURL error
	for i, r := range req {
		for key, v := range st.s[r.UserID] {
			if v.URL == r.URL {
				req[i].ID = key
				r.ID = key
				errUniqueURL = myerrors.ErrConflictURL
			}
		}
		if st.s[r.UserID] == nil {
			st.s[r.UserID] = make(map[string]models.GetShortenerResponse)
		}

		st.s[r.UserID][r.ID] = models.GetShortenerResponse{
			URL:       r.URL,
			IsDeleted: false,
		}
	}

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
