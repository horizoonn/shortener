package response

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func WriteError(w http.ResponseWriter, status int, message string, code string) {
	WriteJSON(w, status, Error{
		Message: message,
		Code:    code,
	})
}
