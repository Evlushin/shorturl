package handler

import (
	"errors"
	"fmt"
	"github.com/Evlushin/shorturl/internal/handler/config"
	"github.com/Evlushin/shorturl/internal/repository"
	"github.com/Evlushin/shorturl/internal/service"
	"io"
	"log"
	"net/http"
	"strconv"
)

func Serve(cfg config.Config, shortener Shortener) error {
	h := newHandlers(shortener, cfg)
	router := newRouter(h)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	return srv.ListenAndServe()
}

func newRouter(h *handlers) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", h.SetShortener)
	mux.HandleFunc("GET /{id}", h.GetShortener)

	return mux
}

type Shortener interface {
	GetShortener(req *service.GetShortenerRequest) (*service.GetShortenerResponse, error)
	SetShortener(req *service.SetShortenerRequest) (*service.SetShortenerResponse, error)
}

type handlers struct {
	shortener Shortener
	cfg       config.Config
}

func newHandlers(shortener Shortener, cfg config.Config) *handlers {
	return &handlers{
		shortener: shortener,
		cfg:       cfg,
	}
}

func (h *handlers) GetShortener(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	resp, err := h.shortener.GetShortener(&service.GetShortenerRequest{
		ID: id,
	})
	if err != nil {
		if errors.Is(err, service.ErrGetShortenerInvalidRequest) || errors.Is(err, service.ErrValidateShortenerInvalidRequest) || errors.Is(err, repository.ErrGetShortenerNotFound) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("failed to get shortener: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", resp.URL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *handlers) SetShortener(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)

	if err != nil {
		log.Printf("error reading request body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := h.shortener.SetShortener(&service.SetShortenerRequest{
		URL: string(body),
	})

	if err != nil {
		if errors.Is(err, service.ErrGetShortenerInvalidRequest) || errors.Is(err, service.ErrValidateShortenerInvalidRequest) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("failed to get shortener: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fullURL := fmt.Sprintf("http://%s/%s", h.cfg.ServerAddr, resp.ID)

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", strconv.Itoa(len(fullURL)))
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fullURL))
}
