package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"authorization/internal/domain"
	"authorization/internal/usecase"
)

type ConfirmPasswordRecoveryUseCase interface {
	Execute(ctx context.Context, input usecase.ConfirmPasswordRecoveryInput) (usecase.ConfirmPasswordRecoveryResult, error)
}

type ConfirmPasswordRecoveryHandler struct {
	confirmPasswordRecovery ConfirmPasswordRecoveryUseCase
}

type ConfirmPasswordRecoveryBody struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func NewConfirmPasswordRecoveryHandler(uc ConfirmPasswordRecoveryUseCase) *ConfirmPasswordRecoveryHandler {
	return &ConfirmPasswordRecoveryHandler{
		confirmPasswordRecovery: uc,
	}
}

func (h *ConfirmPasswordRecoveryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	var body ConfirmPasswordRecoveryBody
	if !DecodeJSON(w, r, &body) {
		return
	}

	errorDetails := validateConfirmPasswordRecoveryBody(&body)
	if len(errorDetails) > 0 {
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: "invalid password recovery data",
				Details: errorDetails,
			},
		})
		return
	}

	result, err := h.confirmPasswordRecovery.Execute(r.Context(), usecase.ConfirmPasswordRecoveryInput{
		Token:       body.Token,
		NewPassword: body.NewPassword,
	})
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, &Response{
		Success: true,
		Data: map[string]any{
			"message":         "password updated successfully",
			"session_revoked": result.SessionRevoked,
		},
	})
}

func validateConfirmPasswordRecoveryBody(body *ConfirmPasswordRecoveryBody) map[string]string {
	errorDetails := make(map[string]string)

	body.Token = strings.TrimSpace(body.Token)
	if body.Token == "" {
		errorDetails["token"] = "invalid password recovery token"
	}

	if len(body.NewPassword) < 8 || len(body.NewPassword) > 50 {
		errorDetails["new_password"] = "password must be between 8 and 50 characters"
	}

	return errorDetails
}

func (h *ConfirmPasswordRecoveryHandler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidPasswordRecoveryToken):
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_PASSWORD_RECOVERY_TOKEN",
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
	case errors.Is(err, usecase.ErrPasswordIsNotStrongEnough):
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"new_password": err.Error(),
				},
			},
		})
	case errors.Is(err, usecase.ErrSameNewPassword):
		RespondJSON(w, http.StatusConflict, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "SAME_PASSWORD",
				Message: err.Error(),
				Details: map[string]string{
					"new_password": err.Error(),
				},
			},
		})
	case errors.Is(err, domain.ErrEmptyPassword):
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"new_password": err.Error(),
				},
			},
		})
	default:
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INTERNAL_ERROR",
				Message: "password recovery confirmation failed",
			},
		})
	}
}
