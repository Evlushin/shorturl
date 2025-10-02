package service

import (
	"fmt"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"net/url"
	"regexp"
	"strings"
)

func GetShortenerValidateRequest(req *models.GetShortenerRequest) error {
	validPattern := regexp.MustCompile(`^[A-Za-z0-9]{8}$`)
	if !validPattern.MatchString(req.ID) {
		return myerrors.ErrValidateShortenerInvalidRequest
	}

	return nil
}

func SetShortenerValidateRequest(req *models.SetShortenerRequest) error {
	_, err := url.ParseRequestURI(req.URL)

	if err != nil {
		return fmt.Errorf("%w : URL : %s", myerrors.ErrValidateShortenerInvalidRequest, req.URL)
	}

	return nil
}

func SetShortenerBatchValidateRequest(req []models.RequestBatch) error {
	notURL := make([]string, 0)
	for _, item := range req {
		err := SetShortenerValidateRequest(&models.SetShortenerRequest{
			URL: item.OriginalURL,
		})

		if err != nil {
			notURL = append(notURL, item.OriginalURL)
		}
	}

	if len(notURL) > 0 {
		return fmt.Errorf("%w : URLs : %s", myerrors.ErrValidateShortenerInvalidRequest, strings.Join(notURL, ", "))
	}

	return nil
}
