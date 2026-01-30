// Package handler implements HTTP handlers for the URL shortening service.
//
// Main endpoints:
//   - POST / and POST /api/url — create a shortened URL
//   - GET /{id} — redirect to the original URL
//   - GET /api/user/urls — retrieve all user URLs
//   - DELETE /api/user/urls — delete a user URL
//   - GET /ping — check service health
//   - POST /api/shorten/batch — batch create shortened URLs
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Evlushin/shorturl/internal/handler/config"
	"github.com/Evlushin/shorturl/internal/handler/utils/auth"
	"github.com/Evlushin/shorturl/internal/logger"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"github.com/Evlushin/shorturl/internal/observers/subjects"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Shortener defines the interface for URL shortening operations.
// It provides methods to create, retrieve, and manage shortened URLs.
type Shortener interface {

	// GetStats get shortener statistics
	// Get the number of URLs and the number of users
	GetStats(ctx context.Context) (*models.ResponseStats, error)

	// GetShortener retrieves a shortened URL by its ID.
	// Returns the full URL if found, otherwise returns an error.
	GetShortener(ctx context.Context, req *models.GetShortenerRequest) (*models.GetShortenerResponse, error)

	// SetShortener creates a new shortened URL from the provided original URL.
	// Returns the shortened URL ID if successful, otherwise returns an error.
	SetShortener(ctx context.Context, req *models.SetShortenerRequest) (*models.SetShortenerResponse, error)

	// SetShortenerBatch creates multiple shortened URLs in a single request.
	// Takes a slice of batch requests and user ID, returns a slice of responses.
	SetShortenerBatch(ctx context.Context, req []models.RequestBatch, userID string) ([]models.SetShortenerBatchRequest, error)

	// GetShortenerUrls retrieves all shortened URLs for a specific user.
	// Returns a slice of URL records, or an error if operation fails.
	GetShortenerUrls(ctx context.Context, userID string) ([]models.GetShortenerUrls, error)

	// DeleteShortenerUrls deletes multiple shortened URLs by their IDs for a specific user.
	// Takes a slice of ID requests and user ID, returns an error if operation fails.
	DeleteShortenerUrls(ctx context.Context, req []models.RequestIDBatch, userID string) error

	// Close gracefully shuts down the shortener service.
	// Returns an error if shutdown fails.
	Close() error

	// Ping checks the health of the underlying storage system.
	// Returns nil if storage is healthy, otherwise returns an error.
	Ping(ctx context.Context) error
}

// Handlers represents the HTTP handlers for the URL shortener service.
// It encapsulates the business logic and routes HTTP requests to appropriate methods.
type Handlers struct {
	shortener    Shortener
	cfg          config.Config
	auditManager *subjects.AuditManager
}

// newHandlers creates a new instance of handlers.
// Parameters:
//   - shortener: implementation of the Shortener interface
//   - cfg: application configuration
//   - auditManager: manager for audit events
//
// Returns a pointer to the newly created handlers instance.
func newHandlers(shortener Shortener, cfg config.Config, auditManager *subjects.AuditManager) *Handlers {
	return &Handlers{
		shortener:    shortener,
		cfg:          cfg,
		auditManager: auditManager,
	}
}

