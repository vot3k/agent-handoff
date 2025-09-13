package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisPoolConfig contains configuration for Redis connection pool (copied from handoff package)
type RedisPoolConfig struct {
	// Connection settings
	Addr     string `json:"addr"`
	Password string `json:"password,omitempty"`
	DB       int    `json:"db"`

	// Pool settings
	PoolSize        int           `json:"pool_size"`         // Maximum number of socket connections
	MinIdleConns    int           `json:"min_idle_conns"`    // Minimum number of idle connections
	MaxConnAge      time.Duration `json:"max_conn_age"`      // Connection age at which client retires connection
	PoolTimeout     time.Duration `json:"pool_timeout"`      // Amount of time client waits for connection
	IdleTimeout     time.Duration `json:"idle_timeout"`      // Amount of time after which client closes idle connections
	IdleCheckFreq   time.Duration `json:"idle_check_freq"`   // Frequency of idle checks
	
	// Operation timeouts
	DialTimeout  time.Duration `json:"dial_timeout"`  // Timeout for socket connection
	ReadTimeout  time.Duration `json:"read_timeout"`  // Timeout for socket reads
	WriteTimeout time.Duration `json:"write_timeout"` // Timeout for socket writes

	// Health check settings
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	MaxRetries          int           `json:"max_retries"`
	MinRetryBackoff     time.Duration `json:"min_retry_backoff"`
	MaxRetryBackoff     time.Duration `json:"max_retry_backoff"`
}

// DefaultRedisPoolConfig returns a production-ready Redis pool configuration
func DefaultRedisPoolConfig() RedisPoolConfig {
	return RedisPoolConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,

		// Pool configuration optimized for high throughput
		PoolSize:      25,                 // Max connections in pool
		MinIdleConns:  5,                  // Keep minimum idle connections
		MaxConnAge:    5 * time.Minute,    // Rotate connections every 5 minutes
		PoolTimeout:   4 * time.Second,    // Wait up to 4 seconds for connection
		IdleTimeout:   10 * time.Minute,   // Close idle connections after 10 minutes
		IdleCheckFreq: 1 * time.Minute,    // Check for idle connections every minute

		// Timeout configuration
		DialTimeout:  5 * time.Second,     // Connection establishment timeout
		ReadTimeout:  3 * time.Second,     // Read operation timeout
		WriteTimeout: 3 * time.Second,     // Write operation timeout

		// Health check configuration
		HealthCheckInterval: 30 * time.Second,
		MaxRetries:          3,
		MinRetryBackoff:     8 * time.Millisecond,
		MaxRetryBackoff:     512 * time.Millisecond,
	}
}

// OptimizedRedisManager provides optimized Redis connection management
type OptimizedRedisManager struct {
	client       *redis.Client
	config       RedisPoolConfig
	healthStatus bool
	healthMutex  sync.RWMutex
	stopHealth   chan struct{}
	healthDone   chan struct{}
}

// NewOptimizedRedisManager creates a new optimized Redis manager
func NewOptimizedRedisManager(config RedisPoolConfig) (*OptimizedRedisManager, error) {
	options := &redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		
		// Pool configuration
		PoolSize:      config.PoolSize,
		MinIdleConns:  config.MinIdleConns,
		MaxConnAge:    config.MaxConnAge,
		PoolTimeout:   config.PoolTimeout,
		IdleTimeout:   config.IdleTimeout,
		IdleCheckFrequency: config.IdleCheckFreq,
		
		// Timeout configuration
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		
		// Retry configuration
		MaxRetries:      config.MaxRetries,
		MinRetryBackoff: config.MinRetryBackoff,
		MaxRetryBackoff: config.MaxRetryBackoff,
	}

	client := redis.NewClient(options)

	// Test initial connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", config.Addr, err)
	}

	manager := &OptimizedRedisManager{
		client:       client,
		config:       config,
		healthStatus: true,
		stopHealth:   make(chan struct{}),
		healthDone:   make(chan struct{}),
	}

	// Start health checking goroutine
	go manager.runHealthCheck()

	log.Printf("Optimized Redis manager initialized with pool size: %d, min idle: %d",
		config.PoolSize, config.MinIdleConns)

	return manager, nil
}

