package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"authorization/internal/domain"
	"authorization/internal/interface/http/response"
	"authorization/internal/usecase"
)

type ResendVerificationEmailUseCase interface {
	Execute(ctx context.Context, input usecase.ResendVerificationEmailInput) error
}

type ResendVerificationEmailHandler struct {
	resendEmail ResendVerificationEmailUseCase
}

type ResendVerificationCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewResendVerificationEmailHandler(uc ResendVerificationEmailUseCase) *ResendVerificationEmailHandler {
	return &ResendVerificationEmailHandler{
		resendEmail: uc,
	}
}

func (h *ResendVerificationEmailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var resendVerificationCreds ResendVerificationCredentials
	if !DecodeJSON(w, r, &resendVerificationCreds) {
		return
	}

	if !isResendCredentialsFormatOk(&resendVerificationCreds) {
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_CREDENTIALS",
				Message: "invalid credentials",
			},
		})
		return
	}

	resendVerificationInput := usecase.ResendVerificationEmailInput{
		Email:    resendVerificationCreds.Email,
		Password: resendVerificationCreds.Password,
	}

	if err := h.resendEmail.Execute(r.Context(), resendVerificationInput); err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	response.RespondJSON(w, http.StatusOK, &response.Response{
		Success: true,
		Data: map[string]string{
			"message": "verification email sent successfully",
		},
	})

}

func isResendCredentialsFormatOk(resendVerificationCreds *ResendVerificationCredentials) bool {
	resendVerificationCreds.Email = strings.TrimSpace(resendVerificationCreds.Email)
	resendVerificationCreds.Email = strings.ToLower(resendVerificationCreds.Email)
	if !domain.IsEmailValid(resendVerificationCreds.Email) {
		return false
	}
	if len(resendVerificationCreds.Password) < 8 || len(resendVerificationCreds.Password) > 50 {
		return false
	}

	return true
}

func (h *ResendVerificationEmailHandler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrEmptyEmail), errors.Is(err, usecase.ErrInvalidCredentials):
		response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_CREDENTIALS",
				Message: "invalid credentials",
			},
		})
	case errors.Is(err, domain.ErrUserDeactivated):
		response.RespondJSON(w, http.StatusForbidden, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "USER_DEACTIVATED",
				Message: "profile is deactivated",
			},
		})
	case errors.Is(err, usecase.ErrEmailAlreadyVerified):
		response.RespondJSON(w, http.StatusConflict, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "USER_ALREADY_ACTIVATED",
				Message: "profile is already activated",
			},
		})
	default:
		response.RespondJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "resending verification message failed",
			},
		})
	}
}
