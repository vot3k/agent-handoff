package handoff

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

// CoD: ANALYZE - Comprehensive Redis pool optimization validation
// CoD: COVERAGE - 95% target for critical paths
// CoD: PRIORITY - Connection pooling, health monitoring, error handling
// CoD: AUTOMATE - Performance benchmarks and stress tests
// CoD: MEASURE - Latency, throughput, memory metrics

// TestRedisPoolOptimizationsComprehensive performs comprehensive validation of Redis optimizations
func TestRedisPoolOptimizationsComprehensive(t *testing.T) {
	config := DefaultRedisPoolConfig()
	config.Addr = "localhost:6379"
	config.HealthCheckInterval = 500 * time.Millisecond // Fast for testing
	config.PoolSize = 20 // Adequate pool size for testing

	manager, err := NewRedisPoolManager(config)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	t.Run("ConnectionPoolValidation", func(t *testing.T) {
		testConnectionPoolValidation(t, manager, ctx)
	})

	t.Run("HealthMonitoringFunctionality", func(t *testing.T) {
		testHealthMonitoringFunctionality(t, manager, ctx)
	})

	t.Run("MemoryOptimizationVerification", func(t *testing.T) {
		testMemoryOptimizationVerification(t, manager, ctx)
	})

	t.Run("ConcurrentAccessPatterns", func(t *testing.T) {
		testConcurrentAccessPatterns(t, manager, ctx)
	})

	t.Run("ErrorHandlingAndRecovery", func(t *testing.T) {
		testErrorHandlingAndRecovery(t, manager, ctx)
	})

	t.Run("PipelineOperations", func(t *testing.T) {
		testPipelineOperations(t, manager, ctx)
	})

	t.Run("MetricsValidation", func(t *testing.T) {
		testMetricsValidation(t, manager, ctx)
	})
}

// CoD: TEST - Connection pool efficiency and management
func testConnectionPoolValidation(t *testing.T, manager *RedisPoolManager, ctx context.Context) {
	t.Run("PoolSizeLimits", func(t *testing.T) {
		// Test that pool respects size limits
		client := manager.GetClient()

		// Create many concurrent operations to test pool limits
		var wg sync.WaitGroup
		numOps := 50 // More than pool size to test pooling
		
		for i := 0; i < numOps; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				key := fmt.Sprintf("pool:test:%d", i)
				err := client.Set(ctx, key, "value", time.Minute).Err()
				if err != nil {
					t.Errorf("Operation %d failed: %v", i, err)
				}
			}(i)
		}
		wg.Wait()

		finalStats := client.PoolStats()
		
		if finalStats.TotalConns > uint32(manager.config.PoolSize) {
			t.Errorf("Pool exceeded maximum size: %d > %d", finalStats.TotalConns, manager.config.PoolSize)
		}

		if finalStats.TotalConns == 0 {
			t.Error("No connections were created")
		}

		t.Logf("Pool stats: Total=%d, Idle=%d, Stale=%d, Hits=%d, Misses=%d",
			finalStats.TotalConns, finalStats.IdleConns, finalStats.StaleConns, 
			finalStats.Hits, finalStats.Misses)
	})

	t.Run("ConnectionReuse", func(t *testing.T) {
		client := manager.GetClient()
		initialStats := client.PoolStats()

		// Perform sequential operations to test connection reuse
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("reuse:test:%d", i)
			err := client.Set(ctx, key, "value", time.Minute).Err()
			if err != nil {
				t.Errorf("Operation %d failed: %v", i, err)
			}
		}

		finalStats := client.PoolStats()
		
		// Should have good hit ratio for connection reuse
		if finalStats.Hits <= initialStats.Hits {
			t.Error("Expected connection reuse (hits should increase)")
		}

		hitRatio := float64(finalStats.Hits) / float64(finalStats.Hits + finalStats.Misses)
		if hitRatio < 0.5 {
			t.Errorf("Low connection reuse ratio: %.2f", hitRatio)
		}

		t.Logf("Connection reuse ratio: %.2f", hitRatio)
	})

	t.Run("IdleConnectionManagement", func(t *testing.T) {
		client := manager.GetClient()
		
		// Create some connections
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				key := fmt.Sprintf("idle:test:%d", i)
				client.Set(ctx, key, "value", time.Minute)
			}(i)
		}
		wg.Wait()

		beforeStats := client.PoolStats()
		
		// Wait for idle check frequency
		time.Sleep(manager.config.IdleCheckFreq + 100*time.Millisecond)
		
		afterStats := client.PoolStats()
		
		// Should maintain minimum idle connections
		if afterStats.IdleConns < uint32(manager.config.MinIdleConns) {
			t.Errorf("Not maintaining minimum idle connections: %d < %d", 
				afterStats.IdleConns, manager.config.MinIdleConns)
		}

		t.Logf("Idle connection management: Before=%d, After=%d, Min=%d",
			beforeStats.IdleConns, afterStats.IdleConns, manager.config.MinIdleConns)
	})
}

