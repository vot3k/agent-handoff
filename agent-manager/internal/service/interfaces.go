package service

import (
	"context"

	"github.com/vot3k/agent-handoff/agent-manager/internal/models"
)

// HandoffServiceInterface defines the interface for handoff operations
type HandoffServiceInterface interface {
	CreateHandoff(ctx context.Context, req *models.CreateHandoffRequest) (*models.Handoff, error)
	GetHandoff(ctx context.Context, handoffID string) (*models.Handoff, error)
	ListHandoffs(ctx context.Context, projectName string, page, pageSize int) (*models.HandoffListResponse, error)
	UpdateStatus(ctx context.Context, handoffID string, status models.HandoffStatus) error
	GetQueues(ctx context.Context, projectName string) ([]models.QueueInfo, error)
	GetQueueDepth(ctx context.Context, queueName string) (int64, error)
	ProcessNextHandoff(ctx context.Context, queueName string) (*models.Handoff, error)
	CompleteHandoff(ctx context.Context, handoffID string) error
	FailHandoff(ctx context.Context, handoffID string) error
	CancelHandoff(ctx context.Context, handoffID string) error
}
