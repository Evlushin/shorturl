package models

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

type ErrorJSONResponse struct {
	Message string `json:"message"`
}

type GetShortenerRequest struct {
	ID string
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
	ID  string
	URL string
}

type SetShortenerBatchRequest struct {
	CorrelationID string
	ID            string
	URL           string
}
