package handler

import (
	"authorization/internal/domain"
	"authorization/internal/interface/http/middleware"
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

var (
	ErrUserIDUndefined = errors.New("user id is undefined")
	ErrUserIDInvalid   = errors.New("user id is invalid")
)

const refreshTokenCookie = "refresh_token"

func setRefreshTokenCookie(w http.ResponseWriter, pair domain.TokenPair) {
	maxAge := int(time.Until(pair.RefreshTokenExpiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookie,
		Value:    pair.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  pair.RefreshTokenExpiresAt,
		MaxAge:   maxAge,
	})
}

func getUserIDFromContext(ctx context.Context) (string, error) {
	id := ctx.Value(middleware.UserIDContextKey)

	if id == nil {
		return "", ErrUserIDUndefined
	}

	authorID, ok := id.(string)
	if !ok || strings.TrimSpace(authorID) == "" {
		return "", ErrUserIDInvalid
	}

	return authorID, nil
}
