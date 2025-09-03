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
