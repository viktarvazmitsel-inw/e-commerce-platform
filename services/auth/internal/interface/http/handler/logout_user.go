package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"authorization/internal/interface/http/response"
	"authorization/internal/usecase"
)

type LogoutUserUseCase interface {
	Execute(ctx context.Context, input usecase.LogoutUserInput) error
}

type LogoutUserHandler struct {
	logoutUser LogoutUserUseCase
}

type LogoutCredentials struct {
	UserID string `json:"user_id"`
}

func NewLogoutUserHandler(uc LogoutUserUseCase) *LogoutUserHandler {
	return &LogoutUserHandler{
		logoutUser: uc,
	}
}

func (h *LogoutUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.RespondJSON(w, http.StatusMethodNotAllowed, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "METHOD_NOT_ALLOWED",
				Message: "method not allowed",
			},
		})
		return
	}

	cookie, err := r.Cookie(refreshTokenCookie)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_REFRESH_TOKEN",
				Message: "invalid or expired refresh token",
			},
		})
		return
	}

	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "UNAUTHORIZED",
				Message: "unauthorized",
			},
		})
		return
	}

	logoutInput := usecase.LogoutUserInput{
		UserID: userID,
		Token:  cookie.Value,
	}

	if err := h.logoutUser.Execute(r.Context(), logoutInput); err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	clearRefreshTokenCookie(w)

	response.RespondJSON(w, http.StatusOK, &response.Response{
		Success: true,
		Data: map[string]string{
			"message": "logout successful",
		},
	})
}

func isLogoutCredentialsFormatOk(creds LogoutCredentials) bool {
	return strings.TrimSpace(creds.UserID) != ""
}

func (h *LogoutUserHandler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrUserIdNotFound):
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "USER_NOT_FOUND",
				Message: "user not found",
			},
		})
	case errors.Is(err, usecase.ErrInvalidRefreshToken):
		clearRefreshTokenCookie(w)
		response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_REFRESH_TOKEN",
				Message: "invalid or expired refresh token",
			},
		})
	default:
		response.RespondJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "logout failed",
			},
		})
	}
}
