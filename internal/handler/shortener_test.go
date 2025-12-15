package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/Evlushin/shorturl/internal/config"
	handlersConfig "github.com/Evlushin/shorturl/internal/handler/config"
	"github.com/Evlushin/shorturl/internal/logger"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/observers"
	"github.com/Evlushin/shorturl/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getConfig() config.Config {
	return config.Config{
		Handlers: handlersConfig.Config{
			ServerAddr: "",
			BaseAddr:   "",
			SecretKey:  "",
			Audit: handlersConfig.TAudit{
				//AuditFile: "audit.log",
				AuditFile: "",
			},
		},
		LogLevel:      "info",
		FileStorePath: "",
		//DatabaseDsn:   "host=127.127.126.41 port=5432 dbname=shorturl user=shorturl password=shorturl connect_timeout=10 sslmode=prefer",
		DatabaseDsn: "",
	}
}

func BenchmarkSetShortener(b *testing.B) {
	ctx := context.Background()
	cfg := getConfig()
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		b.Errorf("logger: %v", err)
	}
	s, err := service.NewShortener(cfg)
	if err != nil {
		b.Errorf("Failed to create service: %v", err)
	}
	defer s.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = s.SetShortener(ctx, &models.SetShortenerRequest{
			UserID: uuid.New().String(),
			URL:    "https://example.com",
		})
		if err != nil {
			b.Errorf("SetShortener failed on iteration %d: %v", i, err)
		}

	}
}

func BenchmarkSetShortenerBatch(b *testing.B) {
	ctx := context.Background()
	cfg := getConfig()
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		b.Errorf("logger: %v", err)
	}
	s, err := service.NewShortener(cfg)
	if err != nil {
		b.Errorf("Failed to create service: %v", err)
	}
	defer s.Close()

	r := []models.RequestBatch{
		{
			CorrelationID: uuid.New().String(),
			OriginalURL:   "https://example" + uuid.New().String() + ".com",
		},
		{
			CorrelationID: uuid.New().String(),
			OriginalURL:   "https://example" + uuid.New().String() + ".com",
		},
		{
			CorrelationID: uuid.New().String(),
			OriginalURL:   "https://example" + uuid.New().String() + ".com",
		},
		{
			CorrelationID: uuid.New().String(),
			OriginalURL:   "https://example" + uuid.New().String() + ".com",
		},
		{
			CorrelationID: uuid.New().String(),
			OriginalURL:   "https://example" + uuid.New().String() + ".com",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = s.SetShortenerBatch(ctx, r, strconv.Itoa(i))
		if err != nil {
			b.Errorf("SetShortenerBatch failed on iteration %d: %v", i, err)
		}
	}
}

func BenchmarkGetShortener(b *testing.B) {
	ctx := context.Background()
	cfg := getConfig()
	s, err := service.NewShortener(cfg)
	if err != nil {
		b.Errorf("Failed to create service: %v", err)
	}
	defer s.Close()

	id, err := s.SetShortener(ctx, &models.SetShortenerRequest{
		UserID: "1",
		URL:    "https://example.com",
	})
	if err != nil {
		b.Errorf("SetShortener failed on iteration: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = s.GetShortener(ctx, &models.GetShortenerRequest{
			ID: id.ID,
		})
		if err != nil {
			b.Errorf("GetShortener failed on iteration %d: %v", i, err)
		}
	}
}

func BenchmarkGetShortenerURLs(b *testing.B) {
	ctx := context.Background()
	cfg := getConfig()
	s, err := service.NewShortener(cfg)
	if err != nil {
		b.Errorf("Failed to create service: %v", err)
	}
	defer s.Close()

	r := []models.RequestBatch{
		{
			CorrelationID: uuid.New().String(),
			OriginalURL:   "https://example" + uuid.New().String() + ".com",
		},
		{
			CorrelationID: uuid.New().String(),
			OriginalURL:   "https://example" + uuid.New().String() + ".com",
		},
		{
			CorrelationID: uuid.New().String(),
			OriginalURL:   "https://example" + uuid.New().String() + ".com",
		},
		{
			CorrelationID: uuid.New().String(),
			OriginalURL:   "https://example" + uuid.New().String() + ".com",
		},
		{
			CorrelationID: uuid.New().String(),
			OriginalURL:   "https://example" + uuid.New().String() + ".com",
		},
	}

	_, err = s.SetShortenerBatch(ctx, r, "1")
	if err != nil {
		b.Errorf("SetShortenerBatch failed on iteration: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = s.GetShortenerUrls(ctx, "1")
		if err != nil {
			b.Errorf("GetShortenerUrls failed on iteration %d: %v", i, err)
		}
	}
}

func getHandlersMemory() (*Handlers, error) {
	ctx := context.Background()
	cfg := getConfig()
	cfg.Handlers.ServerAddr = "localhost:8080"
	cfg.Handlers.SecretKey = "123"
	shortenerService, err := service.NewShortener(cfg)
	auditManager := observers.InitAuditObservers(ctx, cfg.Handlers)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the shortener result from the store: %w", err)
	}
	return newHandlers(shortenerService, cfg.Handlers, auditManager), nil
}

