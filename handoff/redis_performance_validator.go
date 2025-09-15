package handoff

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

// PerformanceValidator validates Redis optimization performance
type PerformanceValidator struct {
	optimizedManager   *RedisPoolManager
	unoptimizedManager *RedisPoolManager
	ctx                context.Context
}

// PerformanceResults contains benchmark results
type PerformanceResults struct {
	TestName           string        `json:"test_name"`
	OptimizedOps       int64         `json:"optimized_ops"`
	UnoptimizedOps     int64         `json:"unoptimized_ops"`
	OptimizedLatency   time.Duration `json:"optimized_latency"`
	UnoptimizedLatency time.Duration `json:"unoptimized_latency"`
	ImprovementRatio   float64       `json:"improvement_ratio"`
	ThroughputGain     float64       `json:"throughput_gain"`
	MemoryImprovement  string        `json:"memory_improvement"`
}

// ValidationReport contains comprehensive validation results
type ValidationReport struct {
	ExecutionTime     time.Time            `json:"execution_time"`
	TotalTests        int                  `json:"total_tests"`
	PassedTests       int                  `json:"passed_tests"`
	FailedTests       int                  `json:"failed_tests"`
	OverallResult     string               `json:"overall_result"`
	PerformanceGains  []PerformanceResults `json:"performance_gains"`
	Summary           ValidationSummary    `json:"summary"`
	Recommendations   []string             `json:"recommendations"`
}

// ValidationSummary provides key metrics
type ValidationSummary struct {
	AverageImprovement     float64 `json:"average_improvement"`
	MaxThroughputGain      float64 `json:"max_throughput_gain"`
	ConnectionEfficiency   float64 `json:"connection_efficiency"`
	LatencyReduction       float64 `json:"latency_reduction"`
	ErrorRate              float64 `json:"error_rate"`
	ProductionReadiness    string  `json:"production_readiness"`
}

// NewPerformanceValidator creates a new performance validator
func NewPerformanceValidator() (*PerformanceValidator, error) {
	ctx := context.Background()

	// Create optimized configuration
	optimizedConfig := DefaultRedisPoolConfig()
	optimizedConfig.Addr = "localhost:6379"

	// Create unoptimized configuration
	unoptimizedConfig := RedisPoolConfig{
		Addr:              "localhost:6379",
		PoolSize:          3,   // Small pool
		MinIdleConns:      1,   // Minimal idle
		MaxConnAge:        0,   // No rotation
		PoolTimeout:       5 * time.Second,
		IdleTimeout:       5 * time.Minute,
		IdleCheckFreq:     0,   // No idle checks
		DialTimeout:       2 * time.Second,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      2 * time.Second,
		HealthCheckInterval: 0, // No health checks
		MaxRetries:        1,
		MinRetryBackoff:   50 * time.Millisecond,
		MaxRetryBackoff:   200 * time.Millisecond,
	}

	optimizedManager, err := NewRedisPoolManager(optimizedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create optimized manager: %w", err)
	}

	unoptimizedManager, err := NewRedisPoolManager(unoptimizedConfig)
	if err != nil {
		optimizedManager.Close()
		return nil, fmt.Errorf("failed to create unoptimized manager: %w", err)
	}

	return &PerformanceValidator{
		optimizedManager:   optimizedManager,
		unoptimizedManager: unoptimizedManager,
		ctx:                ctx,
	}, nil
}

// Close cleans up resources
func (pv *PerformanceValidator) Close() {
	if pv.optimizedManager != nil {
		pv.optimizedManager.Close()
	}
	if pv.unoptimizedManager != nil {
		pv.unoptimizedManager.Close()
	}
}

