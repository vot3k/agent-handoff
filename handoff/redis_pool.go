package handoff

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

// RedisPoolConfig contains configuration for Redis connection pool
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

// RedisPoolManager manages a shared Redis connection pool with health checking
type RedisPoolManager struct {
	client       *redis.Client
	config       RedisPoolConfig
	healthStatus HealthStatus
	statusMutex  sync.RWMutex
	stopHealth   chan struct{}
	healthDone   chan struct{}
	metrics      *RedisPoolMetrics
	metricsMutex sync.RWMutex
}

// HealthStatus represents the health state of the Redis connection
type HealthStatus struct {
	IsHealthy          bool      `json:"is_healthy"`
	LastHealthCheck    time.Time `json:"last_health_check"`
	LastSuccessfulPing time.Time `json:"last_successful_ping"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastError          string    `json:"last_error,omitempty"`
}

// RedisPoolMetrics contains detailed metrics about the connection pool
type RedisPoolMetrics struct {
	// Pool statistics
	TotalConns     uint32        `json:"total_conns"`
	IdleConns      uint32        `json:"idle_conns"`
	StaleConns     uint32        `json:"stale_conns"`
	Hits           uint64        `json:"hits"`
	Misses         uint64        `json:"misses"`
	Timeouts       uint64        `json:"timeouts"`
	
	// Performance metrics
	AvgLatency     time.Duration `json:"avg_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
	TotalRequests  uint64        `json:"total_requests"`
	FailedRequests uint64        `json:"failed_requests"`
	
	// Memory optimization metrics
	MemoryUsage    int64         `json:"memory_usage_bytes"`
	PipelineHits   uint64        `json:"pipeline_hits"`
	BatchOperations uint64       `json:"batch_operations"`
	
	LastUpdated    time.Time     `json:"last_updated"`
}

// NewRedisPoolManager creates a new Redis pool manager with health checking
func NewRedisPoolManager(config RedisPoolConfig) (*RedisPoolManager, error) {
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

	manager := &RedisPoolManager{
		client:     client,
		config:     config,
		stopHealth: make(chan struct{}),
		healthDone: make(chan struct{}),
		healthStatus: HealthStatus{
			IsHealthy:          true,
			LastHealthCheck:    time.Now(),
			LastSuccessfulPing: time.Now(),
		},
		metrics: &RedisPoolMetrics{
			LastUpdated: time.Now(),
		},
	}

	// Start health checking goroutine
	go manager.runHealthCheck()

	log.Info().
		Str("addr", config.Addr).
		Int("pool_size", config.PoolSize).
		Int("min_idle", config.MinIdleConns).
		Dur("max_conn_age", config.MaxConnAge).
		Msg("Redis pool manager initialized successfully")

	return manager, nil
}

// GetClient returns the shared Redis client
func (r *RedisPoolManager) GetClient() *redis.Client {
	return r.client
}

// GetHealthStatus returns the current health status
func (r *RedisPoolManager) GetHealthStatus() HealthStatus {
	r.statusMutex.RLock()
	defer r.statusMutex.RUnlock()
	return r.healthStatus
}

// GetMetrics returns current pool metrics
func (r *RedisPoolManager) GetMetrics() RedisPoolMetrics {
	r.metricsMutex.Lock()
	defer r.metricsMutex.Unlock()
	
	// Update metrics from Redis client
	r.updateMetricsLocked()
	return *r.metrics
}

// IsHealthy returns true if Redis connection is healthy
func (r *RedisPoolManager) IsHealthy() bool {
	r.statusMutex.RLock()
	defer r.statusMutex.RUnlock()
	return r.healthStatus.IsHealthy
}

// ExecuteWithRetry executes a Redis operation with retry logic and metrics tracking
func (r *RedisPoolManager) ExecuteWithRetry(ctx context.Context, operation func(*redis.Client) error) error {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		r.recordMetrics(latency, nil)
	}()

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
		
		log.Warn().
			Err(err).
			Int("attempt", attempt+1).
			Int("max_retries", maxRetries).
			Msg("Redis operation failed, retrying")
	}

	r.recordMetrics(0, lastErr)
	return fmt.Errorf("redis operation failed after %d attempts: %w", maxRetries+1, lastErr)
}