// GetClient returns the shared Redis client
func (r *OptimizedRedisManager) GetClient() *redis.Client {
	return r.client
}

// IsHealthy returns true if Redis connection is healthy
func (r *OptimizedRedisManager) IsHealthy() bool {
	r.healthMutex.RLock()
	defer r.healthMutex.RUnlock()
	return r.healthStatus
}

// GetPoolStats returns connection pool statistics
func (r *OptimizedRedisManager) GetPoolStats() *redis.PoolStats {
	return r.client.PoolStats()
}

// ExecuteWithRetry executes a Redis operation with retry logic
func (r *OptimizedRedisManager) ExecuteWithRetry(ctx context.Context, operation func(*redis.Client) error) error {
	var lastErr error
	maxRetries := r.config.MaxRetries
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt) * r.config.MinRetryBackoff
			if backoff > r.config.MaxRetryBackoff {
				backoff = r.config.MaxRetryBackoff
			}
			time.Sleep(backoff)
		}

		err := operation(r.client)
		if err == nil {
			return nil
		}

		lastErr = err
		
		// Check if error is retriable
		if !r.isRetriableError(err) {
			break
		}
		
		log.Printf("Redis operation failed (attempt %d/%d): %v", attempt+1, maxRetries+1, err)
	}

	return fmt.Errorf("redis operation failed after %d attempts: %w", maxRetries+1, lastErr)
}

// BatchScanQueues scans for queues using pipeline operations
func (r *OptimizedRedisManager) BatchScanQueues(ctx context.Context, pattern string) ([]string, error) {
	var allKeys []string
	
	err := r.ExecuteWithRetry(ctx, func(client *redis.Client) error {
		var cursor uint64
		var keys []string
		var err error
		
		for {
			// Use optimal batch size for scanning
			keys, cursor, err = client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				return err
			}
			
			allKeys = append(allKeys, keys...)
			
			if cursor == 0 {
				break
			}
		}
		
		return nil
	})
	
	return allKeys, err
}

// BatchPopFromQueues pops from multiple queues in a single pipeline operation
func (r *OptimizedRedisManager) BatchPopFromQueues(ctx context.Context, queues []string) (map[string][]redis.Z, error) {
	results := make(map[string][]redis.Z)
	
	if len(queues) == 0 {
		return results, nil
	}
	
	err := r.ExecuteWithRetry(ctx, func(client *redis.Client) error {
		pipe := client.Pipeline()
		cmds := make(map[string]*redis.ZSliceCmd)
		
		// Add all ZPopMin operations to pipeline
		for _, queue := range queues {
			cmds[queue] = pipe.ZPopMin(ctx, queue, 1)
		}
		
		_, err := pipe.Exec(ctx)
		if err != nil && err != redis.Nil {
			return err
		}
		
		// Collect results
		for queue, cmd := range cmds {
			if result, err := cmd.Result(); err == nil && len(result) > 0 {
				results[queue] = result
			}
		}
		
		return nil
	})
	
	return results, err
}

// Close gracefully closes the Redis manager
func (r *OptimizedRedisManager) Close() error {
	// Stop health checking
	close(r.stopHealth)
	<-r.healthDone

	// Close Redis client
	return r.client.Close()
}

// runHealthCheck performs periodic health checks
func (r *OptimizedRedisManager) runHealthCheck() {
	defer close(r.healthDone)
	
	ticker := time.NewTicker(r.config.HealthCheckInterval)
	defer ticker.Stop()

	log.Printf("Starting Redis health checks every %v", r.config.HealthCheckInterval)

	for {
		select {
		case <-r.stopHealth:
			log.Printf("Redis health checking stopped")
			return
		case <-ticker.C:
			r.performHealthCheck()
		}
	}
}

// performHealthCheck executes a single health check
func (r *OptimizedRedisManager) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	err := r.client.Ping(ctx).Err()
	latency := time.Since(start)

	r.healthMutex.Lock()
	defer r.healthMutex.Unlock()

	if err != nil {
		r.healthStatus = false
		log.Printf("Redis health check failed (latency: %v): %v", latency, err)
	} else {
		r.healthStatus = true
		log.Printf("Redis health check successful (latency: %v)", latency)
	}
}

