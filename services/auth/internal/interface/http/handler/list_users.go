package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"authorization/internal/domain"
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

func (h *ListUsersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
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

	RespondJSON(w, http.StatusOK, &Response{
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
		RespondJSON(w, http.StatusNotFound, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "USER_NOT_FOUND",
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
	default:
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INTERNAL_ERROR",
				Message: "failed to load users",
			},
		})
	}
}
