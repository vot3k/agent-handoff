package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vot3k/agent-handoff/agent-manager/internal/config"
	"github.com/vot3k/agent-handoff/agent-manager/internal/models"

	"github.com/go-redis/redis/v8"
)

// RedisClient wraps redis client with our interface
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg config.RedisConfig) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{client: rdb}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Ping tests the Redis connection
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// HandoffRepository handles handoff data persistence in Redis
type HandoffRepository struct {
	redis *RedisClient
}

// Ensure HandoffRepository implements the interface at compile time
var _ HandoffRepositoryInterface = (*HandoffRepository)(nil)

// NewHandoffRepository creates a new handoff repository
func NewHandoffRepository(redisClient *RedisClient) *HandoffRepository {
	return &HandoffRepository{
		redis: redisClient,
	}
}

// Create stores a new handoff in Redis and adds it to the appropriate queue
func (r *HandoffRepository) Create(ctx context.Context, handoff *models.Handoff) error {
	// Serialize handoff to JSON
	data, err := handoff.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize handoff: %w", err)
	}

	// Store handoff data with 24 hour expiration
	handoffKey := GetHandoffKey(handoff.Metadata.HandoffID)
	queueName := handoff.GetQueueName()
	score := handoff.GetPriorityScore()

	// Use Redis transaction to ensure atomicity
	_, err = r.redis.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		// Store handoff data
		pipe.Set(ctx, handoffKey, data, 24*time.Hour)

		// Add to priority queue
		pipe.ZAdd(ctx, queueName, &redis.Z{
			Score:  score,
			Member: handoff.Metadata.HandoffID,
		})

		// Add to project set for efficient listing
		projectSetKey := GetHandoffProjectSetKey(handoff.Metadata.ProjectName)
		pipe.SAdd(ctx, projectSetKey, handoff.Metadata.HandoffID)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to store handoff with transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a handoff by its ID
func (r *HandoffRepository) GetByID(ctx context.Context, handoffID string) (*models.Handoff, error) {
	key := GetHandoffKey(handoffID)

	data, err := r.redis.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("handoff not found: %s", handoffID)
		}
		return nil, fmt.Errorf("failed to retrieve handoff: %w", err)
	}

	var handoff models.Handoff
	if err := handoff.FromJSON([]byte(data)); err != nil {
		return nil, fmt.Errorf("failed to deserialize handoff: %w", err)
	}

	return &handoff, nil
}

// UpdateStatus updates the status of a handoff
func (r *HandoffRepository) UpdateStatus(ctx context.Context, handoffID string, status models.HandoffStatus) error {
	// Get existing handoff
	handoff, err := r.GetByID(ctx, handoffID)
	if err != nil {
		return err
	}

	// Update status and timestamp
	handoff.Status = status
	handoff.UpdatedAt = time.Now()

	// Store updated handoff
	data, err := handoff.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize updated handoff: %w", err)
	}

	key := handoff.GetRedisKey()
	if err := r.redis.client.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to update handoff: %w", err)
	}

	return nil
}

// List retrieves handoffs with pagination
func (r *HandoffRepository) List(ctx context.Context, projectName string, page, pageSize int) (*models.HandoffListResponse, error) {
	// Use Redis sets for efficient listing instead of KEYS command
	var handoffIDs []string
	var err error

	if projectName != "" {
		// Get all handoff IDs for this project from the set
		projectSetKey := GetHandoffProjectSetKey(projectName)
		handoffIDs, err = r.redis.client.SMembers(ctx, projectSetKey).Result()
	} else {
		// Get all handoff IDs from the global set
		globalSetKey := GetHandoffListKey("")
		handoffIDs, err = r.redis.client.SMembers(ctx, globalSetKey).Result()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve handoff IDs: %w", err)
	}

	// Get the handoffs by their IDs
	var handoffs []models.Handoff
	for _, handoffID := range handoffIDs {
		handoff, err := r.GetByID(ctx, handoffID)
		if err != nil {
			continue // Skip failed retrievals
		}

		// Filter by project if specified
		if projectName != "" && handoff.Metadata.ProjectName != projectName {
			continue
		}

		handoffs = append(handoffs, *handoff)
	}

	// Simple pagination
	totalCount := len(handoffs)
	start := (page - 1) * pageSize
	end := start + pageSize

	if start > totalCount {
		start = totalCount
	}
	if end > totalCount {
		end = totalCount
	}

	pagedHandoffs := handoffs[start:end]
	hasMore := end < totalCount

	return &models.HandoffListResponse{
		Handoffs:   pagedHandoffs,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		HasMore:    hasMore,
	}, nil
}

