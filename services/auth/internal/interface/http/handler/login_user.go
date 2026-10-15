package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"authorization/internal/domain"
	"authorization/internal/usecase"
)

type LoginUserUseCase interface {
	Execute(ctx context.Context, input usecase.LoginInput) (domain.TokenPair, error)
}

type LoginUserHandler struct {
	loginUser LoginUserUseCase
}

type UserCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewLoginUserHandler(uc LoginUserUseCase) *LoginUserHandler {
	return &LoginUserHandler{
		loginUser: uc,
	}
}

func (h *LoginUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	var userCredentials UserCredentials
	if !DecodeJSON(w, r, &userCredentials) {
		return
	}

	if !isCredentialsFormatOk(userCredentials) {
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_CREDENTIALS",
				Message: "invalid credentials",
			},
		})
		return
	}

	loginInput := usecase.LoginInput{
		Email:    strings.TrimSpace(userCredentials.Email),
		Password: userCredentials.Password,
	}

	tokenPair, err := h.loginUser.Execute(r.Context(), loginInput)
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, &Response{
		Success: true,
		Data: map[string]string{
			"message":       "login successful",
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
		},
	})
}

func isCredentialsFormatOk(uCreds UserCredentials) bool {
	trimmedEmail := strings.TrimSpace(uCreds.Email)
	if trimmedEmail == "" || !domain.IsEmailValid(trimmedEmail) {
		return false
	}
	if len(uCreds.Password) < 8 || len(uCreds.Password) > 50 {
		return false
	}

	return true
}

func (h *LoginUserHandler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidCredentials):
		RespondJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_CREDENTIALS",
				Message: "invalid email or password",
			},
		})
	case errors.Is(err, usecase.ErrUserEmailIsNotVerified):
		RespondJSON(w, http.StatusForbidden, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "UNVERIFIED_EMAIL",
				Message: "user email is not verified",
			},
		})
	case errors.Is(err, domain.ErrUserDeactivated):
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
				Message: "login failed",
			},
		})
	}
}
