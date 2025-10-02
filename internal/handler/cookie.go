package handler

import (
	"fmt"
	"github.com/Evlushin/shorturl/internal/handler/config"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"time"
)

func BuildJWTString(userID string, cfg config.Config) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, models.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(365 * 24 * time.Hour)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(cfg.SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func SetUserCookie(w http.ResponseWriter, cfg config.Config, userID string) error {
	signature, err := BuildJWTString(userID, cfg)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    signature,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,                                // установите true для HTTPS
		Expires:  time.Now().Add(365 * 24 * time.Hour), // срок до 1 года
	})

	return nil
}

func GetUserID(r *http.Request, cfg config.Config) (string, error) {
	cookie, err := r.Cookie("jwt")
	if err != nil {
		return "", err
	}

	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(cfg.SecretKey), nil
		})
	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", myerrors.ErrValidateUserID
	}

	return claims.UserID, nil
}