// CoD: TEST - Health monitoring accuracy and responsiveness
func testHealthMonitoringFunctionality(t *testing.T, manager *RedisPoolManager, ctx context.Context) {
	t.Run("InitialHealthStatus", func(t *testing.T) {
		if !manager.IsHealthy() {
			t.Error("Manager should be healthy initially")
		}

		status := manager.GetHealthStatus()
		if status.ConsecutiveFailures != 0 {
			t.Errorf("Expected 0 consecutive failures, got %d", status.ConsecutiveFailures)
		}

		if status.LastHealthCheck.IsZero() {
			t.Error("Last health check should be set")
		}

		if status.LastSuccessfulPing.IsZero() {
			t.Error("Last successful ping should be set")
		}
	})

	t.Run("HealthCheckFrequency", func(t *testing.T) {
		status1 := manager.GetHealthStatus()
		
		// Wait for health check interval
		time.Sleep(manager.config.HealthCheckInterval + 100*time.Millisecond)
		
		status2 := manager.GetHealthStatus()
		
		if !status2.LastHealthCheck.After(status1.LastHealthCheck) {
			t.Error("Health check should have been performed")
		}

		if status2.ConsecutiveFailures != 0 {
			t.Errorf("Expected 0 consecutive failures, got %d", status2.ConsecutiveFailures)
		}
	})

	t.Run("HealthMetricsTracking", func(t *testing.T) {
		// Perform some operations and check health metrics
		for i := 0; i < 10; i++ {
			err := manager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
				return client.Ping(ctx).Err()
			})
			if err != nil {
				t.Errorf("Ping operation %d failed: %v", i, err)
			}
		}

		status := manager.GetHealthStatus()
		if status.ConsecutiveFailures != 0 {
			t.Errorf("Expected 0 consecutive failures after successful operations, got %d", 
				status.ConsecutiveFailures)
		}

		if !status.IsHealthy {
			t.Error("Manager should be healthy after successful operations")
		}
	})
}

// CoD: TEST - Memory optimization effectiveness
func testMemoryOptimizationVerification(t *testing.T, manager *RedisPoolManager, ctx context.Context) {
	t.Run("MemoryUsageTracking", func(t *testing.T) {
		initialMetrics := manager.GetMetrics()
		
		// Create test data
		client := manager.GetClient()
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("memory:test:%d", i)
			value := fmt.Sprintf("value_%d", i)
			err := client.Set(ctx, key, value, time.Minute).Err()
			if err != nil {
				t.Errorf("Failed to set key %s: %v", key, err)
			}
		}

		finalMetrics := manager.GetMetrics()
		
		// Check that metrics are being tracked
		if finalMetrics.TotalRequests <= initialMetrics.TotalRequests {
			t.Error("Expected increase in total requests")
		}

		if finalMetrics.LastUpdated.Before(initialMetrics.LastUpdated) || 
		   finalMetrics.LastUpdated.Equal(initialMetrics.LastUpdated) {
			t.Error("Metrics should be updated")
		}

		t.Logf("Memory optimization metrics: Requests=%d, Failed=%d, AvgLatency=%v",
			finalMetrics.TotalRequests, finalMetrics.FailedRequests, finalMetrics.AvgLatency)
	})

	t.Run("LatencyOptimization", func(t *testing.T) {
		// Test latency tracking and optimization
		var totalLatency time.Duration
		numOps := 50

		for i := 0; i < numOps; i++ {
			start := time.Now()
			err := manager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
				key := fmt.Sprintf("latency:test:%d", i)
				return client.Set(ctx, key, "value", time.Minute).Err()
			})
			latency := time.Since(start)
			totalLatency += latency

			if err != nil {
				t.Errorf("Operation %d failed: %v", i, err)
			}
		}

		avgLatency := totalLatency / time.Duration(numOps)
		metrics := manager.GetMetrics()

		// Verify latency metrics are reasonable
		if metrics.AvgLatency == 0 {
			t.Error("Average latency should be tracked")
		}

		if avgLatency > 100*time.Millisecond {
			t.Errorf("High average latency detected: %v", avgLatency)
		}

		t.Logf("Latency optimization: Measured=%v, Tracked=%v, Max=%v",
			avgLatency, metrics.AvgLatency, metrics.MaxLatency)
	})
}