func Test_handlers_SetShortener(t *testing.T) {
	h, err := getHandlersMemory()
	require.NoError(t, err)

	ts := httptest.NewServer(newRouter(h))
	defer ts.Close()

	type want struct {
		code        int
		request     string
		contentType string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "positive test #1",
			want: want{
				code:        201,
				request:     `https://practicum.yandex.ru/`,
				contentType: "text/plain",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestSet, err := http.NewRequest(http.MethodPost, ts.URL+"/", strings.NewReader(test.want.request))
			require.NoError(t, err)
			requestSet.Header.Add("Content-Type", test.want.contentType)
			resSet, err := ts.Client().Do(requestSet)
			require.NoError(t, err)
			defer resSet.Body.Close()

			resBodySet, err := io.ReadAll(resSet.Body)
			require.NoError(t, err)
			parseURL, err := url.Parse(string(resBodySet))
			require.NoError(t, err)

			assert.Equal(t, test.want.code, resSet.StatusCode)
			assert.Equal(t, test.want.contentType, resSet.Header.Get("Content-Type"))
			assert.Equal(t, strconv.Itoa(len(parseURL.String())), resSet.Header.Get("Content-Length"))
		})
	}
}

func Test_handlers_GetShortener(t *testing.T) {
	h, err := getHandlersMemory()
	require.NoError(t, err)

	ts := httptest.NewServer(newRouter(h))
	defer ts.Close()

	type want struct {
		code        int
		request     string
		contentType string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "positive test #1",
			want: want{
				code:        307,
				request:     `https://practicum.yandex.ru/`,
				contentType: "text/plain",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestSet, err := http.NewRequest(http.MethodPost, ts.URL+"/", strings.NewReader(test.want.request))
			require.NoError(t, err)
			requestSet.Header.Add("Content-Type", test.want.contentType)
			resSet, err := ts.Client().Do(requestSet)

			require.NoError(t, err)
			defer resSet.Body.Close()

			resBodySet, err := io.ReadAll(resSet.Body)
			require.NoError(t, err)
			parseURL, err := url.Parse(string(resBodySet))
			require.NoError(t, err)

			requestGet, err := http.NewRequest(http.MethodGet, ts.URL+parseURL.Path, nil)
			for _, cookie := range resSet.Cookies() {
				requestGet.AddCookie(cookie)
			}
			require.NoError(t, err)
			requestGet.Header.Add("Content-Type", test.want.contentType)

			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}

			resGet, err := client.Do(requestGet)
			require.NoError(t, err)
			defer resGet.Body.Close()

			assert.Equal(t, test.want.code, resGet.StatusCode)
			assert.Equal(t, test.want.request, resGet.Header.Get("Location"))
		})
	}
}

func Test_handlers_SetShortenerAPI(t *testing.T) {
	h, err := getHandlersMemory()
	require.NoError(t, err)

	ts := httptest.NewServer(newRouter(h))
	defer ts.Close()

	type want struct {
		code        int
		request     string
		contentType string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "positive test #1",
			want: want{
				code:        201,
				request:     `{"url": "https://practicum.yandex.ru"}`,
				contentType: "application/json",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestSet, err := http.NewRequest(http.MethodPost, ts.URL+"/api/shorten", strings.NewReader(test.want.request))
			require.NoError(t, err)
			requestSet.Header.Add("Content-Type", test.want.contentType)
			resSet, err := ts.Client().Do(requestSet)
			require.NoError(t, err)
			defer resSet.Body.Close()

			resBodySet, err := io.ReadAll(resSet.Body)
			require.NoError(t, err)

			var response map[string]string

			err = json.Unmarshal(resBodySet, &response)
			require.NoError(t, err)

			assert.Contains(t, response, "result", "JSON должен содержать ключ 'result'")

			_, err = url.Parse(response["result"])
			require.NoError(t, err)

			assert.Equal(t, test.want.code, resSet.StatusCode)
			assert.Equal(t, test.want.contentType, resSet.Header.Get("Content-Type"))
		})
	}
}

