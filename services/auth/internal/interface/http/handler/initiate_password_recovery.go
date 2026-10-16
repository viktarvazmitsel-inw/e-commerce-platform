package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"authorization/internal/domain"
	"authorization/internal/usecase"
)

type InitiatePasswordRecoveryUseCase interface {
	Execute(ctx context.Context, input usecase.InitiatePasswordRecoveryInput) error
}

type InitiatePasswordRecoveryHandler struct {
	initiatePasswordRecovery InitiatePasswordRecoveryUseCase
}

type PasswordRecoveryBody struct {
	Email string `json:"email"`
}

func NewInitiatePasswordRecoveryHandler(uc InitiatePasswordRecoveryUseCase) *InitiatePasswordRecoveryHandler {
	return &InitiatePasswordRecoveryHandler{
		initiatePasswordRecovery: uc,
	}
}

func (h *InitiatePasswordRecoveryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	var passwordRecoveryBody PasswordRecoveryBody
	if !DecodeJSON(w, r, &passwordRecoveryBody) {
		return
	}
	passwordRecoveryBody.Email = strings.TrimSpace(passwordRecoveryBody.Email)
	passwordRecoveryBody.Email = strings.ToLower(passwordRecoveryBody.Email)
	if !domain.IsEmailValid(passwordRecoveryBody.Email) {
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_EMAIL",
				Message: "invalid email",
			},
		})
		return
	}

	initiatePasswordRecoveryInput := usecase.InitiatePasswordRecoveryInput{
		UserEmail: passwordRecoveryBody.Email,
	}

	if err := h.initiatePasswordRecovery.Execute(r.Context(), initiatePasswordRecoveryInput); err != nil {
		h.handleUsecaseError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, &Response{
		Success: true,
		Data:    map[string]string{"message": "check your email for password recovery instructions"},
	})
}

func (h *InitiatePasswordRecoveryHandler) handleUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrUserEmailNotFound),
		errors.Is(err, usecase.ErrEmptyEmail),
		errors.Is(err, domain.ErrUserDeactivated):
		RespondJSON(w, http.StatusOK, &Response{
			Success: true,
			Data:    map[string]string{"message": "check your email for password recovery instructions"},
		})
		return
	default:
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INTERNAL_SERVER_ERROR",
				Message: "failed to initiate password recovery",
			},
		})
	}
}