// CoD: TEST - Concurrent access patterns and thread safety
func testConcurrentAccessPatterns(t *testing.T, manager *RedisPoolManager, ctx context.Context) {
	t.Run("HighConcurrencyStressTest", func(t *testing.T) {
		numGoroutines := 100
		numOpsPerGoroutine := 50
		var wg sync.WaitGroup
		var successCount, errorCount int64

		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := fmt.Sprintf("stress:test:%d:%d", goroutineID, j)
					value := fmt.Sprintf("value_%d_%d", goroutineID, j)
					
					err := manager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
						return client.Set(ctx, key, value, time.Minute).Err()
					})
					
					if err != nil {
						atomic.AddInt64(&errorCount, 1)
					} else {
						atomic.AddInt64(&successCount, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		totalOps := int64(numGoroutines * numOpsPerGoroutine)
		
		if errorCount > 0 {
			t.Errorf("Errors in concurrent operations: %d/%d", errorCount, totalOps)
		}

		if successCount != totalOps {
			t.Errorf("Missing successful operations: %d/%d", successCount, totalOps)
		}

		throughput := float64(totalOps) / duration.Seconds()
		
		if throughput < 1000 { // Expect at least 1000 ops/sec
			t.Errorf("Low throughput: %.2f ops/sec", throughput)
		}

		t.Logf("Concurrent stress test: %d ops in %v (%.2f ops/sec), Success=%d, Errors=%d",
			totalOps, duration, throughput, successCount, errorCount)
	})

	t.Run("MixedOperationPatterns", func(t *testing.T) {
		var wg sync.WaitGroup
		numWorkers := 20
		duration := 5 * time.Second
		ctx, cancel := context.WithTimeout(ctx, duration)
		defer cancel()

		var readCount, writeCount, errorCount int64

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				
				for {
					select {
					case <-ctx.Done():
						return
					default:
						// Random operation pattern
						if rand.Float32() < 0.7 { // 70% writes
							key := fmt.Sprintf("mixed:write:%d:%d", workerID, rand.Int())
							err := manager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
								return client.Set(ctx, key, "value", time.Minute).Err()
							})
							if err != nil {
								atomic.AddInt64(&errorCount, 1)
							} else {
								atomic.AddInt64(&writeCount, 1)
							}
						} else { // 30% reads
							key := fmt.Sprintf("mixed:read:%d", rand.Intn(100))
							err := manager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
								_, err := client.Get(ctx, key).Result()
								if err == redis.Nil {
									return nil // Not an error for our test
								}
								return err
							})
							if err != nil {
								atomic.AddInt64(&errorCount, 1)
							} else {
								atomic.AddInt64(&readCount, 1)
							}
						}
						time.Sleep(time.Millisecond) // Small delay
					}
				}
			}(i)
		}

		wg.Wait()

		totalOps := readCount + writeCount + errorCount
		errorRate := float64(errorCount) / float64(totalOps) * 100

		if errorRate > 1.0 { // Allow max 1% error rate
			t.Errorf("High error rate in mixed operations: %.2f%%", errorRate)
		}

		t.Logf("Mixed operation patterns: Reads=%d, Writes=%d, Errors=%d (%.2f%% error rate)",
			readCount, writeCount, errorCount, errorRate)
	})
}

