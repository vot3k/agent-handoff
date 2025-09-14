package service

import (
	"context"
	"testing"

	"agent-manager/internal/config"
	"agent-manager/internal/models"
)

func TestHandoffService_GenerateHandoffID(t *testing.T) {
	// Create a mock config
	cfg := &config.Config{
		Pagination: config.PaginationConfig{
			DefaultPageSize: 20,
			MaxPageSize:     100,
		},
	}

	// Create a mock repository
	mockRepo := &MockHandoffRepository{}

	// Create service
	service := NewHandoffService(mockRepo, cfg)

	// Generate ID
	id := service.generateHandoffID()

	// Verify it's not empty
	if id == "" {
		t.Error("Generated ID should not be empty")
	}

	// Verify it's a valid UUID format (basic check)
	if len(id) < 32 {
		t.Error("Generated ID should be a valid UUID")
	}
}

// Mock repository for testing
type MockHandoffRepository struct{}

func (m *MockHandoffRepository) Create(ctx context.Context, handoff *models.Handoff) error {
	return nil
}

func (m *MockHandoffRepository) GetByID(ctx context.Context, handoffID string) (*models.Handoff, error) {
	return nil, nil
}

func (m *MockHandoffRepository) UpdateStatus(ctx context.Context, handoffID string, status models.HandoffStatus) error {
	return nil
}

func (m *MockHandoffRepository) List(ctx context.Context, projectName string, page, pageSize int) (*models.HandoffListResponse, error) {
	return nil, nil
}

func (m *MockHandoffRepository) GetQueues(ctx context.Context, projectName string) ([]models.QueueInfo, error) {
	return nil, nil
}

func (m *MockHandoffRepository) GetQueueDepth(ctx context.Context, queueName string) (int64, error) {
	return 0, nil
}

func (m *MockHandoffRepository) RemoveFromQueue(ctx context.Context, queueName, handoffID string) error {
	return nil
}

func (m *MockHandoffRepository) PopFromQueue(ctx context.Context, queueName string) (string, error) {
	return "", nil
}
