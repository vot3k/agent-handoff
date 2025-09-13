package handoff

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// ExampleOptimizedUsage demonstrates how to use the optimized Redis handoff system
func ExampleOptimizedUsage() {
	// Initialize optimized Redis configuration
	redisConfig := DefaultRedisPoolConfig()
	
	// Override from environment variables if available
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		redisConfig.Addr = addr
	}
	if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		redisConfig.Password = password
	}

	// Create optimized agent configuration
	agentConfig := OptimizedConfig{
		RedisConfig: redisConfig,
		LogLevel:    "info",
		RetryPolicy: &RetryPolicy{
			MaxRetries:      3,
			InitialDelay:    time.Second,
			MaxDelay:        time.Minute,
			BackoffFactor:   2.0,
			RetriableErrors: []string{"connection error", "timeout", "temporary failure"},
		},
	}

	// Create optimized handoff agent
	agent, err := NewOptimizedHandoffAgent(agentConfig)
	if err != nil {
		log.Fatalf("Failed to create optimized handoff agent: %v", err)
	}
	defer agent.Close()

	// Register agent capabilities
	capabilities := []AgentCapabilities{
		{
			Name:          "golang-expert",
			Description:   "Go backend development expert",
			Triggers:      []string{"go", "golang", "backend"},
			InputTypes:    []string{"code", "specification"},
			OutputTypes:   []string{"implementation", "tests"},
			MaxConcurrent: 3,
		},
		{
			Name:          "test-expert",
			Description:   "Testing and QA expert",
			Triggers:      []string{"test", "testing", "qa"},
			InputTypes:    []string{"code", "requirements"},
			OutputTypes:   []string{"tests", "coverage"},
			MaxConcurrent: 2,
		},
		{
			Name:          "devops-expert",
			Description:   "DevOps and deployment expert",
			Triggers:      []string{"deploy", "docker", "k8s"},
			InputTypes:    []string{"code", "configuration"},
			OutputTypes:   []string{"deployment", "infrastructure"},
			MaxConcurrent: 2,
		},
	}

	for _, cap := range capabilities {
		if err := agent.RegisterAgent(cap); err != nil {
			log.Printf("Failed to register agent %s: %v", cap.Name, err)
		}
	}

	// Create optimized monitor
	redisManager := agent.GetRedisManager()
	monitor := NewOptimizedHandoffMonitor(redisManager)

	// Add alert rules
	alertRules := []AlertRule{
		{
			Name:      "High Queue Depth",
			Type:      AlertQueueDepth,
			Condition: "greater_than",
			Threshold: 10,
			Duration:  time.Minute,
			Enabled:   true,
			Cooldown:  5 * time.Minute,
		},
		{
			Name:      "High Processing Time",
			Type:      AlertProcessingTime,
			Condition: "greater_than",
			Threshold: 30000, // 30 seconds in milliseconds
			Duration:  2 * time.Minute,
			Enabled:   true,
			Cooldown:  10 * time.Minute,
		},
		{
			Name:      "System Health Critical",
			Type:      AlertSystemHealth,
			Condition: "less_than",
			Threshold: 50,
			Duration:  time.Minute,
			Enabled:   true,
			Cooldown:  5 * time.Minute,
		},
	}

	for _, rule := range alertRules {
		monitor.AddAlertRule(rule)
	}

	// Start monitoring in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitor.StartMonitoring(ctx, 30*time.Second)

	// Subscribe to critical alerts
	criticalAlerts := monitor.SubscribeToAlerts(AlertSystemHealth)
	go func() {
		for alert := range criticalAlerts {
			log.Printf("CRITICAL ALERT: %s - %s", alert.Rule.Name, alert.Message)
		}
	}()

	// Example: Publish a handoff from architect-expert to golang-expert
	handoff := &Handoff{
		Metadata: Metadata{
			ProjectName: "agent-handoff-optimization",
			FromAgent:   "architect-expert",
			ToAgent:     "golang-expert",
			TaskContext: "Optimize Redis connection pooling in handoff system",
			Priority:    PriorityHigh,
		},
		Content: Content{
			Summary: "Optimize the Redis connection pooling in the handoff system",
			Requirements: []string{
				"Implement connection pooling for Redis",
				"Add connection health checks",
				"Optimize memory usage",
			},
			TechnicalDetails: map[string]interface{}{
				"target_files": []string{
					"handoff/monitor.go",
					"handoff/router.go",
					"agent-manager Redis connection code",
				},
				"optimization_goals": []string{
					"Reduce connection overhead",
					"Improve fault tolerance",
					"Optimize memory usage",
				},
			},
			NextSteps: []string{
				"Analyze current Redis implementation",
				"Implement connection pooling optimizations",
				"Add health checks for Redis connections",
				"Optimize memory usage patterns",
				"Create handoff to test-expert for testing",
			},
		},
	}

	// Publish the handoff
	if err := agent.PublishHandoff(ctx, handoff); err != nil {
		log.Printf("Failed to publish handoff: %v", err)
		return
	}

	log.Printf("Successfully published handoff: %s", handoff.Metadata.HandoffID)

	// Start consuming handoffs for golang-expert
	go func() {
		err := agent.ConsumeHandoffs(ctx, "golang-expert", func(ctx context.Context, h *Handoff) error {
			log.Printf("Processing handoff: %s", h.Metadata.HandoffID)
			log.Printf("Task: %s", h.Content.Summary)
			
			// Simulate processing
			time.Sleep(2 * time.Second)
			
			// Record successful processing
			monitor.RecordHandoffMetrics(ctx, h, 2*time.Second, true)
			
			// After processing, create handoff to test-expert
			testHandoff := &Handoff{
				Metadata: Metadata{
					ProjectName: h.Metadata.ProjectName,
					FromAgent:   "golang-expert",
					ToAgent:     "test-expert",
					TaskContext: "Test the Redis optimization implementation",
					Priority:    PriorityNormal,
				},
				Content: Content{
					Summary: "Test handoff system Redis optimizations and create comprehensive test suite",
					Requirements: []string{
						"Test Redis connection pooling functionality",
						"Verify health check mechanisms",
						"Test memory optimization features",
						"Performance testing for connection pool",
						"Load testing with multiple concurrent operations",
					},
					Artifacts: Artifacts{
						Created: []string{
							"/Users/jimmy/Dev/ai-platforms/agent-handoff/handoff/redis_pool.go",
							"/Users/jimmy/Dev/ai-platforms/agent-handoff/handoff/redis_manager.go",
							"/Users/jimmy/Dev/ai-platforms/agent-handoff/handoff/agent_optimized.go",
							"/Users/jimmy/Dev/ai-platforms/agent-handoff/handoff/monitor_optimized.go",
						},
						Modified: []string{
							"/Users/jimmy/Dev/ai-platforms/agent-handoff/agent-manager/cmd/manager/main_optimized.go",
						},
					},
					TechnicalDetails: map[string]interface{}{
						"optimization_features": []string{
							"Connection pooling with configurable limits",
							"Health checks with automatic failover",
							"Memory optimization with LRU eviction",
							"Batch operations for better performance",
							"Retry logic with exponential backoff",
						},
						"test_coverage": []string{
							"Unit tests for connection pool",
							"Integration tests for health checks",
							"Performance benchmarks",
							"Load testing scenarios",
							"Failure recovery testing",
						},
						"performance_metrics": map[string]interface{}{
							"pool_size":               25,
							"min_idle_connections":    5,
							"max_connection_age":      "5m",
							"health_check_interval":   "30s",
							"connection_timeout":      "5s",
							"operation_timeout":       "3s",
						},
					},
					NextSteps: []string{
						"Create comprehensive test suite for Redis optimizations",
						"Implement performance benchmarks",
						"Test connection pool under load",
						"Verify health check reliability",
						"Create integration tests with actual Redis instances",
						"Document performance improvements",
					},
				},
			}
			
			// Publish to test-expert
			return agent.PublishHandoff(ctx, testHandoff)
		})
		
		if err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	// Monitor metrics for a while
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Second)
		
		// Get detailed metrics including Redis pool metrics
		handoffMetrics, redisMetrics := agent.GetOptimizedMetrics()
		healthStatus := agent.GetHealthStatus()
		
		log.Printf("=== Metrics Report ===")
		log.Printf("Handoffs - Total: %d, Completed: %d, Failed: %d, Queue Depth: %d",
			handoffMetrics.TotalHandoffs, handoffMetrics.CompletedHandoffs,
			handoffMetrics.FailedHandoffs, handoffMetrics.QueueDepth)
		
		log.Printf("Redis Pool - Total Conns: %d, Idle: %d, Hits: %d, Misses: %d",
			redisMetrics.TotalConns, redisMetrics.IdleConns,
			redisMetrics.Hits, redisMetrics.Misses)
		
		log.Printf("Redis Health - Healthy: %t, Last Check: %v, Failures: %d",
			healthStatus.IsHealthy, healthStatus.LastHealthCheck,
			healthStatus.ConsecutiveFailures)
		
		log.Printf("Performance - Avg Latency: %v, Max Latency: %v, Failed Requests: %d",
			redisMetrics.AvgLatency, redisMetrics.MaxLatency, redisMetrics.FailedRequests)
		
		// Perform maintenance periodically
		if i%3 == 0 {
			if err := agent.PerformMaintenance(ctx); err != nil {
				log.Printf("Maintenance failed: %v", err)
			}
		}
	}

	log.Printf("Example completed successfully!")
}

