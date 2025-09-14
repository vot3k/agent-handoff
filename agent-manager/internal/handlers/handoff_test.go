package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/models"
)

// MockHandoffService is a mock implementation of the handoff service for testing
type MockHandoffService struct {
	handoffs map[string]*models.Handoff
}

func NewMockHandoffService() *MockHandoffService {
	return &MockHandoffService{
		handoffs: make(map[string]*models.Handoff),
	}
}

func (m *MockHandoffService) CreateHandoff(ctx context.Context, req *models.CreateHandoffRequest) (*models.Handoff, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	now := time.Now()
	handoff := &models.Handoff{
		Metadata: models.HandoffMetadata{
			ProjectName: req.ProjectName,
			FromAgent:   req.FromAgent,
			ToAgent:     req.ToAgent,
			Timestamp:   now,
			TaskContext: req.TaskContext,
			Priority:    req.Priority,
			HandoffID:   "test-handoff-123",
		},
		Content: models.HandoffContent{
			Summary:          req.Summary,
			Requirements:     req.Requirements,
			Artifacts:        req.Artifacts,
			TechnicalDetails: req.TechnicalDetails,
			NextSteps:        req.NextSteps,
		},
		Status:    models.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.handoffs[handoff.Metadata.HandoffID] = handoff
	return handoff, nil
}

func (m *MockHandoffService) GetHandoff(ctx context.Context, handoffID string) (*models.Handoff, error) {
	if handoff, exists := m.handoffs[handoffID]; exists {
		return handoff, nil
	}
	return nil, fmt.Errorf("handoff not found: %s", handoffID)
}

func (m *MockHandoffService) ListHandoffs(ctx context.Context, projectName string, page, pageSize int) (*models.HandoffListResponse, error) {
	var handoffs []models.Handoff
	for _, handoff := range m.handoffs {
		if projectName == "" || handoff.Metadata.ProjectName == projectName {
			handoffs = append(handoffs, *handoff)
		}
	}

	return &models.HandoffListResponse{
		Handoffs:   handoffs,
		TotalCount: len(handoffs),
		Page:       page,
		PageSize:   pageSize,
		HasMore:    false,
	}, nil
}

func (m *MockHandoffService) UpdateStatus(ctx context.Context, handoffID string, status models.HandoffStatus) error {
	if handoff, exists := m.handoffs[handoffID]; exists {
		handoff.Status = status
		handoff.UpdatedAt = time.Now()
		return nil
	}
	return fmt.Errorf("handoff not found: %s", handoffID)
}

func (m *MockHandoffService) GetQueues(ctx context.Context, projectName string) ([]models.QueueInfo, error) {
	return []models.QueueInfo{
		{
			QueueName:   "handoff:project:test:queue:golang-expert",
			ProjectName: "test",
			AgentName:   "golang-expert",
			Depth:       2,
		},
	}, nil
}

func (m *MockHandoffService) GetQueueDepth(ctx context.Context, queueName string) (int64, error) {
	return 2, nil
}

func (m *MockHandoffService) ProcessNextHandoff(ctx context.Context, queueName string) (*models.Handoff, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

func (m *MockHandoffService) CompleteHandoff(ctx context.Context, handoffID string) error {
	return m.UpdateStatus(ctx, handoffID, models.StatusCompleted)
}

func (m *MockHandoffService) FailHandoff(ctx context.Context, handoffID string) error {
	return m.UpdateStatus(ctx, handoffID, models.StatusFailed)
}

func (m *MockHandoffService) CancelHandoff(ctx context.Context, handoffID string) error {
	return m.UpdateStatus(ctx, handoffID, models.StatusCancelled)
}

func TestHandoffHandler_CreateHandoff(t *testing.T) {
	tests := []struct {
		name           string
		payload        models.CreateHandoffRequest
		expectedStatus int
	}{
		{
			name: "valid handoff creation",
			payload: models.CreateHandoffRequest{
				ProjectName: "test-project",
				FromAgent:   "api-expert",
				ToAgent:     "golang-expert",
				TaskContext: "test-context",
				Priority:    models.PriorityNormal,
				Summary:     "Test handoff",
				Requirements: []string{"requirement1", "requirement2"},
				Artifacts: map[string][]string{
					"created": {"file1.go", "file2.go"},
				},
				TechnicalDetails: map[string]interface{}{
					"language": "go",
				},
				NextSteps: []string{"step1", "step2"},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing required fields",
			payload: models.CreateHandoffRequest{
				ProjectName: "",
				FromAgent:   "api-expert",
				ToAgent:     "golang-expert",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := NewMockHandoffService()
			handler := NewHandoffHandler(mockService)

			payload, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/api/v1/handoffs", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			handler.CreateHandoff(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedStatus == http.StatusCreated {
				var response models.Handoff
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Errorf("failed to decode response: %v", err)
				}

				if response.Metadata.ProjectName != tt.payload.ProjectName {
					t.Errorf("expected project name %s, got %s", tt.payload.ProjectName, response.Metadata.ProjectName)
				}
			}
		})
	}
}

func TestHandoffHandler_GetHandoff(t *testing.T) {
	mockService := NewMockHandoffService()
	handler := NewHandoffHandler(mockService)

	// Create a test handoff first
	createReq := &models.CreateHandoffRequest{
		ProjectName: "test-project",
		FromAgent:   "api-expert",
		ToAgent:     "golang-expert",
		Summary:     "Test handoff",
	}
	handoff, _ := mockService.CreateHandoff(context.Background(), createReq)

	tests := []struct {
		name           string
		handoffID      string
		expectedStatus int
	}{
		{
			name:           "existing handoff",
			handoffID:      handoff.Metadata.HandoffID,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-existent handoff",
			handoffID:      "non-existent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "missing handoff ID",
			handoffID:      "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/handoffs/"+tt.handoffID, nil)
			req.SetPathValue("id", tt.handoffID)
			
			rr := httptest.NewRecorder()
			handler.GetHandoff(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response models.Handoff
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Errorf("failed to decode response: %v", err)
				}

				if response.Metadata.HandoffID != tt.handoffID {
					t.Errorf("expected handoff ID %s, got %s", tt.handoffID, response.Metadata.HandoffID)
				}
			}
		})
	}
}

