package handoff

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

// TestRedisPoolManager tests the optimized Redis pool manager
func TestRedisPoolManager(t *testing.T) {
	config := DefaultRedisPoolConfig()
	config.Addr = "localhost:6379"
	config.HealthCheckInterval = 1 * time.Second // Faster for testing

	manager, err := NewRedisPoolManager(config)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Test basic operations
	t.Run("BasicOperations", func(t *testing.T) {
		client := manager.GetClient()
		
		// Test SET/GET
		err := client.Set(ctx, "test:key", "test:value", time.Minute).Err()
		if err != nil {
			t.Fatalf("Failed to set key: %v", err)
		}

		val, err := client.Get(ctx, "test:key").Result()
		if err != nil {
			t.Fatalf("Failed to get key: %v", err)
		}

		if val != "test:value" {
			t.Errorf("Expected 'test:value', got '%s'", val)
		}
	})

	// Test health checking
	t.Run("HealthCheck", func(t *testing.T) {
		// Wait for at least one health check
		time.Sleep(2 * time.Second)
		
		if !manager.IsHealthy() {
			t.Error("Manager should be healthy")
		}

		status := manager.GetHealthStatus()
		if status.ConsecutiveFailures != 0 {
			t.Errorf("Expected 0 consecutive failures, got %d", status.ConsecutiveFailures)
		}
	})

	// Test retry mechanism
	t.Run("RetryMechanism", func(t *testing.T) {
		attempts := 0
		err := manager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
			attempts++
			if attempts < 3 {
				return fmt.Errorf("temporary failure")
			}
			return nil
		})

		if err != nil {
			t.Fatalf("Expected success after retries, got: %v", err)
		}

		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})

	// Test metrics collection
	t.Run("MetricsCollection", func(t *testing.T) {
		// Perform some operations to generate metrics
		for i := 0; i < 10; i++ {
			manager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
				return client.Set(ctx, fmt.Sprintf("metric:test:%d", i), "value", time.Minute).Err()
			})
		}

		metrics := manager.GetMetrics()
		if metrics.TotalRequests == 0 {
			t.Error("Expected some total requests in metrics")
		}
	})
}

// TestRedisManager tests the centralized Redis manager
func TestRedisManager(t *testing.T) {
	config := DefaultRedisPoolConfig()
	config.Addr = "localhost:6379"

	manager, err := NewRedisManager(config)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}
	defer manager.Shutdown()

	ctx := context.Background()

	t.Run("OptimizedOperations", func(t *testing.T) {
		// Test optimized set/get
		err := manager.SetWithOptimizedExpiry(ctx, "test:optimized", "optimized:value", time.Minute)
		if err != nil {
			t.Fatalf("Failed to set with optimized expiry: %v", err)
		}

		var result string
		err = manager.GetWithDeserialization(ctx, "test:optimized", &result)
		if err != nil {
			t.Fatalf("Failed to get with deserialization: %v", err)
		}

		if result != "optimized:value" {
			t.Errorf("Expected 'optimized:value', got '%s'", result)
		}
	})

	t.Run("BatchOperations", func(t *testing.T) {
		operations := []func(redis.Pipeliner) error{
			func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, "batch:1", "value1", time.Minute)
				return nil
			},
			func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, "batch:2", "value2", time.Minute)
				return nil
			},
			func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, "batch:3", "value3", time.Minute)
				return nil
			},
		}

		err := manager.ExecuteBatch(ctx, operations)
		if err != nil {
			t.Fatalf("Failed to execute batch operations: %v", err)
		}

		// Verify all keys were set
		client := manager.GetClient()
		for i := 1; i <= 3; i++ {
			key := fmt.Sprintf("batch:%d", i)
			val, err := client.Get(ctx, key).Result()
			if err != nil {
				t.Errorf("Failed to get key %s: %v", key, err)
			}
			expected := fmt.Sprintf("value%d", i)
			if val != expected {
				t.Errorf("Expected '%s', got '%s'", expected, val)
			}
		}
	})

	t.Run("QueueOperations", func(t *testing.T) {
		queueOps := manager.GetQueueOps()
		
		// Test batch queue operations
		members := []*redis.Z{
			{Score: 1.0, Member: "task1"},
			{Score: 2.0, Member: "task2"},
			{Score: 3.0, Member: "task3"},
		}

		err := queueOps.ZAddBatch(ctx, "test:queue", members)
		if err != nil {
			t.Fatalf("Failed to add batch to queue: %v", err)
		}

		// Test batch pop operations
		results, err := queueOps.ZPopMinBatch(ctx, []string{"test:queue"}, 2)
		if err != nil {
			t.Fatalf("Failed to pop batch from queue: %v", err)
		}

		queueResults, exists := results["test:queue"]
		if !exists {
			t.Fatal("Expected results for test:queue")
		}

		if len(queueResults) != 2 {
			t.Errorf("Expected 2 results, got %d", len(queueResults))
		}

		// Verify order (lowest score first)
		if queueResults[0].Member != "task1" || queueResults[1].Member != "task2" {
			t.Errorf("Queue results not in expected order")
		}
	})
}

