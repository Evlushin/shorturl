package myerrors

import "errors"

var (
	ErrGetShortenerNotFound            = errors.New("no url")
	ErrGetShortenerInvalidRequest      = errors.New("invalid get shortener request")
	ErrEndRandomStrings                = errors.New("end random strings")
	ErrValidateShortenerInvalidRequest = errors.New("invalid validate shortener request")
	ErrRepoFailed                      = errors.New("repo failed")
	ErrContentType                     = errors.New("error Content-Type")
	ErrJSONDecode                      = errors.New("error JSON decode")
	ErrInternalServer                  = errors.New("internal Server Error")
)
