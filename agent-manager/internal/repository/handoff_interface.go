package repository

import (
	"agent-manager/internal/models"
	"context"
)

// HandoffRepositoryInterface defines the interface for handoff repository operations
type HandoffRepositoryInterface interface {
	// Create stores a new handoff
	Create(ctx context.Context, handoff *models.Handoff) error

	// GetByID retrieves a handoff by its ID
	GetByID(ctx context.Context, handoffID string) (*models.Handoff, error)

	// UpdateStatus updates the status of a handoff
	UpdateStatus(ctx context.Context, handoffID string, status models.HandoffStatus) error

	// List retrieves handoffs with pagination
	List(ctx context.Context, projectName string, page, pageSize int) (*models.HandoffListResponse, error)

	// GetQueues returns information about all active queues
	GetQueues(ctx context.Context, projectName string) ([]models.QueueInfo, error)

	// GetQueueDepth returns the depth of a specific queue
	GetQueueDepth(ctx context.Context, queueName string) (int64, error)

	// RemoveFromQueue removes a handoff from its queue
	RemoveFromQueue(ctx context.Context, queueName, handoffID string) error

	// PopFromQueue removes and returns the highest priority handoff from a queue
	PopFromQueue(ctx context.Context, queueName string) (string, error)
}
