package handler

import (
	"encoding/json"
	"errors"
	"net/http"
)

const maxRequestBodySize = 1024 * 1024

func DecodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondJSON(w, http.StatusRequestEntityTooLarge, &Response{
				Success: false,
				Errors: &APIError{
					Code:    "REQUEST_ENTITY_TOO_LARGE",
					Message: "oversized request",
				},
			})
			return false
		}
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_JSON",
				Message: "incorrect data format",
			},
		})
		return false
	}
	return true
}
