package handler

import (
	"encoding/json"
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
	store, _ := repository.NewStore("storage.txt")
	shortenerService := service.NewShortener(store)
	return newHandlers(shortenerService, cfg)
}

func Test_handlers_SetShortener(t *testing.T) {
	h := getHandlers()

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
	h := getHandlers()
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
	h := getHandlers()

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