// TestOptimizedHandoffAgent tests the optimized handoff agent
func TestOptimizedHandoffAgent(t *testing.T) {
	config := OptimizedConfig{
		RedisConfig: DefaultRedisPoolConfig(),
		LogLevel:    "error", // Reduce noise in tests
	}
	config.RedisConfig.Addr = "localhost:6379"

	agent, err := NewOptimizedHandoffAgent(config)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}
	defer agent.Close()

	// Register test agent
	cap := AgentCapabilities{
		Name:          "test-agent",
		Description:   "Test agent for optimized operations",
		MaxConcurrent: 2,
	}
	
	err = agent.RegisterAgent(cap)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	ctx := context.Background()

	t.Run("PublishHandoff", func(t *testing.T) {
		handoff := &Handoff{
			Metadata: Metadata{
				ProjectName: "test-project",
				FromAgent:   "test-source",
				ToAgent:     "test-agent",
				TaskContext: "Test optimized handoff publishing",
				Priority:    PriorityNormal,
			},
			Content: Content{
				Summary:      "Test handoff for optimization verification",
				Requirements: []string{"Test requirement 1", "Test requirement 2"},
			},
		}

		err := agent.PublishHandoff(ctx, handoff)
		if err != nil {
			t.Fatalf("Failed to publish handoff: %v", err)
		}

		if handoff.Metadata.HandoffID == "" {
			t.Error("Expected handoff ID to be set")
		}
	})

	t.Run("HealthAndMetrics", func(t *testing.T) {
		if !agent.IsHealthy() {
			t.Error("Agent should be healthy")
		}

		handoffMetrics, redisMetrics := agent.GetOptimizedMetrics()
		if handoffMetrics.LastUpdated.IsZero() {
			t.Error("Expected metrics to have LastUpdated timestamp")
		}

		if redisMetrics.LastUpdated.IsZero() {
			t.Error("Expected Redis metrics to have LastUpdated timestamp")
		}
	})

	t.Run("MaintenanceOperations", func(t *testing.T) {
		err := agent.PerformMaintenance(ctx)
		if err != nil {
			t.Errorf("Maintenance should not fail: %v", err)
		}
	})
}

// BenchmarkRedisOperations benchmarks Redis operations with and without optimizations
func BenchmarkRedisOperations(b *testing.B) {
	config := DefaultRedisPoolConfig()
	config.Addr = "localhost:6379"

	manager, err := NewRedisManager(config)
	if err != nil {
		b.Skipf("Redis not available for benchmarking: %v", err)
	}
	defer manager.Shutdown()

	ctx := context.Background()

	b.Run("SingleOperations", func(b *testing.B) {
		client := manager.GetClient()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("bench:single:%d", i)
			client.Set(ctx, key, "value", time.Minute)
			client.Get(ctx, key)
		}
	})

	b.Run("BatchOperations", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			operations := make([]func(redis.Pipeliner) error, 10)
			for j := 0; j < 10; j++ {
				j := j // Capture loop variable
				operations[j] = func(pipe redis.Pipeliner) error {
					key := fmt.Sprintf("bench:batch:%d:%d", i, j)
					pipe.Set(ctx, key, "value", time.Minute)
					return nil
				}
			}
			manager.ExecuteBatch(ctx, operations)
		}
	})

	b.Run("ConcurrentOperations", func(b *testing.B) {
		b.ResetTimer()

		var wg sync.WaitGroup
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				key := fmt.Sprintf("bench:concurrent:%d", i)
				manager.SetWithOptimizedExpiry(ctx, key, "value", time.Minute)
				var result string
				manager.GetWithDeserialization(ctx, key, &result)
			}(i)
		}
		wg.Wait()
	})
}

