package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// HandoffStatus represents the current status of a handoff
type HandoffStatus string

const (
	StatusPending    HandoffStatus = "pending"
	StatusProcessing HandoffStatus = "processing"
	StatusCompleted  HandoffStatus = "completed"
	StatusFailed     HandoffStatus = "failed"
	StatusCancelled  HandoffStatus = "cancelled"
)

// Priority represents the priority level of a handoff
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// GetPriorityScore returns numeric score for priority sorting
func (p Priority) GetScore() float64 {
	switch p {
	case PriorityUrgent:
		return 1.0
	case PriorityHigh:
		return 2.0
	case PriorityNormal:
		return 3.0
	case PriorityLow:
		return 4.0
	default:
		return 3.0 // Default to normal
	}
}

// Handoff represents a task handoff between agents
type Handoff struct {
	Metadata HandoffMetadata `json:"metadata"`
	Content  HandoffContent  `json:"content"`
	Status   HandoffStatus   `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// HandoffMetadata contains metadata about the handoff
type HandoffMetadata struct {
	ProjectName string    `json:"project_name"`
	FromAgent   string    `json:"from_agent"`
	ToAgent     string    `json:"to_agent"`
	Timestamp   time.Time `json:"timestamp"`
	TaskContext string    `json:"task_context"`
	Priority    Priority  `json:"priority"`
	HandoffID   string    `json:"handoff_id"`
}

// HandoffContent contains the actual content and requirements
type HandoffContent struct {
	Summary          string                 `json:"summary"`
	Requirements     []string               `json:"requirements"`
	Artifacts        map[string][]string    `json:"artifacts"`
	TechnicalDetails map[string]interface{} `json:"technical_details"`
	NextSteps        []string               `json:"next_steps"`
}

// CreateHandoffRequest represents a request to create a new handoff
type CreateHandoffRequest struct {
	ProjectName      string                 `json:"project_name"`
	FromAgent        string                 `json:"from_agent"`
	ToAgent          string                 `json:"to_agent"`
	TaskContext      string                 `json:"task_context"`
	Priority         Priority               `json:"priority"`
	Summary          string                 `json:"summary"`
	Requirements     []string               `json:"requirements"`
	Artifacts        map[string][]string    `json:"artifacts"`
	TechnicalDetails map[string]interface{} `json:"technical_details"`
	NextSteps        []string               `json:"next_steps"`
}

// UpdateStatusRequest represents a request to update handoff status
type UpdateStatusRequest struct {
	Status HandoffStatus `json:"status"`
}

// HandoffListResponse represents a paginated list of handoffs
type HandoffListResponse struct {
	Handoffs   []Handoff `json:"handoffs"`
	TotalCount int       `json:"total_count"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	HasMore    bool      `json:"has_more"`
}

// QueueInfo represents information about a queue
type QueueInfo struct {
	QueueName   string `json:"queue_name"`
	ProjectName string `json:"project_name"`
	AgentName   string `json:"agent_name"`
	Depth       int64  `json:"depth"`
	OldestTask  *time.Time `json:"oldest_task,omitempty"`
}

// Validate validates the handoff data
func (h *Handoff) Validate() error {
	if h.Metadata.ProjectName == "" {
		return fmt.Errorf("project name is required")
	}
	if h.Metadata.FromAgent == "" {
		return fmt.Errorf("from agent is required")
	}
	if h.Metadata.ToAgent == "" {
		return fmt.Errorf("to agent is required")
	}
	if h.Metadata.HandoffID == "" {
		return fmt.Errorf("handoff ID is required")
	}
	if h.Content.Summary == "" {
		return fmt.Errorf("summary is required")
	}
	return nil
}

// Validate validates the create handoff request
func (r *CreateHandoffRequest) Validate() error {
	if r.ProjectName == "" {
		return fmt.Errorf("project name is required")
	}
	if r.FromAgent == "" {
		return fmt.Errorf("from agent is required")
	}
	if r.ToAgent == "" {
		return fmt.Errorf("to agent is required")
	}
	if r.Summary == "" {
		return fmt.Errorf("summary is required")
	}
	
	// Validate priority
	switch r.Priority {
	case PriorityLow, PriorityNormal, PriorityHigh, PriorityUrgent:
		// Valid priority
	case "":
		r.Priority = PriorityNormal // Default to normal
	default:
		return fmt.Errorf("invalid priority: %s", r.Priority)
	}
	
	return nil
}

// ToJSON converts handoff to JSON string
func (h *Handoff) ToJSON() ([]byte, error) {
	return json.MarshalIndent(h, "", "  ")
}

// FromJSON creates handoff from JSON data
func (h *Handoff) FromJSON(data []byte) error {
	return json.Unmarshal(data, h)
}

// GetQueueName returns the Redis queue name for this handoff
func (h *Handoff) GetQueueName() string {
	return fmt.Sprintf("handoff:project:%s:queue:%s", h.Metadata.ProjectName, h.Metadata.ToAgent)
}

// GetRedisKey returns the Redis key for storing this handoff
func (h *Handoff) GetRedisKey() string {
	return fmt.Sprintf("handoff:%s", h.Metadata.HandoffID)
}

// GetPriorityScore returns the priority score for queue ordering
func (h *Handoff) GetPriorityScore() float64 {
	baseScore := h.Metadata.Priority.GetScore()
	// Add timestamp component for FIFO within same priority
	timestampComponent := float64(h.CreatedAt.UnixNano()) / 1e18
	return baseScore + timestampComponent
}