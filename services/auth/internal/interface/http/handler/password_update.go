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

type PasswordUpdateUseCase interface {
	Execute(ctx context.Context, input usecase.PasswordUpdateInput) (usecase.PasswordUpdateResult, error)
}

type PasswordUpdateHandler struct {
	passwordUpdate PasswordUpdateUseCase
}

type PasswordUpdateBody struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func NewPasswordUpdateHandler(uc PasswordUpdateUseCase) *PasswordUpdateHandler {
	return &PasswordUpdateHandler{
		passwordUpdate: uc,
	}
}

func (h *PasswordUpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		response.RespondJSON(w, http.StatusMethodNotAllowed, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "METHOD_NOT_ALLOWED",
				Message: "method not allowed",
			},
		})
		return
	}

	userIDVal, ok := r.Context().Value("userID").(string)
	if !ok {
		response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "UNAUTHORIZED_ACTION",
				Message: "user id is undefined",
			},
		})
		return
	}
	userID := strings.TrimSpace(userIDVal)
	if userID == "" {
		response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "UNAUTHORIZED_ACTION",
				Message: "user id is undefined",
			},
		})
		return
	}

	var body PasswordUpdateBody
	if !DecodeJSON(w, r, &body) {
		return
	}

	errorDetails := validatePasswordUpdateBody(&body)
	if len(errorDetails) > 0 {
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: "invalid input data",
				Details: errorDetails,
			},
		})
		return
	}

	result, err := h.passwordUpdate.Execute(r.Context(), usecase.PasswordUpdateInput{
		UserID:      userID,
		OldPassword: body.OldPassword,
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

func validatePasswordUpdateBody(body *PasswordUpdateBody) map[string]string {
	errorDetails := make(map[string]string)

	if len(body.OldPassword) < 8 || len(body.OldPassword) > 50 {
		errorDetails["old_password"] = "password must be between 8 and 50 characters"
	}

	if len(body.NewPassword) < 8 || len(body.NewPassword) > 50 {
		errorDetails["new_password"] = "password must be between 8 and 50 characters"
	}

	return errorDetails
}

func (h *PasswordUpdateHandler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch {
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
	case errors.Is(err, usecase.ErrInvalidPassword):
		response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_CREDENTIALS",
				Message: err.Error(),
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
				Message: "password update failed",
			},
		})
	}
}
