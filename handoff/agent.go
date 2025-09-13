package handoff

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// HandoffAgent manages Redis-based agent-to-agent communication
type HandoffAgent struct {
	redis         *redis.Client
	logger        zerolog.Logger
	capabilities  map[string]AgentCapabilities
	retryPolicy   RetryPolicy
	metrics       *HandoffMetrics
	metricsMutex  sync.RWMutex
	consumers     map[string]context.CancelFunc
	consumerMutex sync.RWMutex
}

// Config contains HandoffAgent configuration
type Config struct {
	RedisAddr     string       `json:"redis_addr"`
	RedisPassword string       `json:"redis_password,omitempty"`
	RedisDB       int          `json:"redis_db"`
	LogLevel      string       `json:"log_level"`
	RetryPolicy   *RetryPolicy `json:"retry_policy,omitempty"`
}

// NewHandoffAgent creates a new handoff agent instance
func NewHandoffAgent(cfg Config) (*HandoffAgent, error) {
	// Setup Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Setup logger
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	logger := log.With().Str("component", "handoff-agent").Logger().Level(level)

	// Setup retry policy
	retryPolicy := DefaultRetryPolicy()
	if cfg.RetryPolicy != nil {
		retryPolicy = *cfg.RetryPolicy
	}

	agent := &HandoffAgent{
		redis:        rdb,
		logger:       logger,
		capabilities: make(map[string]AgentCapabilities),
		retryPolicy:  retryPolicy,
		metrics: &HandoffMetrics{
			LastUpdated: time.Now(),
		},
		consumers: make(map[string]context.CancelFunc),
	}

	logger.Info().Msg("HandoffAgent initialized successfully")
	return agent, nil
}

// GetRedisClient returns the Redis client for external use
func (h *HandoffAgent) GetRedisClient() *redis.Client {
	return h.redis
}

// RegisterAgent registers an agent's capabilities
func (h *HandoffAgent) RegisterAgent(cap AgentCapabilities) error {
	if cap.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if cap.QueueName == "" {
		cap.QueueName = fmt.Sprintf("handoff:queue:%s", cap.Name)
	}
	if cap.MaxConcurrent == 0 {
		cap.MaxConcurrent = 5
	}

	h.capabilities[cap.Name] = cap
	h.logger.Info().
		Str("agent", cap.Name).
		Str("queue", cap.QueueName).
		Int("max_concurrent", cap.MaxConcurrent).
		Msg("Agent registered")

	return nil
}

// PublishHandoff publishes a handoff to the appropriate queue
func (h *HandoffAgent) PublishHandoff(ctx context.Context, handoff *Handoff) error {
	// Validate handoff
	if err := handoff.Validate(); err != nil {
		return fmt.Errorf("invalid handoff: %w", err)
	}

	// Set handoff metadata
	if handoff.Metadata.HandoffID == "" {
		handoff.Metadata.HandoffID = uuid.New().String()
	}
	handoff.Status = StatusPending
	handoff.CreatedAt = time.Now()
	handoff.UpdatedAt = time.Now()

	// Find target agent capabilities
	targetCap, exists := h.capabilities[handoff.Metadata.ToAgent]
	if !exists {
		return fmt.Errorf("target agent %s not registered", handoff.Metadata.ToAgent)
	}

	// Create queue message
	message := HandoffQueueMessage{
		HandoffID: handoff.Metadata.HandoffID,
		Queue:     targetCap.QueueName,
		Timestamp: time.Now(),
		Priority:  handoff.Metadata.Priority,
		Payload:   *handoff,
	}

	// Serialize message
	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to serialize handoff: %w", err)
	}

	// Store handoff in Redis with expiration (24 hours)
	handoffKey := fmt.Sprintf("handoff:%s", handoff.Metadata.HandoffID)
	if err := h.redis.Set(ctx, handoffKey, messageData, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store handoff: %w", err)
	}

	// Push to priority queue (higher priority = lower score)
	var score float64
	switch string(handoff.Metadata.Priority) {
	case string(PriorityCritical):
		score = 1
	case string(PriorityHigh):
		score = 2
	case string(PriorityNormal):
		score = 3
	case string(PriorityLow):
		score = 4
	default:
		score = 3
	}

	// Add timestamp to ensure FIFO within same priority
	score += float64(time.Now().UnixNano()) / 1e18

	if err := h.redis.ZAdd(ctx, targetCap.QueueName, &redis.Z{
		Score:  score,
		Member: handoff.Metadata.HandoffID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to queue handoff: %w", err)
	}

	// Update metrics
	h.metricsMutex.Lock()
	h.metrics.TotalHandoffs++
	h.metrics.LastUpdated = time.Now()
	h.metricsMutex.Unlock()

	h.logger.Info().
		Str("handoff_id", handoff.Metadata.HandoffID).
		Str("from_agent", handoff.Metadata.FromAgent).
		Str("to_agent", handoff.Metadata.ToAgent).
		Str("queue", targetCap.QueueName).
		Str("priority", string(handoff.Metadata.Priority)).
		Msg("Handoff published")

	return nil
}