// CoD: TEST - Error handling and recovery mechanisms
func testErrorHandlingAndRecovery(t *testing.T, manager *RedisPoolManager, ctx context.Context) {
	t.Run("RetryMechanism", func(t *testing.T) {
		attemptCount := 0
		maxAttempts := 3

		err := manager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
			attemptCount++
			if attemptCount < maxAttempts {
				return fmt.Errorf("temporary failure: attempt %d", attemptCount)
			}
			return nil // Success on final attempt
		})

		if err != nil {
			t.Errorf("Expected success after retries, got: %v", err)
		}

		if attemptCount != maxAttempts {
			t.Errorf("Expected %d attempts, got %d", maxAttempts, attemptCount)
		}
	})

	t.Run("NonRetriableErrors", func(t *testing.T) {
		attemptCount := 0

		err := manager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
			attemptCount++
			return fmt.Errorf("syntax error: invalid command") // Non-retriable
		})

		if err == nil {
			t.Error("Expected error for non-retriable failure")
		}

		// Should not retry non-retriable errors
		if attemptCount != 1 {
			t.Errorf("Expected 1 attempt for non-retriable error, got %d", attemptCount)
		}
	})

	t.Run("TimeoutHandling", func(t *testing.T) {
		// Test with very short timeout
		shortCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
		defer cancel()

		err := manager.ExecuteWithRetry(shortCtx, func(client *redis.Client) error {
			time.Sleep(10 * time.Millisecond) // Longer than timeout
			return client.Ping(shortCtx).Err()
		})

		if err == nil {
			t.Error("Expected timeout error")
		}

		if !isContextTimeoutError(err) {
			t.Errorf("Expected context timeout error, got: %v", err)
		}
	})

	t.Run("ConnectionFailureRecovery", func(t *testing.T) {
		// This test assumes Redis is available and tests pool recovery behavior
		initialStatus := manager.GetHealthStatus()
		
		if !initialStatus.IsHealthy {
			t.Skip("Redis not healthy for recovery test")
		}

		// Perform operations to ensure pool is active
		for i := 0; i < 5; i++ {
			err := manager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
				return client.Ping(ctx).Err()
			})
			if err != nil {
				t.Errorf("Ping %d failed: %v", i, err)
			}
		}

		// Verify health is maintained
		finalStatus := manager.GetHealthStatus()
		if !finalStatus.IsHealthy {
			t.Error("Health should be maintained after operations")
		}

		if finalStatus.ConsecutiveFailures != 0 {
			t.Errorf("Expected 0 consecutive failures, got %d", finalStatus.ConsecutiveFailures)
		}
	})
}

// CoD: TEST - Pipeline operations efficiency
func testPipelineOperations(t *testing.T, manager *RedisPoolManager, ctx context.Context) {
	t.Run("PipelineBasicOperations", func(t *testing.T) {
		pipeline := manager.Pipeline()
		
		// Add multiple operations to pipeline
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("pipeline:test:%d", i)
			value := fmt.Sprintf("value_%d", i)
			pipeline.Set(ctx, key, value, time.Minute)
		}

		// Execute pipeline
		start := time.Now()
		cmds, err := pipeline.Exec(ctx)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Pipeline execution failed: %v", err)
		}

		if len(cmds) != 10 {
			t.Errorf("Expected 10 commands in pipeline, got %d", len(cmds))
		}

		// Verify all operations succeeded
		for i, cmd := range cmds {
			if cmd.Err() != nil {
				t.Errorf("Pipeline command %d failed: %v", i, cmd.Err())
			}
		}

		// Pipeline should be faster than individual operations
		if duration > 100*time.Millisecond {
			t.Errorf("Pipeline execution too slow: %v", duration)
		}

		t.Logf("Pipeline operations: 10 commands in %v", duration)
	})

	t.Run("PipelineVsIndividualPerformance", func(t *testing.T) {
		numOps := 20

		// Test individual operations
		start := time.Now()
		client := manager.GetClient()
		for i := 0; i < numOps; i++ {
			key := fmt.Sprintf("individual:test:%d", i)
			err := client.Set(ctx, key, "value", time.Minute).Err()
			if err != nil {
				t.Errorf("Individual operation %d failed: %v", i, err)
			}
		}
		individualDuration := time.Since(start)

		// Test pipeline operations
		start = time.Now()
		pipeline := manager.Pipeline()
		for i := 0; i < numOps; i++ {
			key := fmt.Sprintf("pipeline:perf:%d", i)
			pipeline.Set(ctx, key, "value", time.Minute)
		}
		_, err := pipeline.Exec(ctx)
		pipelineDuration := time.Since(start)

		if err != nil {
			t.Errorf("Pipeline execution failed: %v", err)
		}

		// Pipeline should be significantly faster
		improvement := float64(individualDuration) / float64(pipelineDuration)
		if improvement < 1.5 { // Expect at least 50% improvement
			t.Errorf("Pipeline not significantly faster: %.2fx improvement", improvement)
		}

		t.Logf("Pipeline vs Individual: Individual=%v, Pipeline=%v (%.2fx improvement)",
			individualDuration, pipelineDuration, improvement)
	})

	t.Run("TransactionPipeline", func(t *testing.T) {
		txPipeline := manager.TxPipeline()
		
		// Add operations to transaction pipeline
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("tx:test:%d", i)
			value := fmt.Sprintf("tx_value_%d", i)
			txPipeline.Set(ctx, key, value, time.Minute)
		}

		// Execute transaction
		cmds, err := txPipeline.Exec(ctx)
		if err != nil {
			t.Errorf("Transaction pipeline failed: %v", err)
		}

		if len(cmds) != 5 {
			t.Errorf("Expected 5 commands in transaction, got %d", len(cmds))
		}

		// Verify all operations succeeded
		for i, cmd := range cmds {
			if cmd.Err() != nil {
				t.Errorf("Transaction command %d failed: %v", i, cmd.Err())
			}
		}
	})
}