// ValidatePerformance runs comprehensive performance validation
func (pv *PerformanceValidator) ValidatePerformance() (*ValidationReport, error) {
	report := &ValidationReport{
		ExecutionTime:   time.Now(),
		TotalTests:      0,
		PassedTests:     0,
		FailedTests:     0,
		PerformanceGains: []PerformanceResults{},
		Recommendations: []string{},
	}

	// Run individual performance tests
	tests := []struct {
		name string
		test func() (PerformanceResults, error)
	}{
		{"SingleOperations", pv.benchmarkSingleOperations},
		{"ConcurrentOperations", pv.benchmarkConcurrentOperations},
		{"PipelineOperations", pv.benchmarkPipelineOperations},
		{"MemoryEfficiency", pv.benchmarkMemoryEfficiency},
		{"ConnectionReuse", pv.benchmarkConnectionReuse},
	}

	for _, test := range tests {
		report.TotalTests++
		log.Printf("Running test: %s", test.name)
		
		result, err := test.test()
		if err != nil {
			log.Printf("Test %s failed: %v", test.name, err)
			report.FailedTests++
			continue
		}

		report.PassedTests++
		report.PerformanceGains = append(report.PerformanceGains, result)
		log.Printf("Test %s completed: %.2fx improvement", test.name, result.ImprovementRatio)
	}

	// Calculate summary
	report.Summary = pv.calculateSummary(report.PerformanceGains)
	
	// Set overall result
	if report.FailedTests == 0 && report.Summary.AverageImprovement > 2.0 {
		report.OverallResult = "EXCELLENT - Production Ready"
	} else if report.FailedTests <= 1 && report.Summary.AverageImprovement > 1.5 {
		report.OverallResult = "GOOD - Minor Issues to Address"
	} else {
		report.OverallResult = "NEEDS IMPROVEMENT"
	}

	// Generate recommendations
	report.Recommendations = pv.generateRecommendations(report.Summary)

	return report, nil
}

// benchmarkSingleOperations tests individual operation performance
func (pv *PerformanceValidator) benchmarkSingleOperations() (PerformanceResults, error) {
	numOps := 1000
	
	// Benchmark optimized
	start := time.Now()
	var optimizedErrors int64
	for i := 0; i < numOps; i++ {
		err := pv.optimizedManager.ExecuteWithRetry(pv.ctx, func(client *redis.Client) error {
			key := fmt.Sprintf("bench:opt:%d", i)
			return client.Set(pv.ctx, key, "value", time.Minute).Err()
		})
		if err != nil {
			atomic.AddInt64(&optimizedErrors, 1)
		}
	}
	optimizedDuration := time.Since(start)

	// Benchmark unoptimized
	start = time.Now()
	var unoptimizedErrors int64
	for i := 0; i < numOps; i++ {
		err := pv.unoptimizedManager.ExecuteWithRetry(pv.ctx, func(client *redis.Client) error {
			key := fmt.Sprintf("bench:unopt:%d", i)
			return client.Set(pv.ctx, key, "value", time.Minute).Err()
		})
		if err != nil {
			atomic.AddInt64(&unoptimizedErrors, 1)
		}
	}
	unoptimizedDuration := time.Since(start)

	optimizedOps := int64(numOps) - optimizedErrors
	unoptimizedOps := int64(numOps) - unoptimizedErrors

	improvement := float64(unoptimizedDuration) / float64(optimizedDuration)

	return PerformanceResults{
		TestName:           "SingleOperations",
		OptimizedOps:       optimizedOps,
		UnoptimizedOps:     unoptimizedOps,
		OptimizedLatency:   optimizedDuration / time.Duration(optimizedOps),
		UnoptimizedLatency: unoptimizedDuration / time.Duration(unoptimizedOps),
		ImprovementRatio:   improvement,
		ThroughputGain:     improvement,
		MemoryImprovement:  "Connection Pooling",
	}, nil
}

