package web

import (
	"encoding/json"
	"net/http"
)

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}

func writeAPIError(writer http.ResponseWriter, status int, code string, message string, details map[string]any) {
	writeJSON(writer, status, apiError{
		Code:    code,
		Message: message,
		Details: details,
	})
}
