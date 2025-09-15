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

// OptimizedHandoffAgent manages Redis-based agent-to-agent communication with optimized connection pooling
type OptimizedHandoffAgent struct {
	redisManager  *RedisManager
	logger        zerolog.Logger
	capabilities  map[string]AgentCapabilities
	retryPolicy   RetryPolicy
	metrics       *HandoffMetrics
	metricsMutex  sync.RWMutex
	consumers     map[string]context.CancelFunc
	consumerMutex sync.RWMutex
}

// OptimizedConfig contains OptimizedHandoffAgent configuration
type OptimizedConfig struct {
	RedisConfig  RedisPoolConfig `json:"redis_config"`
	LogLevel     string          `json:"log_level"`
	RetryPolicy  *RetryPolicy    `json:"retry_policy,omitempty"`
}

// NewOptimizedHandoffAgent creates a new handoff agent instance with optimized Redis pooling
func NewOptimizedHandoffAgent(cfg OptimizedConfig) (*OptimizedHandoffAgent, error) {
	// Initialize Redis manager with optimized pooling
	if err := InitializeRedisManager(cfg.RedisConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize Redis manager: %w", err)
	}
	
	redisManager := GetRedisManager()

	// Test connection health
	if !redisManager.IsHealthy() {
		return nil, fmt.Errorf("Redis connection is not healthy")
	}

	// Setup logger
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	logger := log.With().Str("component", "optimized-handoff-agent").Logger().Level(level)

	// Setup retry policy
	retryPolicy := DefaultRetryPolicy()
	if cfg.RetryPolicy != nil {
		retryPolicy = *cfg.RetryPolicy
	}

	agent := &OptimizedHandoffAgent{
		redisManager: redisManager,
		logger:       logger,
		capabilities: make(map[string]AgentCapabilities),
		retryPolicy:  retryPolicy,
		metrics: &HandoffMetrics{
			LastUpdated: time.Now(),
		},
		consumers: make(map[string]context.CancelFunc),
	}

	logger.Info().
		Str("redis_addr", cfg.RedisConfig.Addr).
		Int("pool_size", cfg.RedisConfig.PoolSize).
		Msg("OptimizedHandoffAgent initialized successfully with connection pooling")
	
	return agent, nil
}

// GetRedisManager returns the Redis manager for external use
func (h *OptimizedHandoffAgent) GetRedisManager() *RedisManager {
	return h.redisManager
}

// GetRedisClient returns the optimized Redis client
func (h *OptimizedHandoffAgent) GetRedisClient() *redis.Client {
	return h.redisManager.GetClient()
}