// ConsumeHandoffs starts consuming handoffs for a specific agent
func (h *HandoffAgent) ConsumeHandoffs(ctx context.Context, agentName string, handler func(context.Context, *Handoff) error) error {
	cap, exists := h.capabilities[agentName]
	if !exists {
		return fmt.Errorf("agent %s not registered", agentName)
	}

	// Create consumer context
	consumerCtx, cancel := context.WithCancel(ctx)

	h.consumerMutex.Lock()
	h.consumers[agentName] = cancel
	h.consumerMutex.Unlock()

	h.logger.Info().
		Str("agent", agentName).
		Str("queue", cap.QueueName).
		Int("max_concurrent", cap.MaxConcurrent).
		Msg("Starting handoff consumer")

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, cap.MaxConcurrent)

	for {
		select {
		case <-consumerCtx.Done():
			h.logger.Info().Str("agent", agentName).Msg("Consumer stopped")
			return consumerCtx.Err()
		default:
			// Pop from priority queue (lowest score first)
			result, err := h.redis.ZPopMin(consumerCtx, cap.QueueName, 1).Result()
			if err != nil {
				if err == redis.Nil {
					// No messages, wait a bit
					time.Sleep(100 * time.Millisecond)
					continue
				}
				h.logger.Error().Err(err).Msg("Failed to pop from queue")
				time.Sleep(time.Second)
				continue
			}

			if len(result) == 0 {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			handoffID := result[0].Member.(string)

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
			case <-consumerCtx.Done():
				return consumerCtx.Err()
			}

			// Process handoff in goroutine
			go func(id string) {
				defer func() { <-semaphore }()

				if err := h.processHandoff(consumerCtx, id, handler); err != nil {
					h.logger.Error().
						Err(err).
						Str("handoff_id", id).
						Msg("Failed to process handoff")
				}
			}(handoffID)
		}
	}
}

// processHandoff processes a single handoff
func (h *HandoffAgent) processHandoff(ctx context.Context, handoffID string, handler func(context.Context, *Handoff) error) error {
	// Retrieve handoff data
	handoffKey := fmt.Sprintf("handoff:%s", handoffID)
	data, err := h.redis.Get(ctx, handoffKey).Result()
	if err != nil {
		if err == redis.Nil {
			h.logger.Warn().Str("handoff_id", handoffID).Msg("Handoff not found")
			return nil
		}
		return fmt.Errorf("failed to retrieve handoff: %w", err)
	}

	var message HandoffQueueMessage
	if err := json.Unmarshal([]byte(data), &message); err != nil {
		return fmt.Errorf("failed to deserialize handoff: %w", err)
	}

	handoff := &message.Payload

	// Update status to processing
	if err := h.updateHandoffStatus(ctx, handoff, StatusProcessing); err != nil {
		h.logger.Error().Err(err).Str("handoff_id", handoffID).Msg("Failed to update status")
	}

	// Process handoff
	start := time.Now()
	err = handler(ctx, handoff)
	duration := time.Since(start)

	// Update metrics and status
	h.metricsMutex.Lock()
	if err != nil {
		h.metrics.FailedHandoffs++
		handoff.Status = StatusFailed
		handoff.ErrorMsg = err.Error()
	} else {
		h.metrics.CompletedHandoffs++
		handoff.Status = StatusCompleted
		handoff.ErrorMsg = ""
	}

	// Update average processing time
	totalCompleted := h.metrics.CompletedHandoffs + h.metrics.FailedHandoffs
	if totalCompleted > 0 {
		h.metrics.AvgProcessingTime = time.Duration(
			(int64(h.metrics.AvgProcessingTime)*(totalCompleted-1) + int64(duration)) / totalCompleted,
		)
	}
	h.metrics.LastUpdated = time.Now()
	h.metricsMutex.Unlock()

	// Update final status
	if err := h.updateHandoffStatus(ctx, handoff, handoff.Status); err != nil {
		h.logger.Error().Err(err).Str("handoff_id", handoffID).Msg("Failed to update final status")
	}

	if err != nil {
		// Check if we should retry
		if h.shouldRetry(err) && handoff.RetryCount < h.retryPolicy.MaxRetries {
			return h.retryHandoff(ctx, handoff, err)
		}

		h.logger.Error().
			Err(err).
			Str("handoff_id", handoffID).
			Str("from_agent", handoff.Metadata.FromAgent).
			Str("to_agent", handoff.Metadata.ToAgent).
			Int("retry_count", handoff.RetryCount).
			Dur("processing_time", duration).
			Msg("Handoff failed")

		return err
	}

	h.logger.Info().
		Str("handoff_id", handoffID).
		Str("from_agent", handoff.Metadata.FromAgent).
		Str("to_agent", handoff.Metadata.ToAgent).
		Dur("processing_time", duration).
		Msg("Handoff completed successfully")

	return nil
}