// ExampleRedisPoolConfiguration shows how to configure Redis pool for different environments
func ExampleRedisPoolConfiguration() {
	// Development configuration - lighter resource usage
	devConfig := RedisPoolConfig{
		Addr:            "localhost:6379",
		PoolSize:        10,
		MinIdleConns:    2,
		MaxConnAge:      2 * time.Minute,
		PoolTimeout:     2 * time.Second,
		IdleTimeout:     5 * time.Minute,
		IdleCheckFreq:   30 * time.Second,
		DialTimeout:     3 * time.Second,
		ReadTimeout:     2 * time.Second,
		WriteTimeout:    2 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxRetries:      2,
		MinRetryBackoff: 10 * time.Millisecond,
		MaxRetryBackoff: 200 * time.Millisecond,
	}

	// Production configuration - optimized for high throughput
	prodConfig := RedisPoolConfig{
		Addr:            "redis-cluster.prod:6379",
		PoolSize:        50,
		MinIdleConns:    10,
		MaxConnAge:      10 * time.Minute,
		PoolTimeout:     5 * time.Second,
		IdleTimeout:     15 * time.Minute,
		IdleCheckFreq:   1 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		MaxRetries:      5,
		MinRetryBackoff: 5 * time.Millisecond,
		MaxRetryBackoff: 1 * time.Second,
	}

	// High availability configuration - optimized for reliability
	haConfig := RedisPoolConfig{
		Addr:            "redis-ha.cluster:6379",
		PoolSize:        30,
		MinIdleConns:    8,
		MaxConnAge:      8 * time.Minute,
		PoolTimeout:     6 * time.Second,
		IdleTimeout:     12 * time.Minute,
		IdleCheckFreq:   45 * time.Second,
		DialTimeout:     7 * time.Second,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		HealthCheckInterval: 15 * time.Second,
		MaxRetries:      10,
		MinRetryBackoff: 50 * time.Millisecond,
		MaxRetryBackoff: 2 * time.Second,
	}

	// Use appropriate config based on environment
	env := os.Getenv("ENVIRONMENT")
	var selectedConfig RedisPoolConfig
	
	switch env {
	case "development":
		selectedConfig = devConfig
		log.Printf("Using development Redis configuration")
	case "production":
		selectedConfig = prodConfig
		log.Printf("Using production Redis configuration")
	case "ha", "high-availability":
		selectedConfig = haConfig
		log.Printf("Using high-availability Redis configuration")
	default:
		selectedConfig = DefaultRedisPoolConfig()
		log.Printf("Using default Redis configuration")
	}

	// Initialize Redis manager with selected configuration
	redisManager, err := NewRedisPoolManager(selectedConfig)
	if err != nil {
		log.Fatalf("Failed to initialize Redis with %s config: %v", env, err)
	}
	defer redisManager.Close()

	log.Printf("Redis pool initialized successfully for %s environment", env)
	log.Printf("Pool size: %d, Min idle: %d, Health check interval: %v",
		selectedConfig.PoolSize, selectedConfig.MinIdleConns, selectedConfig.HealthCheckInterval)
}

// init sets up structured logging for examples
func init() {
	zerolog.TimeFieldFormat = time.RFC3339
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}