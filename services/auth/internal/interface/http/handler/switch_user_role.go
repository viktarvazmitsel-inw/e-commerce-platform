package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"authorization/internal/domain"
	"authorization/internal/usecase"

	"github.com/go-chi/chi/v5"
)

type SwitchUserRoleUseCase interface {
	Execute(ctx context.Context, input usecase.SwitchUserRoleInput) (usecase.SwitchUserRoleResult, error)
}

type SwitchUserRoleHandler struct {
	switchRole SwitchUserRoleUseCase
}

func NewSwitchUserRoleHandler(uc SwitchUserRoleUseCase) *SwitchUserRoleHandler {
	return &SwitchUserRoleHandler{
		switchRole: uc,
	}
}

func (h *SwitchUserRoleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		RespondJSON(w, http.StatusMethodNotAllowed, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "METHOD_NOT_ALLOWED",
				Message: "method not allowed",
			},
		})
		return
	}

	authorIDVal, ok := r.Context().Value("userID").(string)
	if !ok {
		RespondJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "UNAUTHORIZED_ACTION",
				Message: "author id is undefined",
			},
		})
		return
	}
	authorID := strings.TrimSpace(authorIDVal)
	if authorID == "" {
		RespondJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "UNAUTHORIZED_ACTION",
				Message: "author id is undefined",
			},
		})
		return
	}

	targetID := strings.TrimSpace(chi.URLParam(r, "userID"))
	if targetID == "" {
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_PARAMETERS",
				Message: "user id is required",
			},
		})
		return
	}

	type switchRoleBody struct {
		NewRole uint8 `json:"new_role"`
	}
	var body switchRoleBody
	if !DecodeJSON(w, r, &body) {
		return
	}

	newRole := domain.Role(body.NewRole)
	if newRole != domain.RoleClient && newRole != domain.RoleAnalyst {
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: "invalid role value",
				Details: map[string]string{
					"new_role": "must be a valid non-admin role",
				},
			},
		})
		return
	}

	input := usecase.SwitchUserRoleInput{
		AuthorID: authorID,
		TargetID: targetID,
		NewRole:  newRole,
	}

	result, err := h.switchRole.Execute(r.Context(), input)
	if err != nil {
		h.handleUsecaseError(w, err)
		return
	}

	if !result.SessionRevoked {
		RespondJSON(w, http.StatusOK, &Response{
			Success: true,
			Data: map[string]any{
				"message":         "role updated, but session revocation failed",
				"target_id":       result.TargetID,
				"session_revoked": false,
			},
		})
		return
	}

	RespondJSON(w, http.StatusOK, &Response{
		Success: true,
		Data: map[string]any{
			"message":         "role updated successfully",
			"target_id":       result.TargetID,
			"session_revoked": true,
		},
	})
}

func (h *SwitchUserRoleHandler) handleUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrUserIdNotFound):
		RespondJSON(w, http.StatusNotFound, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "USER_NOT_FOUND",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrTargetUserNotFound):
		RespondJSON(w, http.StatusNotFound, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "TARGET_USER_NOT_FOUND",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrUnauthorizedAction), errors.Is(err, domain.ErrPermissionDenied):
		RespondJSON(w, http.StatusForbidden, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "FORBIDDEN",
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
	case errors.Is(err, domain.ErrUnableToSetAdmin):
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_ROLE",
				Message: err.Error(),
			},
		})
	case errors.Is(err, domain.ErrUnchangedRole):
		RespondJSON(w, http.StatusConflict, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "UNCHANGED_ROLE",
				Message: err.Error(),
			},
		})
	default:
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INTERNAL_ERROR",
				Message: "role switch failed",
			},
		})
	}
}
