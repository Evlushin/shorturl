package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/Evlushin/shorturl/internal/handler/config"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/Evlushin/shorturl/internal/models"
)

// ExampleHandlers_SetShortenerAPI демонстрирует создание сокращённого URL через JSON‑запрос.
func ExampleHandlers_SetShortenerAPI() {
	// Подготавливаем тестовый обработчик
	h := newHandlers(nil, config.Config{BaseAddr: "http://localhost:8080"}, nil)

	// Формируем тело запроса
	reqBody := `{"url": "https://example.com/very/long/path"}`
	req := httptest.NewRequest(http.MethodPost, "/api/url", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Запускаем обработчик
	w := httptest.NewRecorder()
	h.SetShortenerAPI(w, req)

	// Проверяем ответ (в реальном тесте использовали бы assert)
	resp := w.Result()
	defer resp.Body.Close()

	// Ожидаем: статус 201 Created, Content-Type: application/json
	// Тело ответа содержит {"result": "http://localhost:8080/b086990c0c744d288448be10219de84e"}
}

// ExampleHandlers_SetShortener демонстрирует создание сокращённого URL через plain‑text запрос.
func ExampleHandlers_SetShortener() {
	h := newHandlers(nil, config.Config{BaseAddr: "http://localhost:8080"}, nil)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com/another/long/url"))
	w := httptest.NewRecorder()
	h.SetShortener(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Ожидаем: статус 201 Created, Content-Type: text/plain
	// Тело ответа: http://localhost:8080/b086990c0c744d288448be10219de84e
}

// ExampleHandlers_SetShortenerBatchAPI демонстрирует пакетное создание сокращённых URL.
func ExampleHandlers_SetShortenerBatchAPI() {
	h := newHandlers(nil, config.Config{BaseAddr: "http://localhost:8080"}, nil)

	batchReq := []models.RequestBatch{
		{OriginalURL: "https://example.com/page1", CorrelationID: "cid1"},
		{OriginalURL: "https://example.com/page2", CorrelationID: "cid2"},
	}
	reqBody, _ := json.Marshal(batchReq)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.SetShortenerBatchAPI(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Ожидаем: статус 201 Created, Content-Type: application/json
	// Тело ответа:
	// [
	//   {"correlation_id": "cid1", "short_url": "http://localhost:8080/xyz789"},
	//   {"correlation_id": "cid2", "short_url": "http://localhost:8080/uvw012"}
	// ]
}

// ExampleHandlers_GetShortener демонстрирует редирект по короткому ID.
func ExampleHandlers_GetShortener() {
	h := newHandlers(nil, config.Config{BaseAddr: "http://localhost:8080"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	w := httptest.NewRecorder()
	h.GetShortener(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Ожидаем: статус 307 Temporary Redirect
	// Заголовок Location: https://example.com/very/long/path
}

// ExampleHandlers_GetShortenerUrlsAPI демонстрирует получение всех сокращённых URL пользователя.
func ExampleHandlers_GetShortenerUrlsAPI() {
	h := newHandlers(nil, config.Config{BaseAddr: "http://localhost:8080"}, nil)

	// Имитируем аутентифицированного пользователя в контексте запроса
	ctx := context.WithValue(context.Background(), "userID", "f9d82f7d-f8da-4943-9fc9-6b64e3c78fb3")
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.GetShortenerUrlsAPI(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Ожидаем: статус 200 OK, Content-Type: application/json
	// Тело ответа:
	// [
	//   {"short_url": "http://localhost:8080/abc123", "original_url": "https://example.com/page1"},
	//   {"short_url": "http://localhost:8080/def456", "original_url": "https://example.com/page2"}
	// ]
}

// ExampleHandlers_DeleteShortenerUrlsAPI демонстрирует удаление нескольких сокращённых URL.
func ExampleHandlers_DeleteShortenerUrlsAPI() {
	h := newHandlers(nil, config.Config{BaseAddr: "http://localhost:8080"}, nil)

	deleteReq := []models.RequestIDBatch{
		"b086990c0c744d288448be10219de84e",
		"cfe4d8a5658e42cc8ee34b1881bde2a8",
	}
	reqBody, _ := json.Marshal(deleteReq)
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	// Имитируем аутентифицированного пользователя
	ctx := context.WithValue(context.Background(), "userID", "f9d82f7d-f8da-4943-9fc9-6b64e3c78fb3")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.DeleteShortenerUrlsAPI(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Ожидаем: статус 202 Accepted (удаление ставится в очередь)
}

// ExampleHandlers_Ping демонстрирует проверку работоспособности сервиса.
func ExampleHandlers_Ping() {
	h := newHandlers(nil, config.Config{BaseAddr: "http://localhost:8080"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	h.Ping(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Ожидаем: статус 200 OK (сервис и хранилище работоспособны)
}
