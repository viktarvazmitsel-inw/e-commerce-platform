package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shop/services/search/internal/search"
)

type SearchHandler struct {
	esClient *search.Client
}

func NewSearchHandler(esClient *search.Client) *SearchHandler {
	return &SearchHandler{
		esClient: esClient,
	}
}

type SearchResponse struct {
	Query   string        `json:"query"`
	Results []interface{} `json:"results"`
	Total   int           `json:"total"`
}

func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query().Get("q")
	results, total, err := h.esClient.Search(r.Context(), q)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(SearchResponse{
		Query:   q,
		Results: results,
		Total:   total,
	})
}
