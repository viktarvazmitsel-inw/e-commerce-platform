package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/shop/services/notification/internal/interface/http/response"
	"github.com/shop/services/notification/internal/usecase"
)

type GetNotificationUseCase interface {
	Execute(ctx context.Context, input usecase.GetNotificationInput) (*usecase.GetNotificationOutput, error)
}

type GetNotificationHandler struct {
	uc GetNotificationUseCase
}

func NewGetNotificationHandler(uc GetNotificationUseCase) *GetNotificationHandler {
	return &GetNotificationHandler{uc: uc}
}

func (h *GetNotificationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		response.ResponseJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_QUERY_PARAMS",
				Message: "id is required",
			},
		})
	}

	output, err := h.uc.Execute(r.Context(), usecase.GetNotificationInput{
		NotificationID: id,
	})
	if err != nil {

		switch {
		case errors.Is(err, usecase.ErrNotificationNotFound):
			response.ResponseJSON(w, http.StatusNotFound, &response.Response{
				Success: false,
				Errors: &response.APIError{
					Code:    "NOT_FOUND",
					Message: "Notification not found",
				},
			})
		default:
			response.ResponseJSON(w, http.StatusInternalServerError, &response.Response{
				Success: false,
				Errors: &response.APIError{
					Code:    "INTERNAL_SERVER_ERROR",
					Message: "Internal server error",
				},
			})
		}
		return
	}

	response.ResponseJSON(w, http.StatusOK, &response.Response{
		Success: true,
		Data:    output,
	})
}
