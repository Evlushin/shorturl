package models

import "github.com/golang-jwt/jwt/v4"

type Request struct {
	URL string `json:"url"`
}

type RequestBatch struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type Response struct {
	Result string `json:"result"`
}

type ResponseBatch struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type ResponseUrls struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type ErrorJSONResponse struct {
	Message string `json:"message"`
}

type GetShortenerRequest struct {
	ID     string
	UserID string
}

type SetShortenerResponse struct {
	ID string
}

type SetShortenerBatchResponse struct {
	CorrelationID string
	ID            string
}

type GetShortenerResponse struct {
	URL string
}

type SetShortenerRequest struct {
	ID     string
	URL    string
	UserID string
}

type SetShortenerBatchRequest struct {
	CorrelationID string
	ID            string
	URL           string
	UserID        string
}

type GetShortenerUrls struct {
	ID  string
	URL string
}

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}
