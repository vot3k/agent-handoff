package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"agent-manager/internal/middleware"
	"agent-manager/internal/models"
	"agent-manager/internal/service"
)

// HandoffHandler handles HTTP requests for handoff operations
type HandoffHandler struct {
	service service.HandoffServiceInterface
}

// NewHandoffHandler creates a new handoff handler
func NewHandoffHandler(service service.HandoffServiceInterface) *HandoffHandler {
	return &HandoffHandler{
		service: service,
	}
}

// CreateHandoff handles POST /api/v1/handoffs
func (h *HandoffHandler) CreateHandoff(w http.ResponseWriter, r *http.Request) {
	var req models.CreateHandoffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, http.StatusBadRequest, "Invalid JSON payload", err)
		return
	}

	handoff, err := h.service.CreateHandoff(r.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "validation failed") {
			h.writeError(w, r, http.StatusBadRequest, "Validation failed", err)
		} else {
			h.writeError(w, r, http.StatusInternalServerError, "Failed to create handoff", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(handoff)
}

// GetHandoff handles GET /api/v1/handoffs/{id}
func (h *HandoffHandler) GetHandoff(w http.ResponseWriter, r *http.Request) {
	handoffID := r.PathValue("id")
	if handoffID == "" {
		h.writeError(w, r, http.StatusBadRequest, "Missing handoff ID", nil)
		return
	}

	handoff, err := h.service.GetHandoff(r.Context(), handoffID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, r, http.StatusNotFound, "Handoff not found", err)
		} else {
			h.writeError(w, r, http.StatusInternalServerError, "Failed to get handoff", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handoff)
}

// ListHandoffs handles GET /api/v1/handoffs
func (h *HandoffHandler) ListHandoffs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	projectName := r.URL.Query().Get("project")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	response, err := h.service.ListHandoffs(r.Context(), projectName, page, pageSize)
	if err != nil {
		h.writeError(w, r, http.StatusInternalServerError, "Failed to list handoffs", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateStatus handles PUT /api/v1/handoffs/{id}/status
func (h *HandoffHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	handoffID := r.PathValue("id")
	if handoffID == "" {
		h.writeError(w, r, http.StatusBadRequest, "Missing handoff ID", nil)
		return
	}

	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, http.StatusBadRequest, "Invalid JSON payload", err)
		return
	}

	if err := h.service.UpdateStatus(r.Context(), handoffID, req.Status); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, r, http.StatusNotFound, "Handoff not found", err)
		} else if strings.Contains(err.Error(), "invalid transition") {
			h.writeError(w, r, http.StatusBadRequest, "Invalid status transition", err)
		} else {
			h.writeError(w, r, http.StatusInternalServerError, "Failed to update status", err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListQueues handles GET /api/v1/queues
func (h *HandoffHandler) ListQueues(w http.ResponseWriter, r *http.Request) {
	projectName := r.URL.Query().Get("project")

	queues, err := h.service.GetQueues(r.Context(), projectName)
	if err != nil {
		h.writeError(w, r, http.StatusInternalServerError, "Failed to get queues", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"queues": queues,
		"count":  len(queues),
	})
}

// GetQueueDepth handles GET /api/v1/queues/{queue}/depth
func (h *HandoffHandler) GetQueueDepth(w http.ResponseWriter, r *http.Request) {
	queueName := r.PathValue("queue")
	if queueName == "" {
		h.writeError(w, r, http.StatusBadRequest, "Missing queue name", nil)
		return
	}

	depth, err := h.service.GetQueueDepth(r.Context(), queueName)
	if err != nil {
		h.writeError(w, r, http.StatusInternalServerError, "Failed to get queue depth", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"queue_name": queueName,
		"depth":      depth,
	})
}

// writeError writes an error response in a consistent format
func (h *HandoffHandler) writeError(w http.ResponseWriter, r *http.Request, statusCode int, message string, err error) {
	requestID := middleware.GetRequestID(r.Context())
	
	response := map[string]interface{}{
		"error":      message,
		"request_id": requestID,
		"status":     statusCode,
	}

	if err != nil {
		response["details"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}