// RegisterAgent registers an agent's capabilities
func (h *OptimizedHandoffAgent) RegisterAgent(cap AgentCapabilities) error {
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

// PublishHandoff publishes a handoff to the appropriate queue with optimized operations
func (h *OptimizedHandoffAgent) PublishHandoff(ctx context.Context, handoff *Handoff) error {
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

	// Calculate priority score
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

	// Use optimized batch operations
	operations := []func(redis.Pipeliner) error{
		// Store handoff in Redis with expiration (24 hours)
		func(pipe redis.Pipeliner) error {
			handoffKey := fmt.Sprintf("handoff:%s", handoff.Metadata.HandoffID)
			messageData, err := json.Marshal(message)
			if err != nil {
				return fmt.Errorf("failed to serialize handoff: %w", err)
			}
			pipe.Set(ctx, handoffKey, messageData, 24*time.Hour)
			return nil
		},
		// Push to priority queue
		func(pipe redis.Pipeliner) error {
			pipe.ZAdd(ctx, targetCap.QueueName, &redis.Z{
				Score:  score,
				Member: handoff.Metadata.HandoffID,
			})
			return nil
		},
		// Update metrics
		func(pipe redis.Pipeliner) error {
			pipe.Incr(ctx, "handoff:metrics:total")
			pipe.Expire(ctx, "handoff:metrics:total", 24*time.Hour)
			return nil
		},
	}

	// Execute all operations in a single batch
	if err := h.redisManager.ExecuteBatch(ctx, operations); err != nil {
		return fmt.Errorf("failed to publish handoff: %w", err)
	}

	// Update local metrics
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
		Msg("Handoff published with optimized operations")

	return nil
}

// ConsumeHandoffs starts consuming handoffs for a specific agent with optimized queue operations
func (h *OptimizedHandoffAgent) ConsumeHandoffs(ctx context.Context, agentName string, handler func(context.Context, *Handoff) error) error {
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
		Msg("Starting optimized handoff consumer")

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, cap.MaxConcurrent)

	// Use optimized queue operations
	queueOps := h.redisManager.GetQueueOps()

	for {
		select {
		case <-consumerCtx.Done():
			h.logger.Info().Str("agent", agentName).Msg("Consumer stopped")
			return consumerCtx.Err()
		default:
			// Use optimized ZPopMin operation
			results, err := queueOps.ZPopMinBatch(consumerCtx, []string{cap.QueueName}, 1)
			if err != nil {
				if err != redis.Nil {
					h.logger.Error().Err(err).Msg("Failed to pop from queue")
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			queueResults, exists := results[cap.QueueName]
			if !exists || len(queueResults) == 0 {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			handoffID := queueResults[0].Member.(string)

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
			case <-consumerCtx.Done():
				return consumerCtx.Err()
			}

			// Process handoff in goroutine
			go func(id string) {
				defer func() { <-semaphore }()

				if err := h.processHandoffOptimized(consumerCtx, id, handler); err != nil {
					h.logger.Error().
						Err(err).
						Str("handoff_id", id).
						Msg("Failed to process handoff")
				}
			}(handoffID)
		}
	}
}

// processHandoffOptimized processes a single handoff with optimized Redis operations
func (h *OptimizedHandoffAgent) processHandoffOptimized(ctx context.Context, handoffID string, handler func(context.Context, *Handoff) error) error {
	// Retrieve handoff data using optimized operations
	var message HandoffQueueMessage
	handoffKey := fmt.Sprintf("handoff:%s", handoffID)
	
	if err := h.redisManager.GetWithDeserialization(ctx, handoffKey, &message); err != nil {
		if err == redis.Nil {
			h.logger.Warn().Str("handoff_id", handoffID).Msg("Handoff not found")
			return nil
		}
		return fmt.Errorf("failed to retrieve handoff: %w", err)
	}

	handoff := &message.Payload

	// Update status to processing
	if err := h.updateHandoffStatusOptimized(ctx, handoff, StatusProcessing); err != nil {
		h.logger.Error().Err(err).Str("handoff_id", handoffID).Msg("Failed to update status")
	}

	// Process handoff
	start := time.Now()
	err := handler(ctx, handoff)
	duration := time.Since(start)

	// Update metrics and status using optimized operations
	success := err == nil
	
	// Prepare batch operations for metrics update
	operations := []func(redis.Pipeliner) error{
		func(pipe redis.Pipeliner) error {
			if success {
				pipe.Incr(ctx, "handoff:metrics:completed")
			} else {
				pipe.Incr(ctx, "handoff:metrics:failed")
			}
			pipe.Expire(ctx, "handoff:metrics:completed", 24*time.Hour)
			pipe.Expire(ctx, "handoff:metrics:failed", 24*time.Hour)
			return nil
		},
		func(pipe redis.Pipeliner) error {
			// Record processing time
			pipe.LPush(ctx, "handoff:processing_times", duration.String())
			pipe.LTrim(ctx, "handoff:processing_times", 0, 99) // Keep last 100
			pipe.Expire(ctx, "handoff:processing_times", 24*time.Hour)
			return nil
		},
	}

	// Execute metrics update batch
	if batchErr := h.redisManager.ExecuteBatch(ctx, operations); batchErr != nil {
		h.logger.Error().Err(batchErr).Msg("Failed to update metrics")
	}

	// Update local metrics
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
	if err := h.updateHandoffStatusOptimized(ctx, handoff, handoff.Status); err != nil {
		h.logger.Error().Err(err).Str("handoff_id", handoffID).Msg("Failed to update final status")
	}

	if err != nil {
		// Check if we should retry
		if h.shouldRetry(err) && handoff.RetryCount < h.retryPolicy.MaxRetries {
			return h.retryHandoffOptimized(ctx, handoff, err)
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

// updateHandoffStatusOptimized updates the handoff status using optimized Redis operations
func (h *OptimizedHandoffAgent) updateHandoffStatusOptimized(ctx context.Context, handoff *Handoff, status HandoffStatus) error {
	handoff.Status = status
	handoff.UpdatedAt = time.Now()

	message := HandoffQueueMessage{
		HandoffID: handoff.Metadata.HandoffID,
		Timestamp: time.Now(),
		Priority:  handoff.Metadata.Priority,
		Payload:   *handoff,
	}

	handoffKey := fmt.Sprintf("handoff:%s", handoff.Metadata.HandoffID)
	return h.redisManager.SetWithOptimizedExpiry(ctx, handoffKey, message, 24*time.Hour)
}

// shouldRetry checks if an error is retriable
func (h *OptimizedHandoffAgent) shouldRetry(err error) bool {
	errStr := strings.ToLower(err.Error())
	for _, retriableErr := range h.retryPolicy.RetriableErrors {
		if strings.Contains(errStr, strings.ToLower(retriableErr)) {
			return true
		}
	}
	return false
}

// retryHandoffOptimized schedules a handoff for retry using optimized operations
func (h *OptimizedHandoffAgent) retryHandoffOptimized(ctx context.Context, handoff *Handoff, originalErr error) error {
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
	if err := h.updateHandoffStatusOptimized(ctx, handoff, StatusRetrying); err != nil {
		return fmt.Errorf("failed to update retry status: %w", err)
	}

	// Schedule retry by re-queuing with delay
	go func() {
		time.Sleep(delay)

		cap := h.capabilities[handoff.Metadata.ToAgent]
		score := float64(time.Now().UnixNano()) / 1e18 // Future timestamp for scheduling

		queueOps := h.redisManager.GetQueueOps()
		members := []*redis.Z{{
			Score:  score,
			Member: handoff.Metadata.HandoffID,
		}}

		if err := queueOps.ZAddBatch(ctx, cap.QueueName, members); err != nil {
			h.logger.Error().
				Err(err).
				Str("handoff_id", handoff.Metadata.HandoffID).
				Msg("Failed to schedule retry")
		}
	}()

	return nil
}

// GetOptimizedMetrics returns current handoff metrics with Redis pool metrics
func (h *OptimizedHandoffAgent) GetOptimizedMetrics() (HandoffMetrics, RedisPoolMetrics) {
	h.metricsMutex.RLock()
	defer h.metricsMutex.RUnlock()

	// Update queue depths using optimized operations
	metrics := *h.metrics
	metrics.QueueDepth = 0

	ctx := context.Background()
	client := h.redisManager.GetClient()
	
	for _, cap := range h.capabilities {
		depth, _ := client.ZCard(ctx, cap.QueueName).Result()
		metrics.QueueDepth += depth
	}

	// Update active agents
	h.consumerMutex.RLock()
	metrics.ActiveAgents = make([]string, 0, len(h.consumers))
	for agent := range h.consumers {
		metrics.ActiveAgents = append(metrics.ActiveAgents, agent)
	}
	h.consumerMutex.RUnlock()

	// Get Redis pool metrics
	redisMetrics := h.redisManager.GetDetailedMetrics()

	return metrics, redisMetrics
}

// GetHandoffStatus retrieves the current status of a handoff using optimized operations
func (h *OptimizedHandoffAgent) GetHandoffStatus(ctx context.Context, handoffID string) (*Handoff, error) {
	var message HandoffQueueMessage
	handoffKey := fmt.Sprintf("handoff:%s", handoffID)
	
	if err := h.redisManager.GetWithDeserialization(ctx, handoffKey, &message); err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("handoff %s not found", handoffID)
		}
		return nil, fmt.Errorf("failed to retrieve handoff: %w", err)
	}

	return &message.Payload, nil
}

// StopConsumer stops the consumer for a specific agent
func (h *OptimizedHandoffAgent) StopConsumer(agentName string) {
	h.consumerMutex.Lock()
	defer h.consumerMutex.Unlock()

	if cancel, exists := h.consumers[agentName]; exists {
		cancel()
		delete(h.consumers, agentName)
		h.logger.Info().Str("agent", agentName).Msg("Consumer stopped")
	}
}

// Close closes the handoff agent and all consumers
func (h *OptimizedHandoffAgent) Close() error {
	h.consumerMutex.Lock()
	for agentName, cancel := range h.consumers {
		cancel()
		h.logger.Info().Str("agent", agentName).Msg("Consumer stopped during shutdown")
	}
	h.consumers = make(map[string]context.CancelFunc)
	h.consumerMutex.Unlock()

	// The Redis manager is shared, so we don't close it here
	// It will be closed when the application shuts down
	return nil
}

// GetHealthStatus returns the health status of the Redis connection
func (h *OptimizedHandoffAgent) GetHealthStatus() HealthStatus {
	return h.redisManager.GetHealth()
}

// IsHealthy returns true if the Redis connection is healthy
func (h *OptimizedHandoffAgent) IsHealthy() bool {
	return h.redisManager.IsHealthy()
}

// PerformMaintenance performs periodic maintenance tasks for memory optimization
func (h *OptimizedHandoffAgent) PerformMaintenance(ctx context.Context) error {
	h.logger.Info().Msg("Performing Redis maintenance for memory optimization")
	
	// Clean up expired handoffs
	expiredPatterns := []string{
		"handoff:*",
		"handoff:metrics:*",
		"handoff:processing_times",
	}
	
	if err := h.redisManager.CleanupExpiredKeys(ctx, expiredPatterns); err != nil {
		h.logger.Error().Err(err).Msg("Failed to cleanup expired keys")
		return err
	}
	
	// Set memory optimizations
	if err := h.redisManager.SetMemoryOptimizations(ctx); err != nil {
		h.logger.Error().Err(err).Msg("Failed to set memory optimizations")
		return err
	}
	
	h.logger.Info().Msg("Redis maintenance completed successfully")
	return nil
}