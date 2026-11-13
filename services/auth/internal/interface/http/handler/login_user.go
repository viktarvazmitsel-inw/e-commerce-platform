package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"authorization/internal/domain"
	"authorization/internal/interface/http/response"
	"authorization/internal/usecase"
)

type LoginUserUseCase interface {
	Execute(ctx context.Context, input usecase.LoginInput) (usecase.LoginOutput, error)
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
	var userCredentials UserCredentials
	if !DecodeJSON(w, r, &userCredentials) {
		return
	}

	if !isCredentialsFormatOk(&userCredentials) {
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
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

	loginOutput, err := h.loginUser.Execute(r.Context(), loginInput)
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	setRefreshTokenCookie(w, loginOutput.TokenPair)

	response.RespondJSON(w, http.StatusOK, &response.Response{
		Success: true,
		Data: map[string]string{
			"message":      "login successful",
			"access_token": loginOutput.TokenPair.AccessToken,
			"user_id":      loginOutput.UserInfo.UserID,
			"role":         strconv.Itoa(int(loginOutput.UserInfo.Role)),
		},
	})
}

func isCredentialsFormatOk(uCreds *UserCredentials) bool {
	uCreds.Email = strings.TrimSpace(uCreds.Email)
	uCreds.Email = strings.ToLower(uCreds.Email)
	if !domain.IsEmailValid(uCreds.Email) {
		return false
	}
	if len(uCreds.Password) < 8 || len(uCreds.Password) > 50 {
		return false
	}

	return true
}

func (h *LoginUserHandler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidCredentials), errors.Is(err, domain.ErrUserDeactivated):
		response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_CREDENTIALS",
				Message: "invalid email or password",
			},
		})
	case errors.Is(err, usecase.ErrUserEmailIsNotVerified):
		response.RespondJSON(w, http.StatusForbidden, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "UNVERIFIED_EMAIL",
				Message: "user email is not verified",
			},
		})
	default:
		response.RespondJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "login failed",
			},
		})
	}
}
