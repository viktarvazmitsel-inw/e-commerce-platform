package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"authorization/internal/domain"
	"authorization/internal/interface/http/response"
	"authorization/internal/usecase"
)

type ListUsersUseCase interface {
	Execute(ctx context.Context, input usecase.ListUsersInput) (*usecase.ListUsersOutput, error)
}

type ListUsersHandler struct {
	listUsers ListUsersUseCase
}

func NewListUsersHandler(uc ListUsersUseCase) *ListUsersHandler {
	return &ListUsersHandler{
		listUsers: uc,
	}
}

func (h *ListUsersHandler) HandlerWithUserID(w http.ResponseWriter, r *http.Request, authorID string) {
	query := r.URL.Query()
	page, pageOK := parseOptionalIntQuery(query.Get("page"))
	quantity, quantityOK := parseOptionalIntQuery(query.Get("quantity"))
	if !pageOK || !quantityOK {
		details := make(map[string]string)
		if !pageOK {
			details["page"] = "page must be an integer"
		}
		if !quantityOK {
			details["quantity"] = "quantity must be an integer"
		}
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_PARAMETERS",
				Message: "invalid query parameters",
				Details: details,
			},
		})
		return
	}

	output, err := h.listUsers.Execute(r.Context(), usecase.ListUsersInput{
		AuthorID: authorID,
		Quantity: quantity,
		Page:     page,
	})
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	response.RespondJSON(w, http.StatusOK, &response.Response{
		Success: true,
		Data:    output,
	})
}

func parseOptionalIntQuery(raw string) (int, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, true
	}

	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, false
	}

	return value, true
}

func (h *ListUsersHandler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrUserIdNotFound):
		response.RespondJSON(w, http.StatusNotFound, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "USER_NOT_FOUND",
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
	default:
		response.RespondJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "failed to load users",
			},
		})
	}
}