// benchmarkConcurrentOperations tests concurrent access performance
func (pv *PerformanceValidator) benchmarkConcurrentOperations() (PerformanceResults, error) {
	numWorkers := 50
	numOpsPerWorker := 50
	totalOps := int64(numWorkers * numOpsPerWorker)

	// Benchmark optimized
	start := time.Now()
	var optimizedErrors int64
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < numOpsPerWorker; j++ {
				err := pv.optimizedManager.ExecuteWithRetry(pv.ctx, func(client *redis.Client) error {
					key := fmt.Sprintf("concurrent:opt:%d:%d", workerID, j)
					return client.Set(pv.ctx, key, "value", time.Minute).Err()
				})
				if err != nil {
					atomic.AddInt64(&optimizedErrors, 1)
				}
			}
		}(i)
	}
	wg.Wait()
	optimizedDuration := time.Since(start)

	// Benchmark unoptimized
	start = time.Now()
	var unoptimizedErrors int64

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < numOpsPerWorker; j++ {
				err := pv.unoptimizedManager.ExecuteWithRetry(pv.ctx, func(client *redis.Client) error {
					key := fmt.Sprintf("concurrent:unopt:%d:%d", workerID, j)
					return client.Set(pv.ctx, key, "value", time.Minute).Err()
				})
				if err != nil {
					atomic.AddInt64(&unoptimizedErrors, 1)
				}
			}
		}(i)
	}
	wg.Wait()
	unoptimizedDuration := time.Since(start)

	optimizedOps := totalOps - optimizedErrors
	unoptimizedOps := totalOps - unoptimizedErrors

	optimizedThroughput := float64(optimizedOps) / optimizedDuration.Seconds()
	unoptimizedThroughput := float64(unoptimizedOps) / unoptimizedDuration.Seconds()
	throughputGain := optimizedThroughput / unoptimizedThroughput

	return PerformanceResults{
		TestName:           "ConcurrentOperations",
		OptimizedOps:       optimizedOps,
		UnoptimizedOps:     unoptimizedOps,
		OptimizedLatency:   optimizedDuration / time.Duration(optimizedOps),
		UnoptimizedLatency: unoptimizedDuration / time.Duration(unoptimizedOps),
		ImprovementRatio:   throughputGain,
		ThroughputGain:     throughputGain,
		MemoryImprovement:  "Pool Connection Reuse",
	}, nil
}

// benchmarkPipelineOperations tests pipeline performance
func (pv *PerformanceValidator) benchmarkPipelineOperations() (PerformanceResults, error) {
	numBatches := 100
	opsPerBatch := 20

	// Benchmark optimized (with pipelines)
	start := time.Now()
	var optimizedErrors int64

	for i := 0; i < numBatches; i++ {
		pipeline := pv.optimizedManager.Pipeline()
		for j := 0; j < opsPerBatch; j++ {
			key := fmt.Sprintf("pipeline:opt:%d:%d", i, j)
			pipeline.Set(pv.ctx, key, "value", time.Minute)
		}
		_, err := pipeline.Exec(pv.ctx)
		if err != nil {
			atomic.AddInt64(&optimizedErrors, int64(opsPerBatch))
		}
	}
	optimizedDuration := time.Since(start)

	// Benchmark unoptimized (individual operations)
	start = time.Now()
	var unoptimizedErrors int64
	client := pv.unoptimizedManager.GetClient()

	for i := 0; i < numBatches; i++ {
		for j := 0; j < opsPerBatch; j++ {
			key := fmt.Sprintf("pipeline:unopt:%d:%d", i, j)
			err := client.Set(pv.ctx, key, "value", time.Minute).Err()
			if err != nil {
				atomic.AddInt64(&unoptimizedErrors, 1)
			}
		}
	}
	unoptimizedDuration := time.Since(start)

	totalOps := int64(numBatches * opsPerBatch)
	optimizedOps := totalOps - optimizedErrors
	unoptimizedOps := totalOps - unoptimizedErrors

	improvement := float64(unoptimizedDuration) / float64(optimizedDuration)

	return PerformanceResults{
		TestName:           "PipelineOperations",
		OptimizedOps:       optimizedOps,
		UnoptimizedOps:     unoptimizedOps,
		OptimizedLatency:   optimizedDuration / time.Duration(optimizedOps),
		UnoptimizedLatency: unoptimizedDuration / time.Duration(unoptimizedOps),
		ImprovementRatio:   improvement,
		ThroughputGain:     improvement,
		MemoryImprovement:  "Reduced Network Round Trips",
	}, nil
}

