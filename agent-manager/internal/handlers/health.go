package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"agent-manager/internal/repository"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	redis *repository.RedisClient
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(redis *repository.RedisClient) *HealthHandler {
	return &HealthHandler{
		redis: redis,
	}
}

// Health handles GET /health
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"service":   "agent-manager",
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Ready handles GET /health/ready
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	checks := map[string]interface{}{
		"redis": h.checkRedis(r),
	}

	allHealthy := true
	for _, check := range checks {
		if checkMap, ok := check.(map[string]interface{}); ok {
			if status, exists := checkMap["status"]; exists && status != "ok" {
				allHealthy = false
				break
			}
		}
	}

	status := "ready"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "not ready"
		statusCode = http.StatusServiceUnavailable
	}

	response := map[string]interface{}{
		"status":    status,
		"timestamp": time.Now().UTC(),
		"checks":    checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// checkRedis performs a health check on Redis connection
func (h *HealthHandler) checkRedis(r *http.Request) map[string]interface{} {
	start := time.Now()
	err := h.redis.Ping(r.Context())
	duration := time.Since(start)

	if err != nil {
		return map[string]interface{}{
			"status":   "error",
			"error":    err.Error(),
			"duration": duration.String(),
		}
	}

	return map[string]interface{}{
		"status":   "ok",
		"duration": duration.String(),
	}
}