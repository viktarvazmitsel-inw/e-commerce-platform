package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"authorization/internal/domain"
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

	var resendVerificationCreds ResendVerificationCredentials
	if !DecodeJSON(w, r, &resendVerificationCreds) {
		return
	}

	if !isResendCredentialsFormatOk(&resendVerificationCreds) {
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
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

	RespondJSON(w, http.StatusOK, &Response{
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
		RespondJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_CREDENTIALS",
				Message: "invalid credentials",
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
	case errors.Is(err, usecase.ErrEmailAlreadyVerified):
		RespondJSON(w, http.StatusConflict, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "USER_ALREADY_ACTIVATED",
				Message: "profile is already activated",
			},
		})
	default:
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INTERNAL_ERROR",
				Message: "resending verification message failed",
			},
		})
	}
}