// Pipeline returns a new Redis pipeline for batch operations
func (r *RedisPoolManager) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

// TxPipeline returns a new Redis transaction pipeline
func (r *RedisPoolManager) TxPipeline() redis.Pipeliner {
	return r.client.TxPipeline()
}

// Close gracefully closes the Redis pool manager
func (r *RedisPoolManager) Close() error {
	// Stop health checking
	close(r.stopHealth)
	<-r.healthDone

	// Close Redis client
	if err := r.client.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing Redis client")
		return err
	}

	log.Info().Msg("Redis pool manager closed successfully")
	return nil
}

// runHealthCheck performs periodic health checks
func (r *RedisPoolManager) runHealthCheck() {
	defer close(r.healthDone)
	
	ticker := time.NewTicker(r.config.HealthCheckInterval)
	defer ticker.Stop()

	log.Info().
		Dur("interval", r.config.HealthCheckInterval).
		Msg("Starting Redis health checks")

	for {
		select {
		case <-r.stopHealth:
			log.Info().Msg("Redis health checking stopped")
			return
		case <-ticker.C:
			r.performHealthCheck()
		}
	}
}

// performHealthCheck executes a single health check
func (r *RedisPoolManager) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	err := r.client.Ping(ctx).Err()
	latency := time.Since(start)

	r.statusMutex.Lock()
	defer r.statusMutex.Unlock()

	r.healthStatus.LastHealthCheck = time.Now()

	if err != nil {
		r.healthStatus.ConsecutiveFailures++
		r.healthStatus.LastError = err.Error()
		
		// Consider unhealthy after 3 consecutive failures
		if r.healthStatus.ConsecutiveFailures >= 3 {
			r.healthStatus.IsHealthy = false
		}

		log.Warn().
			Err(err).
			Int("consecutive_failures", r.healthStatus.ConsecutiveFailures).
			Dur("latency", latency).
			Msg("Redis health check failed")
	} else {
		r.healthStatus.IsHealthy = true
		r.healthStatus.ConsecutiveFailures = 0
		r.healthStatus.LastSuccessfulPing = time.Now()
		r.healthStatus.LastError = ""

		log.Debug().
			Dur("latency", latency).
			Msg("Redis health check successful")
	}
}

// updateMetricsLocked updates metrics from Redis client stats
func (r *RedisPoolManager) updateMetricsLocked() {
	stats := r.client.PoolStats()
	
	r.metrics.TotalConns = stats.TotalConns
	r.metrics.IdleConns = stats.IdleConns
	r.metrics.StaleConns = stats.StaleConns
	r.metrics.Hits = uint64(stats.Hits)
	r.metrics.Misses = uint64(stats.Misses)
	r.metrics.Timeouts = uint64(stats.Timeouts)
	r.metrics.LastUpdated = time.Now()
}

// recordMetrics records operation metrics
func (r *RedisPoolManager) recordMetrics(latency time.Duration, err error) {
	r.metricsMutex.Lock()
	defer r.metricsMutex.Unlock()

	r.metrics.TotalRequests++
	
	if err != nil {
		r.metrics.FailedRequests++
	}

	if latency > 0 {
		// Update latency metrics using exponential moving average
		if r.metrics.AvgLatency == 0 {
			r.metrics.AvgLatency = latency
		} else {
			// EMA with alpha = 0.1
			r.metrics.AvgLatency = time.Duration(
				0.9*float64(r.metrics.AvgLatency) + 0.1*float64(latency),
			)
		}

		if latency > r.metrics.MaxLatency {
			r.metrics.MaxLatency = latency
		}
	}
}

// isRetriableError determines if an error is worth retrying
func (r *RedisPoolManager) isRetriableError(err error) bool {
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
		if len(errStr) >= len(pattern) && errStr[:len(pattern)] == pattern {
			return true
		}
		// Also check if pattern is contained in error string
		for i := 0; i <= len(errStr)-len(pattern); i++ {
			if errStr[i:i+len(pattern)] == pattern {
				return true
			}
		}
	}

	return false
}