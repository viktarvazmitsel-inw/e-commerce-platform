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

func (h *GetUserProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "METHOD_NOT_ALLOWED",
				Message: "method not allowed",
			},
		})
		return
	}

	authorID := strings.TrimSpace(r.Context().Value("userID").(string))
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

	input := usecase.GetUserInput{
		AuthorID: authorID,
		TargetID: targetID,
	}

	profile, err := h.getUserProfile.Execute(r.Context(), input)
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, &Response{
		Success: true,
		Data:    profile,
	})
}

func (h *GetUserProfileHandler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrUserIdNotFound):
		RespondJSON(w, http.StatusNotFound, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "USER_NOT_FOUND",
				Message: "user not found",
			},
		})
	case errors.Is(err, usecase.ErrTargetUserNotFound):
		RespondJSON(w, http.StatusNotFound, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "USER_NOT_FOUND",
				Message: "target user not found",
			},
		})
	case errors.Is(err, domain.ErrUserDeactivated):
		RespondJSON(w, http.StatusForbidden, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "USER_DEACTIVATED",
				Message: "profile is deactivated",
			},
		})
	default:
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INTERNAL_ERROR",
				Message: "failed to retrieve user profile",
			},
		})
	}
}
