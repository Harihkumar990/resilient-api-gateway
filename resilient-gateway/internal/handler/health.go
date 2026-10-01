package handler 

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)


type HealthHandler struct {
	rdb *redis.Client
}

func NewHealthHandler(rdb *redis.Client) *HealthHandler {
	return &HealthHandler{rdb: rdb}
}

func (h *HealthHandler) Livez(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second )
	defer cancel()
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		w.Header().Set("Content-Type","application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string {
			"status": "unhealthy",
			"error" : "redis unreachable",
		})
		return
	}
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}