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

func (h *ConfirmEmailUpdateHandler) HandlerWithUserID(w http.ResponseWriter, r *http.Request, authorID string) {
	var tokenData TokenData
	if !DecodeJSON(w, r, &tokenData) {
		return
	}

	token := strings.TrimSpace(tokenData.Token)
	if token == "" {
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_UPDATE_TOKEN",
				Message: "invalid update token",
			},
		})
		return
	}

	result, err := h.confirmEmailUpdate.Execute(r.Context(), usecase.ConfirmEmailUpdateInput{
		AuthorID: strings.TrimSpace(authorID),
		Token:    token,
	})
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	response.RespondJSON(w, http.StatusOK, &response.Response{
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
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_UPDATE_TOKEN",
				Message: err.Error(),
				Details: map[string]string{
					"token": err.Error(),
				},
			},
		})
	case errors.Is(err, usecase.ErrUserIdNotFound):
		response.RespondJSON(w, http.StatusNotFound, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "USER_NOT_FOUND",
				Message: err.Error(),
			},
		})
	case errors.Is(err, domain.ErrUserDeactivated):
		response.RespondJSON(w, http.StatusForbidden, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "DEACTIVATED_USER",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrEmailAlreadyTaken):
		response.RespondJSON(w, http.StatusConflict, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "EMAIL_ALREADY_TAKEN",
				Message: err.Error(),
			},
		})
	case errors.Is(err, domain.ErrInvalidEmail):
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"new_email": err.Error(),
				},
			},
		})
	case errors.Is(err, usecase.ErrUnauthorizedAction):
		response.RespondJSON(w, http.StatusForbidden, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "FORBIDDEN",
				Message: err.Error(),
			},
		})
	default:
		response.RespondJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "email update confirmation failed",
			},
		})
	}
}