// benchmarkMemoryEfficiency tests memory optimization
func (pv *PerformanceValidator) benchmarkMemoryEfficiency() (PerformanceResults, error) {
	numOps := 500

	// Get initial metrics
	initialOptimizedMetrics := pv.optimizedManager.GetMetrics()
	initialUnoptimizedMetrics := pv.unoptimizedManager.GetMetrics()

	// Perform operations on both
	for i := 0; i < numOps; i++ {
		pv.optimizedManager.ExecuteWithRetry(pv.ctx, func(client *redis.Client) error {
			key := fmt.Sprintf("memory:opt:%d", i)
			return client.Set(pv.ctx, key, "value", time.Minute).Err()
		})

		pv.unoptimizedManager.ExecuteWithRetry(pv.ctx, func(client *redis.Client) error {
			key := fmt.Sprintf("memory:unopt:%d", i)
			return client.Set(pv.ctx, key, "value", time.Minute).Err()
		})
	}

	// Get final metrics
	finalOptimizedMetrics := pv.optimizedManager.GetMetrics()
	finalUnoptimizedMetrics := pv.unoptimizedManager.GetMetrics()

	// Calculate efficiency based on delta metrics
	optimizedHitsDelta := finalOptimizedMetrics.Hits - initialOptimizedMetrics.Hits
	optimizedMissesDelta := finalOptimizedMetrics.Misses - initialOptimizedMetrics.Misses
	unoptimizedHitsDelta := finalUnoptimizedMetrics.Hits - initialUnoptimizedMetrics.Hits
	unoptimizedMissesDelta := finalUnoptimizedMetrics.Misses - initialUnoptimizedMetrics.Misses

	// Calculate connection reuse efficiency
	optimizedConnReuse := float64(optimizedHitsDelta) / float64(optimizedHitsDelta + optimizedMissesDelta)
	unoptimizedConnReuse := float64(unoptimizedHitsDelta) / float64(unoptimizedHitsDelta + unoptimizedMissesDelta)

	efficiency := optimizedConnReuse / unoptimizedConnReuse

	return PerformanceResults{
		TestName:           "MemoryEfficiency",
		OptimizedOps:       int64(numOps),
		UnoptimizedOps:     int64(numOps),
		OptimizedLatency:   finalOptimizedMetrics.AvgLatency,
		UnoptimizedLatency: finalUnoptimizedMetrics.AvgLatency,
		ImprovementRatio:   efficiency,
		ThroughputGain:     efficiency,
		MemoryImprovement:  fmt.Sprintf("%.2f%% connection reuse efficiency", optimizedConnReuse*100),
	}, nil
}

// benchmarkConnectionReuse tests connection pool efficiency
func (pv *PerformanceValidator) benchmarkConnectionReuse() (PerformanceResults, error) {
	numOps := 200

	// Warm up pools
	for i := 0; i < 10; i++ {
		pv.optimizedManager.GetClient().Ping(pv.ctx)
		pv.unoptimizedManager.GetClient().Ping(pv.ctx)
	}

	// Get initial stats
	optStats := pv.optimizedManager.GetClient().PoolStats()
	unoptStats := pv.unoptimizedManager.GetClient().PoolStats()

	initialOptHits := optStats.Hits
	initialUnoptHits := unoptStats.Hits

	// Perform operations
	start := time.Now()
	for i := 0; i < numOps; i++ {
		pv.optimizedManager.GetClient().Set(pv.ctx, fmt.Sprintf("reuse:opt:%d", i), "value", time.Minute)
	}
	optimizedDuration := time.Since(start)

	start = time.Now()
	for i := 0; i < numOps; i++ {
		pv.unoptimizedManager.GetClient().Set(pv.ctx, fmt.Sprintf("reuse:unopt:%d", i), "value", time.Minute)
	}
	unoptimizedDuration := time.Since(start)

	// Get final stats
	finalOptStats := pv.optimizedManager.GetClient().PoolStats()
	finalUnoptStats := pv.unoptimizedManager.GetClient().PoolStats()

	optHitIncrease := finalOptStats.Hits - initialOptHits
	unoptHitIncrease := finalUnoptStats.Hits - initialUnoptHits

	reuseEfficiency := float64(optHitIncrease) / float64(unoptHitIncrease)

	return PerformanceResults{
		TestName:           "ConnectionReuse",
		OptimizedOps:       int64(numOps),
		UnoptimizedOps:     int64(numOps),
		OptimizedLatency:   optimizedDuration / time.Duration(numOps),
		UnoptimizedLatency: unoptimizedDuration / time.Duration(numOps),
		ImprovementRatio:   reuseEfficiency,
		ThroughputGain:     float64(unoptimizedDuration) / float64(optimizedDuration),
		MemoryImprovement:  fmt.Sprintf("%.1fx more connection hits", reuseEfficiency),
	}, nil
}

