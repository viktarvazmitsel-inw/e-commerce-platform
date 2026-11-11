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
	var passwordRecoveryBody PasswordRecoveryBody
	if !DecodeJSON(w, r, &passwordRecoveryBody) {
		return
	}
	passwordRecoveryBody.Email = strings.TrimSpace(passwordRecoveryBody.Email)
	passwordRecoveryBody.Email = strings.ToLower(passwordRecoveryBody.Email)
	if !domain.IsEmailValid(passwordRecoveryBody.Email) {
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
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

	response.RespondJSON(w, http.StatusOK, &response.Response{
		Success: true,
		Data:    map[string]string{"message": "check your email for password recovery instructions"},
	})
}

func (h *InitiatePasswordRecoveryHandler) handleUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrUserEmailNotFound),
		errors.Is(err, usecase.ErrEmptyEmail),
		errors.Is(err, domain.ErrUserDeactivated):
		response.RespondJSON(w, http.StatusOK, &response.Response{
			Success: true,
			Data:    map[string]string{"message": "check your email for password recovery instructions"},
		})
		return
	default:
		response.RespondJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INTERNAL_SERVER_ERROR",
				Message: "failed to initiate password recovery",
			},
		})
	}
}
