package handler

import (
	"github.com/Evlushin/shorturl/internal/handler/config"
	"github.com/Evlushin/shorturl/internal/repository"
	"github.com/Evlushin/shorturl/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func getHandlers() *handlers {
	cfg := config.Config{
		ServerAddr: "localhost:8080",
	}
	store := repository.NewStore()
	shortenerService := service.NewShortener(store)
	return newHandlers(shortenerService, cfg)
}

func Test_handlers_SetShortener(t *testing.T) {
	h := getHandlers()

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
			requestSet := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.want.request))
			requestSet.Header.Add("Content-Type", test.want.contentType)

			wSet := httptest.NewRecorder()

			h.SetShortener(wSet, requestSet)

			resSet := wSet.Result()
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
	h := getHandlers()

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
			requestSet := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.want.request))
			requestSet.Header.Add("Content-Type", "text/plain")

			wSet := httptest.NewRecorder()

			h.SetShortener(wSet, requestSet)

			resSet := wSet.Result()
			defer resSet.Body.Close()

			resBodySet, err := io.ReadAll(resSet.Body)
			require.NoError(t, err)
			parseURL, err := url.Parse(string(resBodySet))
			require.NoError(t, err)

			requestGet := httptest.NewRequest(http.MethodGet, parseURL.Path, nil)
			requestGet.SetPathValue("id", parseURL.Path[1:])
			requestGet.Header.Add("Content-Type", test.want.contentType)

			wGet := httptest.NewRecorder()
			h.GetShortener(wGet, requestGet)

			resGet := wGet.Result()
			defer resGet.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, test.want.code, resGet.StatusCode)
			assert.Equal(t, test.want.request, resGet.Header.Get("Location"))
		})
	}
}
