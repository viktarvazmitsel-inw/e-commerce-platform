package handler

import (
	"authorization/internal/interface/http/response"
	"net/http"
)

type Healthcheck interface {
	IsHealthy() bool
}

type HealthcheckHandler struct {
	readyzHealthcheck Healthcheck
}

func NewHealthcheckHandler(readyzHealthcheck Healthcheck) *HealthcheckHandler {
	return &HealthcheckHandler{
		readyzHealthcheck: readyzHealthcheck,
	}
}

func (h *HealthcheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.readyzHealthcheck.IsHealthy() {
		response.RespondJSON(w, http.StatusServiceUnavailable, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "SERVICE_UNAVAILABLE",
				Message: "Service is not healthy",
			},
		})
		return
	}

	response.RespondJSON(w, http.StatusOK, &response.Response{
		Success: true,
		Data:    "OK",
	})
}
