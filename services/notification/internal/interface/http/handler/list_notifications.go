package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/shop/services/notification/internal/interface/http/response"
	"github.com/shop/services/notification/internal/usecase"
)

type ListNotificationsUseCase interface {
	Execute(ctx context.Context, input usecase.ListNotificationsInput) (*usecase.ListNotificationsOutput, error)
}

type ListNotificationsHandler struct {
	uc ListNotificationsUseCase
}

func NewListNotificationsHandler(uc ListNotificationsUseCase) *ListNotificationsHandler {
	return &ListNotificationsHandler{uc: uc}
}

func (h *ListNotificationsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	qurty := r.URL.Query()
	page, pageOK := parseOptionalIntQuery(qurty.Get("page"))
	quantity, quantityOK := parseOptionalIntQuery(qurty.Get("quantity"))
	if !pageOK || !quantityOK {
		details := make(map[string]string)
		if !pageOK {
			details["page"] = "page must be a positive integer"
		}
		if !quantityOK {
			details["quantity"] = "quantity must be a positive integer"
		}

		response.ResponseJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_QUERY_PARAMS",
				Message: "page and quantity must be positive integers",
				Details: details,
			},
		})
		return
	}

	output, err := h.uc.Execute(r.Context(), usecase.ListNotificationsInput{
		Page:     page,
		Quantity: quantity,
	})
	if err != nil {
		response.ResponseJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INTERNAL_SERVER_ERROR",
				Message: "Internal server error",
			},
		})
		return
	}

	response.ResponseJSON(w, http.StatusOK, &response.Response{
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