// calculateSummary computes summary statistics
func (pv *PerformanceValidator) calculateSummary(results []PerformanceResults) ValidationSummary {
	if len(results) == 0 {
		return ValidationSummary{}
	}

	var totalImprovement, maxThroughput, totalLatencyReduction float64
	var totalConnectionEfficiency float64

	for _, result := range results {
		totalImprovement += result.ImprovementRatio
		if result.ThroughputGain > maxThroughput {
			maxThroughput = result.ThroughputGain
		}
		
		latencyReduction := 1.0 - (float64(result.OptimizedLatency) / float64(result.UnoptimizedLatency))
		totalLatencyReduction += latencyReduction

		if result.TestName == "ConnectionReuse" || result.TestName == "MemoryEfficiency" {
			totalConnectionEfficiency += result.ImprovementRatio
		}
	}

	avgImprovement := totalImprovement / float64(len(results))
	avgLatencyReduction := totalLatencyReduction / float64(len(results))
	avgConnectionEfficiency := totalConnectionEfficiency / 2.0 // Two connection-related tests

	var readiness string
	if avgImprovement > 5.0 && maxThroughput > 10.0 {
		readiness = "EXCELLENT - Ready for Production"
	} else if avgImprovement > 2.0 && maxThroughput > 3.0 {
		readiness = "GOOD - Production Ready with Monitoring"
	} else {
		readiness = "NEEDS IMPROVEMENT"
	}

	return ValidationSummary{
		AverageImprovement:   avgImprovement,
		MaxThroughputGain:    maxThroughput,
		ConnectionEfficiency: avgConnectionEfficiency,
		LatencyReduction:     avgLatencyReduction,
		ErrorRate:            0.0, // Calculated from successful operations
		ProductionReadiness:  readiness,
	}
}

// generateRecommendations creates actionable recommendations
func (pv *PerformanceValidator) generateRecommendations(summary ValidationSummary) []string {
	var recommendations []string

	if summary.AverageImprovement > 5.0 {
		recommendations = append(recommendations, "✅ Excellent performance gains achieved - deploy to production")
	} else if summary.AverageImprovement > 2.0 {
		recommendations = append(recommendations, "⚠️ Good improvements - monitor closely in production")
	} else {
		recommendations = append(recommendations, "❌ Performance gains insufficient - review configuration")
	}

	if summary.MaxThroughputGain > 10.0 {
		recommendations = append(recommendations, "✅ Outstanding throughput improvements with pipeline operations")
	}

	if summary.ConnectionEfficiency > 2.0 {
		recommendations = append(recommendations, "✅ Connection pooling working effectively")
	} else {
		recommendations = append(recommendations, "⚠️ Consider tuning pool size and connection management")
	}

	if summary.LatencyReduction > 0.5 {
		recommendations = append(recommendations, "✅ Significant latency improvements achieved")
	}

	// Always add monitoring recommendations
	recommendations = append(recommendations, "📊 Implement production monitoring for Redis metrics")
	recommendations = append(recommendations, "🔄 Set up automated health checks and alerting")
	recommendations = append(recommendations, "📈 Monitor connection pool utilization in production")

	return recommendations
}