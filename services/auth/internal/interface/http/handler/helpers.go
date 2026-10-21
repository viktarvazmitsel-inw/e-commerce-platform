package handler

import (
	"authorization/internal/domain"
	"net/http"
	"time"
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