// GetQueues returns information about all active queues
func (r *HandoffRepository) GetQueues(ctx context.Context, projectName string) ([]models.QueueInfo, error) {
	pattern := "handoff:project:*:queue:*"
	if projectName != "" {
		pattern = fmt.Sprintf("handoff:project:%s:queue:*", projectName)
	}

	queueNames, err := r.redis.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to scan queue keys: %w", err)
	}

	var queues []models.QueueInfo
	for _, queueName := range queueNames {
		depth, err := r.redis.client.ZCard(ctx, queueName).Result()
		if err != nil {
			continue // Skip failed queries
		}

		// Parse project and agent name from queue name
		projectName, agentName := GetProjectAndAgentFromQueueKey(queueName)

		queueInfo := models.QueueInfo{
			QueueName:   queueName,
			ProjectName: projectName,
			AgentName:   agentName,
			Depth:       depth,
		}

		// Get oldest task timestamp if queue is not empty
		if depth > 0 {
			if oldestTask, err := r.getOldestTaskTime(ctx, queueName); err == nil {
				queueInfo.OldestTask = &oldestTask
			}
		}

		queues = append(queues, queueInfo)
	}

	return queues, nil
}

// GetQueueDepth returns the depth of a specific queue
func (r *HandoffRepository) GetQueueDepth(ctx context.Context, queueName string) (int64, error) {
	depth, err := r.redis.client.ZCard(ctx, queueName).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue depth: %w", err)
	}
	return depth, nil
}

// RemoveFromQueue removes a handoff from its queue (used when processing starts)
func (r *HandoffRepository) RemoveFromQueue(ctx context.Context, queueName, handoffID string) error {
	removed, err := r.redis.client.ZRem(ctx, queueName, handoffID).Result()
	if err != nil {
		return fmt.Errorf("failed to remove from queue: %w", err)
	}
	if removed == 0 {
		return fmt.Errorf("handoff not found in queue: %s", handoffID)
	}
	return nil
}

// PopFromQueue removes and returns the highest priority handoff from a queue
func (r *HandoffRepository) PopFromQueue(ctx context.Context, queueName string) (string, error) {
	result, err := r.redis.client.ZPopMin(ctx, queueName, 1).Result()
	if err != nil {
		return "", fmt.Errorf("failed to pop from queue: %w", err)
	}

	if len(result) == 0 {
		return "", fmt.Errorf("queue is empty: %s", queueName)
	}

	handoffID := result[0].Member.(string)
	return handoffID, nil
}

// Helper functions

// parseQueueName extracts project and agent name from queue name
// Expected format: "handoff:project:{projectName}:queue:{agentName}"
func parseQueueName(queueName string) (string, string) {
	parts := strings.Split(queueName, ":")
	if len(parts) == 5 && parts[0] == "handoff" && parts[1] == "project" && parts[3] == "queue" {
		return parts[2], parts[4]
	}
	return "", ""
}

// getOldestTaskTime retrieves the creation time of the oldest task in a queue
func (r *HandoffRepository) getOldestTaskTime(ctx context.Context, queueName string) (time.Time, error) {
	// Get the task with the lowest score (highest priority, oldest timestamp)
	result, err := r.redis.client.ZRangeWithScores(ctx, queueName, 0, 0).Result()
	if err != nil || len(result) == 0 {
		return time.Time{}, fmt.Errorf("no tasks in queue")
	}

	// Get the handoff ID
	handoffID := result[0].Member.(string)

	// Retrieve the handoff to get creation time
	handoff, err := r.GetByID(ctx, handoffID)
	if err != nil {
		return time.Time{}, err
	}

	return handoff.CreatedAt, nil
}
