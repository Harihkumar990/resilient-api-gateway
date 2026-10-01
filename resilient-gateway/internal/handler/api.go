package handler 

import (
	"encoding/json"
	"net/http"
)

type APIHandler struct {}

func NewAPIHandler() *APIHandler {
	return &APIHandler{}
}

func (h *APIHandler) HandleData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":"request processed successfully",
		"path":r.URL.Path,
	})
}