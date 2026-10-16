package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"authorization/internal/domain"
	"authorization/internal/usecase"
)

type ConfirmEmailUpdateUseCase interface {
	Execute(ctx context.Context, input usecase.ConfirmEmailUpdateInput) (usecase.ConfirmEmailUpdateResult, error)
}

type ConfirmEmailUpdateHandler struct {
	confirmEmailUpdate ConfirmEmailUpdateUseCase
}

func NewConfirmEmailUpdateHandler(uc ConfirmEmailUpdateUseCase) *ConfirmEmailUpdateHandler {
	return &ConfirmEmailUpdateHandler{
		confirmEmailUpdate: uc,
	}
}

func (h *ConfirmEmailUpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	var tokenData TokenData
	if !DecodeJSON(w, r, &tokenData) {
		return
	}

	token := strings.TrimSpace(tokenData.Token)
	if token == "" {
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_UPDATE_TOKEN",
				Message: "invalid update token",
				Details: map[string]string{
					"token": "invalid update token",
				},
			},
		})
		return
	}

	result, err := h.confirmEmailUpdate.Execute(r.Context(), usecase.ConfirmEmailUpdateInput{
		Token: token,
	})
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, &Response{
		Success: true,
		Data: map[string]any{
			"message":         "email updated successfully",
			"session_revoked": result.SessionRevoked,
		},
	})
}

func (h *ConfirmEmailUpdateHandler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidUpdateEmailToken):
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_UPDATE_TOKEN",
				Message: err.Error(),
				Details: map[string]string{
					"token": err.Error(),
				},
			},
		})
	case errors.Is(err, usecase.ErrUserIdNotFound):
		RespondJSON(w, http.StatusNotFound, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "USER_NOT_FOUND",
				Message: err.Error(),
			},
		})
	case errors.Is(err, domain.ErrUserDeactivated):
		RespondJSON(w, http.StatusForbidden, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "DEACTIVATED_USER",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrEmailAlreadyTaken):
		RespondJSON(w, http.StatusConflict, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "EMAIL_ALREADY_TAKEN",
				Message: err.Error(),
			},
		})
	case errors.Is(err, domain.ErrInvalidEmail):
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"new_email": err.Error(),
				},
			},
		})
	default:
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INTERNAL_ERROR",
				Message: "email update confirmation failed",
			},
		})
	}
}
