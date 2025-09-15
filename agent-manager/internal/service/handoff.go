package service

import (
	"context"
	"fmt"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/models"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// HandoffService provides business logic for handoff operations
type HandoffService struct {
	repo   repository.HandoffRepositoryInterface
	config *config.Config
}

// NewHandoffService creates a new handoff service
func NewHandoffService(repo repository.HandoffRepositoryInterface, cfg *config.Config) *HandoffService {
	return &HandoffService{
		repo:   repo,
		config: cfg,
	}
}

// CreateHandoff creates a new handoff from a request
func (s *HandoffService) CreateHandoff(ctx context.Context, req *models.CreateHandoffRequest) (*models.Handoff, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Generate handoff ID
	handoffID := s.generateHandoffID()

	// Create handoff model
	now := time.Now()
	handoff := &models.Handoff{
		Metadata: models.HandoffMetadata{
			ProjectName: req.ProjectName,
			FromAgent:   req.FromAgent,
			ToAgent:     req.ToAgent,
			Timestamp:   now,
			TaskContext: req.TaskContext,
			Priority:    req.Priority,
			HandoffID:   handoffID,
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

	// Validate handoff
	if err := handoff.Validate(); err != nil {
		return nil, fmt.Errorf("handoff validation failed: %w", err)
	}

	// Store handoff
	if err := s.repo.Create(ctx, handoff); err != nil {
		return nil, fmt.Errorf("failed to create handoff: %w", err)
	}

	return handoff, nil
}

// GetHandoff retrieves a handoff by ID
func (s *HandoffService) GetHandoff(ctx context.Context, handoffID string) (*models.Handoff, error) {
	if handoffID == "" {
		return nil, fmt.Errorf("handoff ID is required")
	}

	handoff, err := s.repo.GetByID(ctx, handoffID)
	if err != nil {
		return nil, fmt.Errorf("failed to get handoff: %w", err)
	}

	return handoff, nil
}

// ListHandoffs retrieves handoffs with pagination and optional filtering
func (s *HandoffService) ListHandoffs(ctx context.Context, projectName string, page, pageSize int) (*models.HandoffListResponse, error) {
	// Validate pagination parameters
	if page < 1 {
		page = 1
	}

	// Use configurable page size from config, with defaults if not set
	defaultPageSize := 20
	maxPageSize := 100

	if s.config != nil {
		if s.config.Pagination.DefaultPageSize > 0 {
			defaultPageSize = s.config.Pagination.DefaultPageSize
		}
		if s.config.Pagination.MaxPageSize > 0 {
			maxPageSize = s.config.Pagination.MaxPageSize
		}
	}

	if pageSize < 1 || pageSize > maxPageSize {
		pageSize = defaultPageSize
	}

	response, err := s.repo.List(ctx, projectName, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list handoffs: %w", err)
	}

	return response, nil
}

// UpdateStatus updates the status of a handoff
func (s *HandoffService) UpdateStatus(ctx context.Context, handoffID string, status models.HandoffStatus) error {
	if handoffID == "" {
		return fmt.Errorf("handoff ID is required")
	}

	// Validate status transition
	if err := s.validateStatusTransition(ctx, handoffID, status); err != nil {
		return fmt.Errorf("invalid status transition: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, handoffID, status); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// GetQueues retrieves information about all queues
func (s *HandoffService) GetQueues(ctx context.Context, projectName string) ([]models.QueueInfo, error) {
	queues, err := s.repo.GetQueues(ctx, projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to get queues: %w", err)
	}

	return queues, nil
}

// GetQueueDepth retrieves the depth of a specific queue
func (s *HandoffService) GetQueueDepth(ctx context.Context, queueName string) (int64, error) {
	if queueName == "" {
		return 0, fmt.Errorf("queue name is required")
	}

	depth, err := s.repo.GetQueueDepth(ctx, queueName)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue depth: %w", err)
	}

	return depth, nil
}

// ProcessNextHandoff processes the next handoff from a queue
func (s *HandoffService) ProcessNextHandoff(ctx context.Context, queueName string) (*models.Handoff, error) {
	// Pop handoff ID from queue
	handoffID, err := s.repo.PopFromQueue(ctx, queueName)
	if err != nil {
		return nil, fmt.Errorf("failed to pop from queue: %w", err)
	}

	// Get handoff details
	handoff, err := s.repo.GetByID(ctx, handoffID)
	if err != nil {
		return nil, fmt.Errorf("failed to get handoff details: %w", err)
	}

	// Update status to processing
	if err := s.repo.UpdateStatus(ctx, handoffID, models.StatusProcessing); err != nil {
		return nil, fmt.Errorf("failed to update status to processing: %w", err)
	}

	return handoff, nil
}

// CompleteHandoff marks a handoff as completed
func (s *HandoffService) CompleteHandoff(ctx context.Context, handoffID string) error {
	return s.UpdateStatus(ctx, handoffID, models.StatusCompleted)
}

// FailHandoff marks a handoff as failed
func (s *HandoffService) FailHandoff(ctx context.Context, handoffID string) error {
	return s.UpdateStatus(ctx, handoffID, models.StatusFailed)
}

// CancelHandoff marks a handoff as cancelled and removes it from queue
func (s *HandoffService) CancelHandoff(ctx context.Context, handoffID string) error {
	// Get handoff to find its queue
	handoff, err := s.repo.GetByID(ctx, handoffID)
	if err != nil {
		return fmt.Errorf("failed to get handoff: %w", err)
	}

	// Remove from queue if still pending
	if handoff.Status == models.StatusPending {
		queueName := handoff.GetQueueName()
		if err := s.repo.RemoveFromQueue(ctx, queueName, handoffID); err != nil {
			// Don't fail if already removed from queue
		}
	}

	// Update status
	return s.UpdateStatus(ctx, handoffID, models.StatusCancelled)
}

// Private helper methods

// generateHandoffID generates a unique handoff ID using UUID
func (s *HandoffService) generateHandoffID() string {
	return uuid.New().String()
}

// validateStatusTransition validates that a status transition is allowed
func (s *HandoffService) validateStatusTransition(ctx context.Context, handoffID string, newStatus models.HandoffStatus) error {
	handoff, err := s.repo.GetByID(ctx, handoffID)
	if err != nil {
		return err
	}

	currentStatus := handoff.Status

	// Define allowed transitions
	allowedTransitions := map[models.HandoffStatus][]models.HandoffStatus{
		models.StatusPending: {
			models.StatusProcessing,
			models.StatusCancelled,
		},
		models.StatusProcessing: {
			models.StatusCompleted,
			models.StatusFailed,
			models.StatusCancelled,
		},
		models.StatusCompleted: {
			// Terminal state - no transitions allowed
		},
		models.StatusFailed: {
			models.StatusPending, // Allow retry
		},
		models.StatusCancelled: {
			// Terminal state - no transitions allowed
		},
	}

	allowedNext, exists := allowedTransitions[currentStatus]
	if !exists {
		return fmt.Errorf("unknown current status: %s", currentStatus)
	}

	for _, allowed := range allowedNext {
		if allowed == newStatus {
			return nil // Valid transition
		}
	}

	return fmt.Errorf("invalid transition from %s to %s", currentStatus, newStatus)
}
