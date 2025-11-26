package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/Evlushin/shorturl/internal/models"
)

var (
	ErrUserNotFoundInContext       = errors.New("user not found in context")
	ErrUnexpectedUserTypeInContext = errors.New("unexpected user type in context")
)

type userCtxKey struct{}

func WithUser(ctx context.Context, user *models.Users) context.Context {
	return context.WithValue(ctx, userCtxKey{}, *user)
}

func GetCtxUserID(ctx context.Context) (*models.Users, error) {
	value := ctx.Value(userCtxKey{})
	if value == nil {
		return nil, ErrUserNotFoundInContext
	}

	if user, ok := value.(models.Users); ok {
		return &user, nil
	}
	return nil, fmt.Errorf("%w: %T", ErrUnexpectedUserTypeInContext, value)
}
