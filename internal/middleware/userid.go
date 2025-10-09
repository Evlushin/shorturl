package middleware

import (
	"context"
	"fmt"
	"github.com/Evlushin/shorturl/internal/handler/config"
	"github.com/Evlushin/shorturl/internal/logger"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net/http"
	"time"
)

const UserKey ContextKey = "user"

type Claims struct {
	jwt.RegisteredClaims `json:",inline"`
	UserID               string `json:"user_id"`
}

func UserIDMiddleware(cfg config.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := GetUser(r, cfg)
			if err != nil {
				user = &models.Users{
					UserID:     uuid.NewString(),
					FromCookie: false,
				}

				if err := SetUserCookie(w, cfg, user.UserID); err != nil {
					logger.Log.Error("failed set user cookie", zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			ctx := context.WithValue(r.Context(), UserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func BuildJWTString(userID string, cfg config.Config) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
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

func GetUser(r *http.Request, cfg config.Config) (*models.Users, error) {
	cookie, err := r.Cookie("jwt")
	if err != nil {
		return nil, err
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(cfg.SecretKey), nil
		})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, myerrors.ErrValidateUserID
	}

	return &models.Users{
		UserID:     claims.UserID,
		FromCookie: true,
	}, nil
}
