package handoff

import (
	"crypto/sha256"
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
	StatusRetrying   HandoffStatus = "retrying"
)

// Priority defines the urgency level of a handoff
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// Metadata contains handoff tracking information
type Metadata struct {
	FromAgent   string    `json:"from_agent" yaml:"from_agent"`
	ToAgent     string    `json:"to_agent" yaml:"to_agent"`
	Timestamp   time.Time `json:"timestamp" yaml:"timestamp"`
	TaskContext string    `json:"task_context" yaml:"task_context"`
	Priority    Priority  `json:"priority" yaml:"priority"`
	HandoffID   string    `json:"handoff_id" yaml:"handoff_id"`
}

// Artifacts represents files created, modified, or reviewed
type Artifacts struct {
	Created  []string `json:"created" yaml:"created"`
	Modified []string `json:"modified" yaml:"modified"`
	Reviewed []string `json:"reviewed" yaml:"reviewed"`
}

// Content contains the main handoff information
type Content struct {
	Summary          string                 `json:"summary" yaml:"summary"`
	Requirements     []string               `json:"requirements" yaml:"requirements"`
	Artifacts        Artifacts              `json:"artifacts" yaml:"artifacts"`
	TechnicalDetails map[string]interface{} `json:"technical_details" yaml:"technical_details"`
	NextSteps        []string               `json:"next_steps" yaml:"next_steps"`
}

// Validation contains schema validation information
type Validation struct {
	SchemaVersion string `json:"schema_version" yaml:"schema_version"`
	Checksum      string `json:"checksum" yaml:"checksum"`
}

// Handoff represents a complete agent-to-agent handoff
type Handoff struct {
	Metadata   Metadata   `json:"metadata" yaml:"metadata"`
	Content    Content    `json:"content" yaml:"content"`
	Validation Validation `json:"validation" yaml:"validation"`
	Status     HandoffStatus `json:"status" yaml:"status"`
	CreatedAt  time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" yaml:"updated_at"`
	RetryCount int        `json:"retry_count" yaml:"retry_count"`
	ErrorMsg   string     `json:"error_msg,omitempty" yaml:"error_msg,omitempty"`
}

// GenerateChecksum creates a SHA256 checksum of the handoff content
func (h *Handoff) GenerateChecksum() string {
	data := fmt.Sprintf("%s:%s:%s:%v:%v", 
		h.Metadata.FromAgent, 
		h.Metadata.ToAgent,
		h.Content.Summary,
		h.Content.Requirements,
		h.Content.NextSteps)
	
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// Validate checks if the handoff is valid according to the schema
func (h *Handoff) Validate() error {
	if h.Metadata.FromAgent == "" {
		return fmt.Errorf("from_agent is required")
	}
	if h.Metadata.ToAgent == "" {
		return fmt.Errorf("to_agent is required")
	}
	if h.Content.Summary == "" {
		return fmt.Errorf("summary is required")
	}
	if h.Validation.SchemaVersion == "" {
		h.Validation.SchemaVersion = "1.0"
	}
	if h.Validation.Checksum == "" {
		h.Validation.Checksum = h.GenerateChecksum()
	}
	return nil
}

// HandoffQueueMessage represents a message in the Redis queue
type HandoffQueueMessage struct {
	HandoffID string    `json:"handoff_id"`
	Queue     string    `json:"queue"`
	Timestamp time.Time `json:"timestamp"`
	Priority  Priority  `json:"priority"`
	Payload   Handoff   `json:"payload"`
}

// AgentCapabilities describes what an agent can handle
type AgentCapabilities struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Triggers     []string `json:"triggers"`
	InputTypes   []string `json:"input_types"`
	OutputTypes  []string `json:"output_types"`
	QueueName    string   `json:"queue_name"`
	MaxConcurrent int     `json:"max_concurrent"`
}

// HandoffMetrics contains performance and monitoring data
type HandoffMetrics struct {
	TotalHandoffs    int64         `json:"total_handoffs"`
	CompletedHandoffs int64        `json:"completed_handoffs"`
	FailedHandoffs   int64         `json:"failed_handoffs"`
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
	QueueDepth       int64         `json:"queue_depth"`
	ActiveAgents     []string      `json:"active_agents"`
	LastUpdated      time.Time     `json:"last_updated"`
}

// RetryPolicy defines how failed handoffs should be retried
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetriableErrors []string      `json:"retriable_errors"`
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  time.Second,
		MaxDelay:      time.Minute,
		BackoffFactor: 2.0,
		RetriableErrors: []string{
			"connection error",
			"timeout",
			"temporary failure",
		},
	}
}