func TestHandoffHandler_ListHandoffs(t *testing.T) {
	mockService := NewMockHandoffService()
	handler := NewHandoffHandler(mockService)

	// Create test handoffs
	createReq := &models.CreateHandoffRequest{
		ProjectName: "test-project",
		FromAgent:   "api-expert",
		ToAgent:     "golang-expert",
		Summary:     "Test handoff",
	}
	mockService.CreateHandoff(context.Background(), createReq)

	req := httptest.NewRequest("GET", "/api/v1/handoffs", nil)
	rr := httptest.NewRecorder()

	handler.ListHandoffs(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response models.HandoffListResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if len(response.Handoffs) != 1 {
		t.Errorf("expected 1 handoff, got %d", len(response.Handoffs))
	}
}

func TestHandoffHandler_UpdateStatus(t *testing.T) {
	mockService := NewMockHandoffService()
	handler := NewHandoffHandler(mockService)

	// Create a test handoff first
	createReq := &models.CreateHandoffRequest{
		ProjectName: "test-project",
		FromAgent:   "api-expert",
		ToAgent:     "golang-expert",
		Summary:     "Test handoff",
	}
	handoff, _ := mockService.CreateHandoff(context.Background(), createReq)

	tests := []struct {
		name           string
		handoffID      string
		status         models.HandoffStatus
		expectedStatus int
	}{
		{
			name:           "valid status update",
			handoffID:      handoff.Metadata.HandoffID,
			status:         models.StatusProcessing,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "non-existent handoff",
			handoffID:      "non-existent",
			status:         models.StatusProcessing,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateReq := models.UpdateStatusRequest{Status: tt.status}
			payload, _ := json.Marshal(updateReq)
			
			req := httptest.NewRequest("PUT", "/api/v1/handoffs/"+tt.handoffID+"/status", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("id", tt.handoffID)
			
			rr := httptest.NewRecorder()
			handler.UpdateStatus(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}