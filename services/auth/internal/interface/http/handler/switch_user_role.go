package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"authorization/internal/domain"
	"authorization/internal/interface/http/response"
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

func (h *SwitchUserRoleHandler) HandlerWithUserID(w http.ResponseWriter, r *http.Request, authorID string) {
	targetID := strings.TrimSpace(chi.URLParam(r, "userID"))
	if targetID == "" {
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
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
		response.RespondJSON(w, http.StatusOK, &response.Response{
			Success: true,
			Data: map[string]any{
				"message":         "role updated, but session revocation failed",
				"target_id":       result.TargetID,
				"session_revoked": false,
			},
		})
		return
	}

	response.RespondJSON(w, http.StatusOK, &response.Response{
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
		response.RespondJSON(w, http.StatusNotFound, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "USER_NOT_FOUND",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrTargetUserNotFound):
		response.RespondJSON(w, http.StatusNotFound, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "TARGET_USER_NOT_FOUND",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrUnauthorizedAction), errors.Is(err, domain.ErrPermissionDenied):
		response.RespondJSON(w, http.StatusForbidden, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "FORBIDDEN",
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
	case errors.Is(err, domain.ErrUnableToSetAdmin), errors.Is(err, domain.ErrInvalidRole):
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_ROLE",
				Message: err.Error(),
			},
		})
	case errors.Is(err, domain.ErrUnchangedRole):
		response.RespondJSON(w, http.StatusConflict, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "UNCHANGED_ROLE",
				Message: err.Error(),
			},
		})
	default:
		response.RespondJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "role switch failed",
			},
		})
	}
}