// updateHandoffStatus updates the handoff status in Redis
func (h *HandoffAgent) updateHandoffStatus(ctx context.Context, handoff *Handoff, status HandoffStatus) error {
	handoff.Status = status
	handoff.UpdatedAt = time.Now()

	message := HandoffQueueMessage{
		HandoffID: handoff.Metadata.HandoffID,
		Timestamp: time.Now(),
		Priority:  handoff.Metadata.Priority,
		Payload:   *handoff,
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to serialize handoff: %w", err)
	}

	handoffKey := fmt.Sprintf("handoff:%s", handoff.Metadata.HandoffID)
	return h.redis.Set(ctx, handoffKey, messageData, 24*time.Hour).Err()
}

// shouldRetry checks if an error is retriable
func (h *HandoffAgent) shouldRetry(err error) bool {
	errStr := strings.ToLower(err.Error())
	for _, retriableErr := range h.retryPolicy.RetriableErrors {
		if strings.Contains(errStr, strings.ToLower(retriableErr)) {
			return true
		}
	}
	return false
}

// retryHandoff schedules a handoff for retry
func (h *HandoffAgent) retryHandoff(ctx context.Context, handoff *Handoff, originalErr error) error {
	handoff.RetryCount++
	handoff.Status = StatusRetrying

	// Calculate delay with exponential backoff
	delay := time.Duration(float64(h.retryPolicy.InitialDelay) *
		float64(handoff.RetryCount) * h.retryPolicy.BackoffFactor)
	if delay > h.retryPolicy.MaxDelay {
		delay = h.retryPolicy.MaxDelay
	}

	h.logger.Warn().
		Err(originalErr).
		Str("handoff_id", handoff.Metadata.HandoffID).
		Int("retry_count", handoff.RetryCount).
		Dur("retry_delay", delay).
		Msg("Scheduling handoff retry")

	// Update status
	if err := h.updateHandoffStatus(ctx, handoff, StatusRetrying); err != nil {
		return fmt.Errorf("failed to update retry status: %w", err)
	}

	// Schedule retry by re-queuing with delay
	go func() {
		time.Sleep(delay)

		cap := h.capabilities[handoff.Metadata.ToAgent]
		score := float64(time.Now().UnixNano()) / 1e18 // Future timestamp for scheduling

		if err := h.redis.ZAdd(ctx, cap.QueueName, &redis.Z{
			Score:  score,
			Member: handoff.Metadata.HandoffID,
		}).Err(); err != nil {
			h.logger.Error().
				Err(err).
				Str("handoff_id", handoff.Metadata.HandoffID).
				Msg("Failed to schedule retry")
		}
	}()

	return nil
}

// GetMetrics returns current handoff metrics
func (h *HandoffAgent) GetMetrics() HandoffMetrics {
	h.metricsMutex.RLock()
	defer h.metricsMutex.RUnlock()

	// Update queue depths
	metrics := *h.metrics
	metrics.QueueDepth = 0

	for _, cap := range h.capabilities {
		depth, _ := h.redis.ZCard(context.Background(), cap.QueueName).Result()
		metrics.QueueDepth += depth
	}

	// Update active agents
	h.consumerMutex.RLock()
	metrics.ActiveAgents = make([]string, 0, len(h.consumers))
	for agent := range h.consumers {
		metrics.ActiveAgents = append(metrics.ActiveAgents, agent)
	}
	h.consumerMutex.RUnlock()

	return metrics
}

// GetHandoffStatus retrieves the current status of a handoff
func (h *HandoffAgent) GetHandoffStatus(ctx context.Context, handoffID string) (*Handoff, error) {
	handoffKey := fmt.Sprintf("handoff:%s", handoffID)
	data, err := h.redis.Get(ctx, handoffKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("handoff %s not found", handoffID)
		}
		return nil, fmt.Errorf("failed to retrieve handoff: %w", err)
	}

	var message HandoffQueueMessage
	if err := json.Unmarshal([]byte(data), &message); err != nil {
		return nil, fmt.Errorf("failed to deserialize handoff: %w", err)
	}

	return &message.Payload, nil
}

// StopConsumer stops the consumer for a specific agent
func (h *HandoffAgent) StopConsumer(agentName string) {
	h.consumerMutex.Lock()
	defer h.consumerMutex.Unlock()

	if cancel, exists := h.consumers[agentName]; exists {
		cancel()
		delete(h.consumers, agentName)
		h.logger.Info().Str("agent", agentName).Msg("Consumer stopped")
	}
}

// Close closes the handoff agent and all consumers
func (h *HandoffAgent) Close() error {
	h.consumerMutex.Lock()
	for agentName, cancel := range h.consumers {
		cancel()
		h.logger.Info().Str("agent", agentName).Msg("Consumer stopped during shutdown")
	}
	h.consumers = make(map[string]context.CancelFunc)
	h.consumerMutex.Unlock()

	return h.redis.Close()
}