// isRetriableError determines if an error is worth retrying
func (r *OptimizedRedisManager) isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	
	// Network errors and timeouts are retriable
	retriablePatterns := []string{
		"connection refused",
		"timeout",
		"network is unreachable",
		"broken pipe",
		"connection reset",
		"io: read/write on closed pipe",
		"use of closed network connection",
	}

	for _, pattern := range retriablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// HandoffPayload represents the structure of messages from the queue
type HandoffPayload struct {
	Metadata struct {
		ProjectName string    `json:"project_name"`
		FromAgent   string    `json:"from_agent"`
		ToAgent     string    `json:"to_agent"`
		Timestamp   time.Time `json:"timestamp"`
		TaskContext string    `json:"task_context"`
		Priority    string    `json:"priority"`
		HandoffID   string    `json:"handoff_id"`
	} `json:"metadata"`
	Content struct {
		Summary          string                 `json:"summary"`
		Requirements     []string               `json:"requirements"`
		Artifacts        map[string][]string    `json:"artifacts"`
		TechnicalDetails map[string]interface{} `json:"technical_details"`
		NextSteps        []string               `json:"next_steps"`
	} `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func main() {
	ctx := context.Background()
	
	// Configure Redis with optimized pool settings
	redisConfig := DefaultRedisPoolConfig()
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		redisConfig.Addr = addr
	}
	if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		redisConfig.Password = password
	}

	// Initialize optimized Redis manager
	redisManager, err := NewOptimizedRedisManager(redisConfig)
	if err != nil {
		log.Fatalf("Failed to initialize Redis manager: %v", err)
	}
	defer redisManager.Close()

	log.Printf("Optimized Agent Manager service started. Listening for tasks...")
	log.Printf("Redis address: %s", redisConfig.Addr)
	log.Printf("Pool size: %d, Min idle connections: %d", redisConfig.PoolSize, redisConfig.MinIdleConns)

	// Track metrics
	var processedTasks uint64
	var failedTasks uint64
	var totalLatency time.Duration
	metricsInterval := 30 * time.Second
	lastMetricsTime := time.Now()

	for {
		// Check Redis health before processing
		if !redisManager.IsHealthy() {
			log.Printf("Redis is unhealthy, waiting before retry...")
			time.Sleep(5 * time.Second)
			continue
		}

		// Use optimized batch scanning for queues
		queuePattern := "handoff:project:*:queue:*"
		queues, err := redisManager.BatchScanQueues(ctx, queuePattern)
		if err != nil {
			log.Printf("Error scanning for queues with pattern %s: %v", queuePattern, err)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(queues) == 0 {
			time.Sleep(2 * time.Second) // No active queues, wait a bit
			continue
		}

		// Use optimized batch popping from queues
		start := time.Now()
		results, err := redisManager.BatchPopFromQueues(ctx, queues)
		if err != nil {
			log.Printf("Error checking queues: %v", err)
			failedTasks++
			time.Sleep(1 * time.Second)
			continue
		}

		// Process results
		for queueName, items := range results {
			for _, item := range items {
				processStart := time.Now()
				
				handoffID := item.Member.(string)
				log.Printf("Received task from queue: %s, handoff ID: %s", queueName, handoffID)

				// Extract project and agent name from queue name
				projectName, agentName := extractProjectAndAgentName(queueName)
				if agentName == "" || projectName == "" {
					log.Printf("Could not extract project/agent name from queue: %s", queueName)
					continue
				}

				// Retrieve the full handoff data from Redis using optimized operations
				var taskPayload string
				handoffKey := fmt.Sprintf("handoff:%s", handoffID)
				
				err := redisManager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
					result, err := client.Get(ctx, handoffKey).Result()
					if err != nil {
						return err
					}
					taskPayload = result
					return nil
				})

				if err != nil {
					if err == redis.Nil {
						log.Printf("Handoff data not found for ID: %s", handoffID)
					} else {
						log.Printf("Error retrieving handoff %s: %v", handoffID, err)
						failedTasks++
					}
					continue
				}

				// Dispatch the task in a new goroutine
				go func(projectName, agentName, taskPayload string, processStart time.Time) {
					if err := dispatchAndArchiveTask(projectName, agentName, taskPayload); err != nil {
						log.Printf("Failed to dispatch task: %v", err)
						failedTasks++
					} else {
						processedTasks++
					}
					
					// Update latency metrics
					processingTime := time.Since(processStart)
					totalLatency += processingTime
				}(projectName, agentName, taskPayload, processStart)
			}
		}

		// Print metrics periodically
		if time.Since(lastMetricsTime) >= metricsInterval {
			poolStats := redisManager.GetPoolStats()
			avgLatency := time.Duration(0)
			if processedTasks > 0 {
				avgLatency = totalLatency / time.Duration(processedTasks)
			}
			
			log.Printf("Metrics - Processed: %d, Failed: %d, Avg Latency: %v", 
				processedTasks, failedTasks, avgLatency)
			log.Printf("Redis Pool - Total: %d, Idle: %d, Stale: %d, Hits: %d, Misses: %d, Timeouts: %d",
				poolStats.TotalConns, poolStats.IdleConns, poolStats.StaleConns, 
				poolStats.Hits, poolStats.Misses, poolStats.Timeouts)
			
			lastMetricsTime = time.Now()
		}

		// Small delay to prevent busy-waiting if all queues were empty
		time.Sleep(100 * time.Millisecond)
	}
}

// extractProjectAndAgentName extracts the project and agent name from a queue name
func extractProjectAndAgentName(queueName string) (string, string) {
	// Expected format: "handoff:project:{projectName}:queue:{agentName}"
	parts := strings.Split(queueName, ":")
	if len(parts) == 5 && parts[0] == "handoff" && parts[1] == "project" && parts[3] == "queue" {
		return parts[2], parts[4]
	}
	return "", ""
}

// dispatchAndArchiveTask handles the execution and archival of a single task
func dispatchAndArchiveTask(projectName, agentName, payload string) error {
	log.Printf("[Dispatch] Processing task for project '%s', agent '%s'", projectName, agentName)

	var handoff HandoffPayload
	if err := json.Unmarshal([]byte(payload), &handoff); err != nil {
		return fmt.Errorf("failed to decode task payload: %w", err)
	}

	handoffID := handoff.Metadata.HandoffID
	if handoffID == "" {
		return fmt.Errorf("missing handoff ID in payload")
	}

	log.Printf("[Dispatch] Invoking agent '%s' for handoff '%s' in project '%s'", agentName, handoffID, projectName)

	// Set environment variable for the agent
	env := os.Environ()
	env = append(env, fmt.Sprintf("AGENT_PROJECT_NAME=%s", projectName))

	cmd := exec.Command("./run-agent.sh", agentName, payload)
	cmd.Dir = "." // Ensure we're in the right directory
	cmd.Env = env // Pass the environment with the project name

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[FAILURE] Agent '%s' failed: %v\n--- Output ---\n%s\n--------------", agentName, err, string(output))
		return fmt.Errorf("agent execution failed: %w", err)
	}

	log.Printf("[SUCCESS] Agent '%s' completed for handoff '%s'.", agentName, handoffID)
	log.Printf("[OUTPUT]\n%s", string(output))

	if err := archiveHandoff(payload, &handoff, handoffID); err != nil {
		log.Printf("[CRITICAL] Agent '%s' succeeded but failed to archive: %v", agentName, err)
		return fmt.Errorf("archival failed: %w", err)
	}

	return nil
}

// archiveHandoff saves the successful handoff payload to the file system
func archiveHandoff(payload string, handoffData *HandoffPayload, handoffID string) error {
	var ts time.Time
	if !handoffData.Metadata.Timestamp.IsZero() {
		ts = handoffData.Metadata.Timestamp
	} else {
		ts = time.Now()
	}

	datePath := ts.UTC().Format("2006-01-02")
	// Include project name in archive path
	projectName := handoffData.Metadata.ProjectName
	if projectName == "" {
		projectName = "unknown-project"
	}

	fileName := fmt.Sprintf("%s-%s-%s.json",
		ts.UTC().Format("20060102T150405Z"),
		handoffData.Metadata.ToAgent,
		handoffID[:8])

	archiveDir := filepath.Join("archive", projectName, datePath)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	filePath := filepath.Join(archiveDir, fileName)
	log.Printf("[Archive] Saving handoff to %s", filePath)

	return os.WriteFile(filePath, []byte(payload), 0644)
}