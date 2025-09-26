package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Evlushin/shorturl/internal/handler/config"
	"github.com/Evlushin/shorturl/internal/logger"
	"github.com/Evlushin/shorturl/internal/middleware"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
)

func Serve(cfg config.Config, shortener Shortener) error {
	h := newHandlers(shortener, cfg)
	router := newRouter(h)

	logger.Log.Info("Starting server", zap.String("addr", cfg.ServerAddr))

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	return srv.ListenAndServe()
}

func newRouter(h *handlers) *chi.Mux {
	r := chi.NewRouter()

	r.Use(logger.RequestLogger)
	r.Use(middleware.GzipMiddleware)

	r.Post("/", h.SetShortener)
	r.Get("/{id}", h.GetShortener)
	r.Get("/ping", h.Ping)

	r.Route("/api", func(r chi.Router) {
		r.Route("/shorten", func(r chi.Router) {
			r.Post("/", h.SetShortenerAPI)
			r.Post("/batch", h.SetShortenerBatchAPI)
		})
	})

	return r
}

type Shortener interface {
	GetShortener(ctx context.Context, req *models.GetShortenerRequest) (*models.GetShortenerResponse, error)
	SetShortener(ctx context.Context, req *models.SetShortenerRequest) (*models.SetShortenerResponse, error)
	SetShortenerBatch(ctx context.Context, req []models.RequestBatch) ([]models.SetShortenerBatchRequest, error)
	Ping(ctx context.Context) error
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
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	resp, err := h.shortener.GetShortener(ctx, &models.GetShortenerRequest{
		ID: id,
	})
	if err != nil {
		if errors.Is(err, myerrors.ErrGetShortenerInvalidRequest) || errors.Is(err, myerrors.ErrValidateShortenerInvalidRequest) || errors.Is(err, myerrors.ErrGetShortenerNotFound) {
			logger.Log.Debug("bad request", zap.Int("status", 400), zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		logger.Log.Error("failed to get shortener", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", resp.URL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *handlers) Ping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := h.shortener.Ping(ctx)
	if err != nil {
		logger.Log.Error("ping store error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *handlers) SetShortener(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body, err := io.ReadAll(r.Body)

	if err != nil {
		logger.Log.Debug("error reading request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp, err := h.shortener.SetShortener(ctx, &models.SetShortenerRequest{
		URL: string(body),
	})

	isErrConflictURL := errors.Is(err, myerrors.ErrConflictURL)
	if err != nil && !isErrConflictURL {
		if errors.Is(err, myerrors.ErrGetShortenerInvalidRequest) || errors.Is(err, myerrors.ErrValidateShortenerInvalidRequest) {
			logger.Log.Debug("bad request", zap.Int("status", 400), zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		logger.Log.Error("failed set shortener", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fullURL := fmt.Sprintf("%s/%s", h.cfg.BaseAddr, resp.ID)

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", strconv.Itoa(len(fullURL)))

	status := http.StatusCreated
	if isErrConflictURL {
		status = http.StatusConflict
	}

	w.WriteHeader(status)
	w.Write([]byte(fullURL))
}

func errorJSON(w http.ResponseWriter, message string, code int) {
	errResp := models.ErrorJSONResponse{
		Message: message,
	}

	logger.Log.Debug(message)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errResp)
}

func (h *handlers) SetShortenerAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		errorJSON(w, myerrors.ErrContentType.Error(), http.StatusBadRequest)
		return
	}

	var req models.Request
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Debug("failed decode json", zap.Int("status", 400), zap.Error(err))
		errorJSON(w, myerrors.ErrJSONDecode.Error(), http.StatusBadRequest)
		return
	}

	shortener, err := h.shortener.SetShortener(ctx, &models.SetShortenerRequest{
		URL: req.URL,
	})

	isErrConflictURL := errors.Is(err, myerrors.ErrConflictURL)
	if err != nil && !isErrConflictURL {
		if errors.Is(err, myerrors.ErrGetShortenerInvalidRequest) || errors.Is(err, myerrors.ErrValidateShortenerInvalidRequest) {
			logger.Log.Debug("bad request", zap.Int("status", 400), zap.Error(err))
			errorJSON(w, err.Error(), http.StatusBadRequest)
			return
		}
		logger.Log.Error("failed set shortener", zap.Error(err))
		errorJSON(w, myerrors.ErrInternalServer.Error(), http.StatusInternalServerError)
		return
	}

	fullURL := fmt.Sprintf("%s/%s", h.cfg.BaseAddr, shortener.ID)

	resp := models.Response{
		Result: fullURL,
	}

	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(resp)
	if err != nil {
		logger.Log.Error("failed json encode", zap.Error(err))
		errorJSON(w, myerrors.ErrInternalServer.Error(), http.StatusInternalServerError)
		return
	}

	jsonBytes := buf.Bytes()
	length := len(jsonBytes)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(length))

	status := http.StatusCreated
	if isErrConflictURL {
		status = http.StatusConflict
	}

	w.WriteHeader(status)
	w.Write(jsonBytes)
}

func (h *handlers) SetShortenerBatchAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		errorJSON(w, myerrors.ErrContentType.Error(), http.StatusBadRequest)
		return
	}

	var req []models.RequestBatch
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Debug("failed decode json", zap.Int("status", 400), zap.Error(err))
		errorJSON(w, myerrors.ErrJSONDecode.Error(), http.StatusBadRequest)
		return
	}

	shorteners, err := h.shortener.SetShortenerBatch(ctx, req)

	isErrConflictURL := errors.Is(err, myerrors.ErrConflictURL)
	if err != nil && !isErrConflictURL {
		if errors.Is(err, myerrors.ErrGetShortenerInvalidRequest) || errors.Is(err, myerrors.ErrValidateShortenerInvalidRequest) {
			logger.Log.Debug("bad request", zap.Int("status", 400), zap.Error(err))
			errorJSON(w, err.Error(), http.StatusBadRequest)
			return
		}
		logger.Log.Error("failed set shortener", zap.Error(err))
		errorJSON(w, myerrors.ErrInternalServer.Error(), http.StatusInternalServerError)
		return
	}

	var resp []models.ResponseBatch
	for _, shortener := range shorteners {
		fullURL := fmt.Sprintf("%s/%s", h.cfg.BaseAddr, shortener.ID)
		resp = append(resp, models.ResponseBatch{
			CorrelationID: shortener.CorrelationID,
			ShortURL:      fullURL,
		})
	}

	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(resp)
	if err != nil {
		logger.Log.Error("failed json encode", zap.Error(err))
		errorJSON(w, myerrors.ErrInternalServer.Error(), http.StatusInternalServerError)
		return
	}

	jsonBytes := buf.Bytes()
	length := len(jsonBytes)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(length))

	status := http.StatusCreated
	if isErrConflictURL {
		status = http.StatusConflict
	}

	w.WriteHeader(status)
	w.Write(jsonBytes)
}