func Test_handlers_SetShortenerBatchAPI(t *testing.T) {
	h, err := getHandlersMemory()
	require.NoError(t, err)

	ts := httptest.NewServer(newRouter(h))
	defer ts.Close()

	type want struct {
		code        int
		request     string
		contentType string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "positive test #1",
			want: want{
				code: 201,
				request: `[
								{"correlation_id":"1","original_url":"https://practicum.yandex.ru/"},
								{"correlation_id":"2","original_url":"https://www.google.com/"}
						]`,
				contentType: "application/json",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestSet, err := http.NewRequest(http.MethodPost, ts.URL+"/api/shorten/batch", strings.NewReader(test.want.request))
			require.NoError(t, err)
			requestSet.Header.Add("Content-Type", test.want.contentType)
			resSet, err := ts.Client().Do(requestSet)
			require.NoError(t, err)
			defer resSet.Body.Close()

			resBodySet, err := io.ReadAll(resSet.Body)
			require.NoError(t, err)

			var response []map[string]string

			err = json.Unmarshal(resBodySet, &response)
			require.NoError(t, err)

			for _, r := range response {
				assert.Contains(t, r, "short_url", "JSON должен содержать ключ 'short_url'")
				assert.Contains(t, r, "correlation_id", "JSON должен содержать ключ 'correlation_id'")

				_, err = url.Parse(r["short_url"])
				require.NoError(t, err)
			}

			assert.Equal(t, test.want.code, resSet.StatusCode)
			assert.Equal(t, test.want.contentType, resSet.Header.Get("Content-Type"))
		})
	}
}

func Test_handlers_GetShortenerUrlsAPI(t *testing.T) {
	h, err := getHandlersMemory()
	require.NoError(t, err)

	ts := httptest.NewServer(newRouter(h))
	defer ts.Close()

	type want struct {
		code        int
		request     string
		contentType string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "positive test #1",
			want: want{
				code: 200,
				request: `[
								{"correlation_id":"1","original_url":"https://practicum.yandex.ru/"},
								{"correlation_id":"2","original_url":"https://www.google.com/"}
						]`,
				contentType: "application/json",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestSet, err := http.NewRequest(http.MethodPost, ts.URL+"/api/shorten/batch", strings.NewReader(test.want.request))
			require.NoError(t, err)
			requestSet.Header.Add("Content-Type", test.want.contentType)
			resSet, err := ts.Client().Do(requestSet)
			require.NoError(t, err)
			defer resSet.Body.Close()

			_, err = io.ReadAll(resSet.Body)
			require.NoError(t, err)

			requestGet, err := http.NewRequest(http.MethodGet, ts.URL+"/api/user/urls", nil)
			for _, cookie := range resSet.Cookies() {
				requestGet.AddCookie(cookie)
			}
			require.NoError(t, err)
			requestGet.Header.Add("Content-Type", test.want.contentType)
			resGet, err := ts.Client().Do(requestGet)
			require.NoError(t, err)
			defer resGet.Body.Close()

			resBodyGet, err := io.ReadAll(resGet.Body)
			require.NoError(t, err)

			var response []map[string]string

			err = json.Unmarshal(resBodyGet, &response)
			require.NoError(t, err)

			for _, r := range response {
				assert.Contains(t, r, "short_url", "JSON должен содержать ключ 'short_url'")
				assert.Contains(t, r, "original_url", "JSON должен содержать ключ 'original_url'")

				_, err = url.Parse(r["short_url"])
				require.NoError(t, err)

				_, err = url.Parse(r["original_url"])
				require.NoError(t, err)
			}

			assert.Equal(t, test.want.code, resGet.StatusCode)
			assert.Equal(t, test.want.contentType, resGet.Header.Get("Content-Type"))
		})
	}
}

func Test_handlers_DeleteShortenerUrlsAPI(t *testing.T) {
	h, err := getHandlersMemory()
	require.NoError(t, err)

	ts := httptest.NewServer(newRouter(h))
	defer ts.Close()

	type want struct {
		code        int
		request     string
		contentType string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "positive test #1",
			want: want{
				code:        202,
				request:     `https://practicum.yandex.ru/`,
				contentType: "application/json",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestSet, err := http.NewRequest(http.MethodPost, ts.URL+"/", strings.NewReader(test.want.request))
			require.NoError(t, err)
			requestSet.Header.Add("Content-Type", "text/plain")
			resSet, err := ts.Client().Do(requestSet)

			require.NoError(t, err)
			defer resSet.Body.Close()

			resBodySet, err := io.ReadAll(resSet.Body)
			require.NoError(t, err)
			parseURL, err := url.Parse(string(resBodySet))
			require.NoError(t, err)

			id := strings.TrimPrefix(parseURL.Path, "/")

			requestGet, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/user/urls", strings.NewReader(fmt.Sprintf(`["%s"]`, id)))
			for _, cookie := range resSet.Cookies() {
				requestGet.AddCookie(cookie)
			}
			require.NoError(t, err)
			requestGet.Header.Add("Content-Type", test.want.contentType)
			resGet, err := ts.Client().Do(requestGet)
			require.NoError(t, err)
			defer resGet.Body.Close()

			assert.Equal(t, test.want.code, resGet.StatusCode)
		})
	}
}