// GetStats shortener statistics
// Endpoint: GET /api/internal/stats
// Returns:
//   - HTTP 200 OK with JSON array of ResponseStats
//   - HTTP 500 Internal Server Error on failure
func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.shortener.GetStats(r.Context())
	if err != nil {
		logger.Log.Error("failed to get stats", zap.Error(err))
		errorJSON(w, myerrors.ErrInternalServer.Error(), http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(stats)
	if err != nil {
		logger.Log.Error("failed json encode", zap.Error(err))
		errorJSON(w, myerrors.ErrInternalServer.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

// DeleteShortenerUrlsAPI handles the HTTP request to delete multiple shortened URLs.
// Endpoint: DELETE /api/user/urls
// Expected request body: JSON array of RequestIDBatch objects
// Returns:
//   - HTTP 202 Accepted if deletion is queued successfully.
//   - HTTP 400 Bad Request on invalid input
//   - HTTP 500 Internal Server Error on failure
func (h *Handlers) DeleteShortenerUrlsAPI(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCtxUserID(r.Context())
	if err != nil || user == nil {
		logger.Log.Error("user ID not found in config")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var req []models.RequestIDBatch
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Debug("failed decode json", zap.Int("status", http.StatusBadRequest), zap.Error(err))
		errorJSON(w, myerrors.ErrJSONDecode.Error(), http.StatusBadRequest)
		return
	}

	go func() {
		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := h.shortener.DeleteShortenerUrls(ctxWithTimeout, req, user.UserID)
		if err != nil {
			logger.Log.Error("failed to get shortener", zap.Error(err))
			return
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}

// GetShortenerUrlsAPI handles the HTTP request to retrieve all shortened URLs for the current user.
// Endpoint: GET /api/user/urls
// Returns:
//   - HTTP 200 OK with JSON array of ResponseUrls if URLs exist
//   - HTTP 204 No Content if no URLs found
//   - HTTP 500 Internal Server Error on failure
func (h *Handlers) GetShortenerUrlsAPI(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCtxUserID(r.Context())
	if err != nil || user == nil {
		logger.Log.Error("user ID not found in config")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	shorteners, err := h.shortener.GetShortenerUrls(r.Context(), user.UserID)
	if err != nil {
		if errors.Is(err, myerrors.ErrGetShortenerNotFound) {
			logger.Log.Debug("no content", zap.Int("status", http.StatusNoContent), zap.Error(err))
			w.WriteHeader(http.StatusNoContent)
			return
		}

		logger.Log.Error("failed to get shortener", zap.Error(err))
		errorJSON(w, myerrors.ErrInternalServer.Error(), http.StatusInternalServerError)
		return
	}

	var resp []models.ResponseUrls
	for _, shortener := range shorteners {
		fullURL := fmt.Sprintf("%s/%s", h.cfg.BaseAddr, shortener.ID)
		resp = append(resp, models.ResponseUrls{
			ShortURL:    fullURL,
			OriginalURL: shortener.URL,
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

	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

// GetShortener handles the HTTP request to redirect to the original URL using a short ID.
// Endpoint: GET /{id}
// Returns:
//   - HTTP 307 Temporary Redirect to the original URL if found
//   - HTTP 400 Bad Request on invalid input
//   - HTTP 410 Gone if URL was deleted
//   - HTTP 500 Internal Server Error on unexpected failure
func (h *Handlers) GetShortener(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := auth.GetCtxUserID(r.Context())
	if err != nil {
		logger.Log.Error("failed to get shortener", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := h.shortener.GetShortener(r.Context(), &models.GetShortenerRequest{
		ID: id,
	})
	if err != nil {
		if errors.Is(err, myerrors.ErrGetShortenerInvalidRequest) || errors.Is(err, myerrors.ErrValidateShortenerInvalidRequest) || errors.Is(err, myerrors.ErrGetShortenerNotFound) {
			logger.Log.Debug("bad request", zap.Int("status", http.StatusBadRequest), zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if errors.Is(err, myerrors.ErrGone410) {
			logger.Log.Debug("bad request", zap.Int("status", http.StatusGone), zap.Error(err))
			http.Error(w, err.Error(), http.StatusGone)
			return
		}

		logger.Log.Error("failed to get shortener", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", resp.URL)
	w.WriteHeader(http.StatusTemporaryRedirect)

	h.auditManager.NotifyObservers(models.AuditEvent{
		TS:      time.Now().Unix(),
		Action:  models.RAShorten,
		UserID:  user.UserID,
		OrigURL: resp.URL,
	})
}

// Ping handles the health check request for the service.
// Endpoint: GET /ping
// Returns:
//   - HTTP 200 OK if storage is healthy
//   - HTTP 500 Internal Server Error if storage is unhealthy
func (h *Handlers) Ping(w http.ResponseWriter, r *http.Request) {
	err := h.shortener.Ping(r.Context())
	if err != nil {
		logger.Log.Error("ping store error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// SetShortener handles the HTTP request to create a shortened URL from plain text input.
// Endpoint: POST /
// Expected request body: plain text with the original URL
// Returns:
//   - HTTP 201 Created with the shortened URL if successful
//   - HTTP 409 Conflict if URL already exists
//   - HTTP 400 Bad Request on invalid input
//   - HTTP 500 Internal Server Error on unexpected failure
func (h *Handlers) SetShortener(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCtxUserID(r.Context())
	if err != nil || user == nil {
		logger.Log.Error("user ID not found in config")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)

	if err != nil {
		logger.Log.Debug("error reading request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ssr := models.SetShortenerRequest{
		URL:    string(body),
		UserID: user.UserID,
	}

	resp, err := h.shortener.SetShortener(r.Context(), &ssr)

	isErrConflictURL := errors.Is(err, myerrors.ErrConflictURL)
	if err != nil && !isErrConflictURL {
		if errors.Is(err, myerrors.ErrGetShortenerInvalidRequest) || errors.Is(err, myerrors.ErrValidateShortenerInvalidRequest) {
			logger.Log.Debug("bad request", zap.Int("status", http.StatusBadRequest), zap.Error(err))
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

	h.auditManager.NotifyObservers(models.AuditEvent{
		TS:      time.Now().Unix(),
		Action:  models.RAShorten,
		UserID:  ssr.UserID,
		OrigURL: ssr.URL,
	})
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

// SetShortenerAPI handles the HTTP request to create a shortened URL from JSON input.
// Endpoint: POST /api/url
// Expected request body: JSON object with "url" field
// Returns:
//   - HTTP 201 Created with JSON response containing the shortened URL
//   - HTTP 409 Conflict if URL already exists
//   - HTTP 400 Bad Request on invalid input or content type
//   - HTTP 500 Internal Server Error on unexpected failure
func (h *Handlers) SetShortenerAPI(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCtxUserID(r.Context())
	if err != nil || user == nil {
		logger.Log.Error("user ID not found in config")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		errorJSON(w, myerrors.ErrContentType.Error(), http.StatusBadRequest)
		return
	}

	var req models.Request
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Debug("failed decode json", zap.Int("status", http.StatusBadRequest), zap.Error(err))
		errorJSON(w, myerrors.ErrJSONDecode.Error(), http.StatusBadRequest)
		return
	}

	ssr := models.SetShortenerRequest{
		URL:    req.URL,
		UserID: user.UserID,
	}

	shortener, err := h.shortener.SetShortener(r.Context(), &models.SetShortenerRequest{
		URL:    req.URL,
		UserID: user.UserID,
	})

	isErrConflictURL := errors.Is(err, myerrors.ErrConflictURL)
	if err != nil && !isErrConflictURL {
		if errors.Is(err, myerrors.ErrGetShortenerInvalidRequest) || errors.Is(err, myerrors.ErrValidateShortenerInvalidRequest) {
			logger.Log.Debug("bad request", zap.Int("status", http.StatusBadRequest), zap.Error(err))
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

	h.auditManager.NotifyObservers(models.AuditEvent{
		TS:      time.Now().Unix(),
		Action:  models.RAShorten,
		UserID:  ssr.UserID,
		OrigURL: ssr.URL,
	})
}

// SetShortenerBatchAPI handles the HTTP request to create multiple shortened URLs in batch.
// Endpoint: POST /api/shorten/batch
// Expected request body: JSON array of objects with "original_url" and "correlation_id" fields
// Returns:
//   - HTTP 201 Created with JSON array of results containing correlation IDs and shortened URLs
//   - HTTP 409 Conflict if any URL already exists
//   - HTTP 400 Bad Request on invalid input or content type
//   - HTTP 500 Internal Server Error on unexpected failure
func (h *Handlers) SetShortenerBatchAPI(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCtxUserID(r.Context())
	if err != nil || user == nil {
		logger.Log.Error("user ID not found in config")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		errorJSON(w, myerrors.ErrContentType.Error(), http.StatusBadRequest)
		return
	}

	var req []models.RequestBatch
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Debug("failed decode json", zap.Int("status", http.StatusBadRequest), zap.Error(err))
		errorJSON(w, myerrors.ErrJSONDecode.Error(), http.StatusBadRequest)
		return
	}

	shorteners, err := h.shortener.SetShortenerBatch(r.Context(), req, user.UserID)

	isErrConflictURL := errors.Is(err, myerrors.ErrConflictURL)
	if err != nil && !isErrConflictURL {
		if errors.Is(err, myerrors.ErrGetShortenerInvalidRequest) || errors.Is(err, myerrors.ErrValidateShortenerInvalidRequest) {
			logger.Log.Debug("bad request", zap.Int("status", http.StatusBadRequest), zap.Error(err))
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
