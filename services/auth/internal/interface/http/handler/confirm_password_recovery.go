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
	var body ConfirmPasswordRecoveryBody
	if !DecodeJSON(w, r, &body) {
		return
	}

	errorDetails := validateConfirmPasswordRecoveryBody(&body)
	if len(errorDetails) > 0 {
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
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

	response.RespondJSON(w, http.StatusOK, &response.Response{
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
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_PASSWORD_RECOVERY_TOKEN",
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
	case errors.Is(err, usecase.ErrPasswordIsNotStrongEnough):
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"new_password": err.Error(),
				},
			},
		})
	case errors.Is(err, usecase.ErrSameNewPassword):
		response.RespondJSON(w, http.StatusConflict, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "SAME_PASSWORD",
				Message: err.Error(),
				Details: map[string]string{
					"new_password": err.Error(),
				},
			},
		})
	case errors.Is(err, domain.ErrEmptyPassword):
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"new_password": err.Error(),
				},
			},
		})
	default:
		response.RespondJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "password recovery confirmation failed",
			},
		})
	}
}
