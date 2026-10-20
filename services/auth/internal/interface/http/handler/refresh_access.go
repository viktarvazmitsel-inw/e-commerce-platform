package handler

import (
	"context"
	"errors"
	"net/http"

	"authorization/internal/domain"
	"authorization/internal/usecase"
)

const refreshTokenCookie = "refresh_token"

type RefreshAccessUseCase interface {
	Execute(ctx context.Context, input usecase.RefreshTokenInput) (domain.TokenPair, error)
}

type RefreshAccessHandler struct {
	refreshAccess RefreshAccessUseCase
}

func NewRefreshAccessHandler(uc RefreshAccessUseCase) *RefreshAccessHandler {
	return &RefreshAccessHandler{
		refreshAccess: uc,
	}
}

func (h *RefreshAccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "METHOD_NOT_ALLOWED",
				Message: "method not allowed",
			},
		})
		return
	}

	cookie, err := r.Cookie(refreshTokenCookie)
	if err != nil || cookie.Value == "" {
		RespondJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_REFRESH_TOKEN",
				Message: "invalid or expired refresh token",
			},
		})
		return
	}

	tokenPair, err := h.refreshAccess.Execute(r.Context(), usecase.RefreshTokenInput{
		Token: cookie.Value,
	})
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	setRefreshTokenCookie(w, tokenPair)

	RespondJSON(w, http.StatusOK, &Response{
		Success: true,
		Data: map[string]string{
			"access_token": tokenPair.AccessToken,
		},
	})
}

func clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (h *RefreshAccessHandler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidRefreshToken):
		clearRefreshTokenCookie(w)
		RespondJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_REFRESH_TOKEN",
				Message: "invalid or expired refresh token",
			},
		})
	case errors.Is(err, usecase.ErrUserIdNotFound):
		clearRefreshTokenCookie(w)
		RespondJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "USER_NOT_FOUND",
				Message: "user not found",
			},
		})
	case errors.Is(err, domain.ErrUserDeactivated):
		clearRefreshTokenCookie(w)
		RespondJSON(w, http.StatusForbidden, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "USER_DEACTIVATED",
				Message: "profile is deactivated",
			},
		})
	default:
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INTERNAL_ERROR",
				Message: "refresh failed",
			},
		})
	}
}