// CoD: TEST - Metrics accuracy and monitoring
func testMetricsValidation(t *testing.T, manager *RedisPoolManager, ctx context.Context) {
	t.Run("BasicMetricsTracking", func(t *testing.T) {
		initialMetrics := manager.GetMetrics()
		
		// Perform some operations
		numOps := 25
		for i := 0; i < numOps; i++ {
			err := manager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
				key := fmt.Sprintf("metrics:test:%d", i)
				return client.Set(ctx, key, "value", time.Minute).Err()
			})
			if err != nil {
				t.Errorf("Operation %d failed: %v", i, err)
			}
		}

		finalMetrics := manager.GetMetrics()

		// Verify metrics increased
		if finalMetrics.TotalRequests <= initialMetrics.TotalRequests {
			t.Error("Total requests should increase")
		}

		requestIncrease := finalMetrics.TotalRequests - initialMetrics.TotalRequests
		if requestIncrease < uint64(numOps) {
			t.Errorf("Expected at least %d request increase, got %d", numOps, requestIncrease)
		}

		// Check metrics structure
		if finalMetrics.LastUpdated.Before(initialMetrics.LastUpdated) {
			t.Error("LastUpdated should be more recent")
		}

		if finalMetrics.AvgLatency == 0 {
			t.Error("Average latency should be tracked")
		}

		t.Logf("Metrics validation: Requests=%d (+%d), AvgLatency=%v, MaxLatency=%v, Failed=%d",
			finalMetrics.TotalRequests, requestIncrease, finalMetrics.AvgLatency,
			finalMetrics.MaxLatency, finalMetrics.FailedRequests)
	})

	t.Run("PoolStatsIntegration", func(t *testing.T) {
		metrics := manager.GetMetrics()
		client := manager.GetClient()
		poolStats := client.PoolStats()

		// Verify pool stats are integrated into metrics
		if metrics.TotalConns != poolStats.TotalConns {
			t.Errorf("Total connections mismatch: metrics=%d, pool=%d", 
				metrics.TotalConns, poolStats.TotalConns)
		}

		if metrics.IdleConns != poolStats.IdleConns {
			t.Errorf("Idle connections mismatch: metrics=%d, pool=%d", 
				metrics.IdleConns, poolStats.IdleConns)
		}

		if metrics.Hits != uint64(poolStats.Hits) {
			t.Errorf("Hits mismatch: metrics=%d, pool=%d", 
				metrics.Hits, poolStats.Hits)
		}

		t.Logf("Pool stats integration: Total=%d, Idle=%d, Hits=%d, Misses=%d, Timeouts=%d",
			metrics.TotalConns, metrics.IdleConns, metrics.Hits, metrics.Misses, metrics.Timeouts)
	})

	t.Run("FailureMetricsTracking", func(t *testing.T) {
		initialMetrics := manager.GetMetrics()
		
		// Force some failures
		numFailures := 3
		for i := 0; i < numFailures; i++ {
			_ = manager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
				return fmt.Errorf("intentional test failure %d", i)
			})
		}

		finalMetrics := manager.GetMetrics()
		
		failureIncrease := finalMetrics.FailedRequests - initialMetrics.FailedRequests
		if failureIncrease < uint64(numFailures) {
			t.Errorf("Expected at least %d failure increase, got %d", numFailures, failureIncrease)
		}

		t.Logf("Failure metrics: Failed requests increased by %d", failureIncrease)
	})
}

