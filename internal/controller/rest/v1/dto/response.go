package dto

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func SendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func SendError(w http.ResponseWriter, status int, errMsg string) {
	SendJSON(w, status, ErrorResponse{Error: errMsg})
}
