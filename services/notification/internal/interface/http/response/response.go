package response

import (
	"encoding/json/v2"
	"net/http"
)

type Response struct {
	Success bool      `json:"success"`
	Data    any       `json:"data"`
	Errors  *APIError `json:"errors"`
}

type APIError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details"`
}

func ResponseJSON(w http.ResponseWriter, status int, response *Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.MarshalWrite(w, response)
}
