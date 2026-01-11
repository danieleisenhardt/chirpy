package main

import (
	"encoding/json"
	"net/http"
)

type errorResponse = struct {
	Message string `json:"message"`
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	responseBody, _ := json.Marshal(errorResponse{Message: msg})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(responseBody)
}
