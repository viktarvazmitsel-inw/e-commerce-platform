package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"authorization/internal/interface/http/response"
)

const maxRequestBodySize = 1024 * 1024

func DecodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			response.RespondJSON(w, http.StatusRequestEntityTooLarge, &response.Response{
				Success: false,
				Errors: &response.APIError{
					Code:    "REQUEST_ENTITY_TOO_LARGE",
					Message: "oversized request",
				},
			})
			return false
		}
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_JSON",
				Message: "incorrect data format",
			},
		})
		return false
	}
	return true
}
