package models

type Request struct {
	URL string `json:"url"`
}

type Response struct {
	Result string `json:"result"`
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

type GetShortenerResponse struct {
	URL string
}

type SetShortenerRequest struct {
	ID  string
	URL string
}