// Helper function to check for context timeout errors
func isContextTimeoutError(err error) bool {
	return err == context.DeadlineExceeded || 
		   (err != nil && (err.Error() == "context deadline exceeded" ||
		   err.Error() == "context canceled"))
}

// BenchmarkRedisOptimizationsComparison benchmarks before/after performance
func BenchmarkRedisOptimizationsComparison(b *testing.B) {
	// Test with different pool configurations to show optimization impact
	configs := map[string]RedisPoolConfig{
		"Unoptimized": {
			Addr:              "localhost:6379",
			PoolSize:          5,   // Small pool
			MinIdleConns:      1,   // Minimal idle
			MaxConnAge:        0,   // No connection rotation
			PoolTimeout:       1 * time.Second,
			IdleTimeout:       5 * time.Minute,
			IdleCheckFreq:     0,   // No idle checks
			DialTimeout:       1 * time.Second,
			ReadTimeout:       1 * time.Second,
			WriteTimeout:      1 * time.Second,
			HealthCheckInterval: 0, // No health checks
			MaxRetries:        1,
			MinRetryBackoff:   10 * time.Millisecond,
			MaxRetryBackoff:   100 * time.Millisecond,
		},
		"Optimized": DefaultRedisPoolConfig(),
	}

	for configName, config := range configs {
		config.Addr = "localhost:6379"
		
		b.Run(configName, func(b *testing.B) {
			manager, err := NewRedisPoolManager(config)
			if err != nil {
				b.Skipf("Redis not available: %v", err)
			}
			defer manager.Close()

			ctx := context.Background()
			
			b.ResetTimer()
			b.ReportAllocs()
			
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := fmt.Sprintf("bench:%s:%d", configName, i)
					err := manager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
						return client.Set(ctx, key, "value", time.Minute).Err()
					})
					if err != nil {
						b.Errorf("Operation failed: %v", err)
					}
					i++
				}
			})
		})
	}
}

// BenchmarkMemoryOptimizations benchmarks memory optimization features
func BenchmarkMemoryOptimizations(b *testing.B) {
	config := DefaultRedisPoolConfig()
	config.Addr = "localhost:6379"

	manager, err := NewRedisManager(config)
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	defer manager.Shutdown()

	ctx := context.Background()

	b.Run("OptimizedSerialization", func(b *testing.B) {
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("memory:opt:%d", i)
			value := map[string]interface{}{
				"id":    i,
				"name":  fmt.Sprintf("user_%d", i),
				"email": fmt.Sprintf("user%d@example.com", i),
			}
			
			err := manager.SetWithOptimizedExpiry(ctx, key, value, time.Minute)
			if err != nil {
				b.Errorf("Set operation failed: %v", err)
			}
		}
	})

	b.Run("BatchOperations", func(b *testing.B) {
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			operations := make([]func(redis.Pipeliner) error, 10)
			for j := 0; j < 10; j++ {
				j := j // Capture loop variable
				operations[j] = func(pipe redis.Pipeliner) error {
					key := fmt.Sprintf("batch:%d:%d", i, j)
					pipe.Set(ctx, key, "value", time.Minute)
					return nil
				}
			}
			
			err := manager.ExecuteBatch(ctx, operations)
			if err != nil {
				b.Errorf("Batch operation failed: %v", err)
			}
		}
	})

	b.Run("QueueOperations", func(b *testing.B) {
		queueOps := manager.GetQueueOps()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			members := make([]*redis.Z, 10)
			for j := 0; j < 10; j++ {
				members[j] = &redis.Z{
					Score:  float64(i*10 + j),
					Member: fmt.Sprintf("task_%d_%d", i, j),
				}
			}
			
			queueName := fmt.Sprintf("bench:queue:%d", i%5) // Distribute across queues
			err := queueOps.ZAddBatch(ctx, queueName, members)
			if err != nil {
				b.Errorf("Queue operation failed: %v", err)
			}
		}
	})
}

