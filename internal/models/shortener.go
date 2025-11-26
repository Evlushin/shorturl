package models

type Request struct {
	URL string `json:"url"`
}

type RequestBatch struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type RequestIDBatch string

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
	ID string `json:"id"`
}

type SetShortenerResponse struct {
	ID string `json:"id"`
}

type SetShortenerBatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ID            string `json:"id"`
}

type GetShortenerResponse struct {
	URL       string `json:"url"`
	IsDeleted bool   `json:"is_deleted"`
	UserID    string
}

type SetShortenerRequest struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	UserID string `json:"user_id"`
}

type SetShortenerBatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	ID            string `json:"id"`
	URL           string `json:"url"`
	UserID        string `json:"user_id"`
}

type GetShortenerUrls struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Users struct {
	UserID     string `json:"user_id"`
	FromCookie bool   `json:"from_cookie"`
}
