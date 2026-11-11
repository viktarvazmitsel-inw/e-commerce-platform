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

type GetUserProfileUseCase interface {
	Execute(ctx context.Context, input usecase.GetUserInput) (*usecase.UserProfile, error)
}

type GetUserProfileHandler struct {
	getUserProfile GetUserProfileUseCase
}

func NewGetUserProfileHandler(uc GetUserProfileUseCase) *GetUserProfileHandler {
	return &GetUserProfileHandler{
		getUserProfile: uc,
	}
}

func (h *GetUserProfileHandler) HandlerWithUserID(w http.ResponseWriter, r *http.Request, authorID string) {
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

	input := usecase.GetUserInput{
		AuthorID: authorID,
		TargetID: targetID,
	}

	profile, err := h.getUserProfile.Execute(r.Context(), input)
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	response.RespondJSON(w, http.StatusOK, &response.Response{
		Success: true,
		Data:    profile,
	})
}

func (h *GetUserProfileHandler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrUserIdNotFound):
		response.RespondJSON(w, http.StatusNotFound, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "USER_NOT_FOUND",
				Message: "user not found",
			},
		})
	case errors.Is(err, usecase.ErrTargetUserNotFound):
		response.RespondJSON(w, http.StatusNotFound, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "USER_NOT_FOUND",
				Message: "target user not found",
			},
		})
	case errors.Is(err, domain.ErrUserDeactivated):
		response.RespondJSON(w, http.StatusForbidden, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "USER_DEACTIVATED",
				Message: "profile is deactivated",
			},
		})
	default:
		response.RespondJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "failed to retrieve user profile",
			},
		})
	}
}