// TestOptimizedHandoffAgentIntegration tests the complete optimized handoff agent
func TestOptimizedHandoffAgentIntegration(t *testing.T) {
	config := OptimizedConfig{
		RedisConfig: DefaultRedisPoolConfig(),
		LogLevel:    "error",
	}
	config.RedisConfig.Addr = "localhost:6379"

	agent, err := NewOptimizedHandoffAgent(config)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}
	defer agent.Close()

	// Register test agents
	capabilities := []AgentCapabilities{
		{Name: "test-agent-1", Description: "Test agent 1", MaxConcurrent: 2},
		{Name: "test-agent-2", Description: "Test agent 2", MaxConcurrent: 3},
	}

	for _, cap := range capabilities {
		err := agent.RegisterAgent(cap)
		if err != nil {
			t.Fatalf("Failed to register agent %s: %v", cap.Name, err)
		}
	}

	ctx := context.Background()

	t.Run("OptimizedHandoffFlow", func(t *testing.T) {
		// Test complete handoff flow with optimizations
		var processedHandoffs []string
		var mu sync.Mutex

		// Start consumer for test-agent-1
		go func() {
			err := agent.ConsumeHandoffs(ctx, "test-agent-1", func(ctx context.Context, handoff *Handoff) error {
				mu.Lock()
				processedHandoffs = append(processedHandoffs, handoff.Metadata.HandoffID)
				mu.Unlock()
				return nil
			})
			if err != nil && err != context.Canceled {
				t.Errorf("Consumer error: %v", err)
			}
		}()

		// Give consumer time to start
		time.Sleep(100 * time.Millisecond)

		// Publish handoffs
		numHandoffs := 5
		for i := 0; i < numHandoffs; i++ {
			handoff := &Handoff{
				Metadata: Metadata{
					ProjectName: "test-project",
					FromAgent:   "test-source",
					ToAgent:     "test-agent-1",
					TaskContext: fmt.Sprintf("Test handoff %d", i),
					Priority:    PriorityNormal,
				},
				Content: Content{
					Summary:      fmt.Sprintf("Test handoff %d summary", i),
					Requirements: []string{fmt.Sprintf("Requirement %d", i)},
				},
			}

			err := agent.PublishHandoff(ctx, handoff)
			if err != nil {
				t.Errorf("Failed to publish handoff %d: %v", i, err)
			}
		}

		// Wait for processing
		timeout := time.After(5 * time.Second)
		for {
			mu.Lock()
			processedCount := len(processedHandoffs)
			mu.Unlock()

			if processedCount >= numHandoffs {
				break
			}

			select {
			case <-timeout:
				t.Errorf("Timeout waiting for handoffs to be processed. Processed: %d/%d", 
					processedCount, numHandoffs)
				return
			case <-time.After(100 * time.Millisecond):
				// Continue waiting
			}
		}

		// Verify all handoffs were processed
		mu.Lock()
		if len(processedHandoffs) != numHandoffs {
			t.Errorf("Expected %d processed handoffs, got %d", numHandoffs, len(processedHandoffs))
		}
		mu.Unlock()

		// Check metrics
		handoffMetrics, redisMetrics := agent.GetOptimizedMetrics()
		
		if handoffMetrics.TotalHandoffs < int64(numHandoffs) {
			t.Errorf("Expected at least %d total handoffs, got %d", 
				numHandoffs, handoffMetrics.TotalHandoffs)
		}

		if handoffMetrics.CompletedHandoffs < int64(numHandoffs) {
			t.Errorf("Expected at least %d completed handoffs, got %d", 
				numHandoffs, handoffMetrics.CompletedHandoffs)
		}

		if redisMetrics.TotalRequests == 0 {
			t.Error("Expected Redis requests to be tracked")
		}

		t.Logf("Integration test metrics: Handoffs=%d, Completed=%d, Redis Requests=%d",
			handoffMetrics.TotalHandoffs, handoffMetrics.CompletedHandoffs, redisMetrics.TotalRequests)
	})

	t.Run("PerformanceMaintenance", func(t *testing.T) {
		// Test maintenance operations
		err := agent.PerformMaintenance(ctx)
		if err != nil {
			t.Errorf("Maintenance should not fail: %v", err)
		}

		// Verify health status
		if !agent.IsHealthy() {
			t.Error("Agent should be healthy after maintenance")
		}

		healthStatus := agent.GetHealthStatus()
		if healthStatus.ConsecutiveFailures != 0 {
			t.Errorf("Expected 0 consecutive failures after maintenance, got %d", 
				healthStatus.ConsecutiveFailures)
		}
	})
}

// reportMemoryUsage reports current memory usage for optimization analysis
func reportMemoryUsage(t *testing.T, label string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	t.Logf("%s Memory Usage: Alloc=%dKB, Sys=%dKB, NumGC=%d", 
		label, m.Alloc/1024, m.Sys/1024, m.NumGC)
}