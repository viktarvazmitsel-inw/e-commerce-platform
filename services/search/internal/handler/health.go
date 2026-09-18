package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shop/services/search/internal/broker"
	"github.com/shop/services/search/internal/search"
)

type HealthResponse struct {
	Status       string `json:"status"`
	Service      string `json:"service"`
	SearchEngine string `json:"search_engine"`
	Broker       string `json:"broker"`
}

type HealthHandler struct {
	esClient     *search.Client
	brokerClient *broker.Client
}

func NewHealthHandler(esClient *search.Client, brokerClient *broker.Client) *HealthHandler {
	return &HealthHandler{
		esClient:     esClient,
		brokerClient: brokerClient,
	}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	esConnected := h.esClient != nil && h.esClient.Ping(r.Context())
	brokerConnected := h.brokerClient != nil && h.brokerClient.IsConnected()

	esStatus := "disconnected"
	if esConnected {
		esStatus = "connected"
	}

	brokerStatus := "disconnected"
	if brokerConnected {
		brokerStatus = "connected"
	}

	status := "ok"
	statusCode := http.StatusOK

	if !esConnected || !brokerConnected {
		status = "degraded"
		if !esConnected && !brokerConnected {
			status = "down"
		}
		statusCode = http.StatusServiceUnavailable
	}

	resp := HealthResponse{
		Status:       status,
		Service:      "search-service",
		SearchEngine: esStatus,
		Broker:       brokerStatus,
	}

	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(resp)
}
