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
	UserID       string `json:"user_id"`
	RefreshToken string `json:"refresh_token"`
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

	var creds LogoutCredentials
	if !DecodeJSON(w, r, &creds) {
		return
	}

	if !isLogoutCredentialsFormatOk(creds) {
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_PARAMETERS",
				Message: "user_id and token are required",
			},
		})
		return
	}

	logoutInput := usecase.LogoutUserInput{
		UserID: strings.TrimSpace(creds.UserID),
		Token:  strings.TrimSpace(creds.RefreshToken),
	}

	if err := h.logoutUser.Execute(r.Context(), logoutInput); err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	response.RespondJSON(w, http.StatusOK, &response.Response{
		Success: true,
		Data: map[string]string{
			"message": "logout successful",
		},
	})
}

func isLogoutCredentialsFormatOk(creds LogoutCredentials) bool {
	if strings.TrimSpace(creds.UserID) == "" {
		return false
	}
	if strings.TrimSpace(creds.RefreshToken) == "" {
		return false
	}
	return true
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