// BenchmarkConnectionPoolEfficiency benchmarks connection pool efficiency
func BenchmarkConnectionPoolEfficiency(b *testing.B) {
	config := DefaultRedisPoolConfig()
	config.Addr = "localhost:6379"
	config.PoolSize = 10 // Limit pool size for testing

	manager, err := NewRedisPoolManager(config)
	if err != nil {
		b.Skipf("Redis not available for benchmarking: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()
	client := manager.GetClient()

	b.Run("HighConcurrency", func(b *testing.B) {
		b.SetParallelism(20) // More than pool size
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("bench:pool:%d", i)
				client.Set(ctx, key, "value", time.Minute)
				client.Get(ctx, key)
				i++
			}
		})
	})

	b.Run("LowConcurrency", func(b *testing.B) {
		b.SetParallelism(2) // Less than pool size
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("bench:pool:low:%d", i)
				client.Set(ctx, key, "value", time.Minute)
				client.Get(ctx, key)
				i++
			}
		})
	})
}

// TestMemoryOptimizations tests memory optimization features
func TestMemoryOptimizations(t *testing.T) {
	config := DefaultRedisPoolConfig()
	config.Addr = "localhost:6379"

	manager, err := NewRedisManager(config)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}
	defer manager.Shutdown()

	ctx := context.Background()

	t.Run("CleanupExpiredKeys", func(t *testing.T) {
		// Create some test keys that will expire
		client := manager.GetClient()
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("cleanup:test:%d", i)
			client.Set(ctx, key, "value", 100*time.Millisecond)
		}

		// Wait for expiration
		time.Sleep(200 * time.Millisecond)

		// Cleanup expired keys
		patterns := []string{"cleanup:test:*"}
		err := manager.CleanupExpiredKeys(ctx, patterns)
		if err != nil {
			t.Errorf("Cleanup should not fail: %v", err)
		}
	})

	t.Run("SetMemoryOptimizations", func(t *testing.T) {
		err := manager.SetMemoryOptimizations(ctx)
		if err != nil {
			// This might fail on some Redis configurations, but shouldn't crash
			t.Logf("Memory optimizations warning (not critical): %v", err)
		}
	})
}

// TestHealthCheckRecovery tests health check and recovery mechanisms
func TestHealthCheckRecovery(t *testing.T) {
	config := DefaultRedisPoolConfig()
	config.Addr = "localhost:6379"
	config.HealthCheckInterval = 100 * time.Millisecond // Fast for testing

	manager, err := NewRedisPoolManager(config)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}
	defer manager.Close()

	// Wait for initial health check
	time.Sleep(200 * time.Millisecond)

	if !manager.IsHealthy() {
		t.Error("Manager should be healthy initially")
	}

	// Test health status structure
	status := manager.GetHealthStatus()
	if status.LastHealthCheck.IsZero() {
		t.Error("Expected LastHealthCheck to be set")
	}

	if status.ConsecutiveFailures != 0 {
		t.Errorf("Expected 0 consecutive failures, got %d", status.ConsecutiveFailures)
	}

	// Wait for a few more health checks
	time.Sleep(500 * time.Millisecond)

	// Verify health status is still good
	status = manager.GetHealthStatus()
	if status.ConsecutiveFailures != 0 {
		t.Errorf("Expected 0 consecutive failures after time, got %d", status.ConsecutiveFailures)
	}
}