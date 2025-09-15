package handoff

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

var (
	globalRedisManager *RedisManager
	managerOnce        sync.Once
)

// RedisManager provides a centralized Redis connection manager for the entire application
type RedisManager struct {
	poolManager *RedisPoolManager
	config      RedisPoolConfig
	mu          sync.RWMutex
}

// GetRedisManager returns the singleton Redis manager instance
func GetRedisManager() *RedisManager {
	managerOnce.Do(func() {
		config := DefaultRedisPoolConfig()
		manager, err := NewRedisManager(config)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize Redis manager")
		}
		globalRedisManager = manager
	})
	return globalRedisManager
}

// InitializeRedisManager initializes the global Redis manager with custom config
func InitializeRedisManager(config RedisPoolConfig) error {
	var err error
	managerOnce.Do(func() {
		globalRedisManager, err = NewRedisManager(config)
	})
	return err
}

// NewRedisManager creates a new Redis manager with optimized connection pooling
func NewRedisManager(config RedisPoolConfig) (*RedisManager, error) {
	poolManager, err := NewRedisPoolManager(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis pool manager: %w", err)
	}

	manager := &RedisManager{
		poolManager: poolManager,
		config:      config,
	}

	log.Info().Msg("Redis manager initialized with optimized connection pooling")
	return manager, nil
}

// GetClient returns the shared Redis client
func (r *RedisManager) GetClient() *redis.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.poolManager.GetClient()
}

// GetPoolManager returns the pool manager for advanced operations
func (r *RedisManager) GetPoolManager() *RedisPoolManager {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.poolManager
}

// ExecuteBatch executes multiple Redis operations in a pipeline for better performance
func (r *RedisManager) ExecuteBatch(ctx context.Context, operations []func(redis.Pipeliner) error) error {
	pipeline := r.poolManager.Pipeline()
	
	// Add all operations to pipeline
	for _, op := range operations {
		if err := op(pipeline); err != nil {
			return fmt.Errorf("failed to add operation to pipeline: %w", err)
		}
	}

	// Execute pipeline
	start := time.Now()
	_, err := pipeline.Exec(ctx)
	latency := time.Since(start)

	// Update metrics
	r.poolManager.recordMetrics(latency, err)
	if err == nil {
		r.poolManager.metricsMutex.Lock()
		r.poolManager.metrics.PipelineHits++
		r.poolManager.metrics.BatchOperations += uint64(len(operations))
		r.poolManager.metricsMutex.Unlock()
	}

	if err != nil {
		return fmt.Errorf("pipeline execution failed: %w", err)
	}

	log.Debug().
		Int("operations", len(operations)).
		Dur("latency", latency).
		Msg("Batch Redis operations completed")

	return nil
}

// SetWithOptimizedExpiry sets a key with optimized expiry handling
func (r *RedisManager) SetWithOptimizedExpiry(ctx context.Context, key string, value interface{}, expiry time.Duration) error {
	return r.poolManager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
		// Serialize value if needed
		var data []byte
		var err error
		
		switch v := value.(type) {
		case string:
			data = []byte(v)
		case []byte:
			data = v
		default:
			data, err = json.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to marshal value: %w", err)
			}
		}

		return client.Set(ctx, key, data, expiry).Err()
	})
}

// GetWithDeserialization gets a key and deserializes it into the target
func (r *RedisManager) GetWithDeserialization(ctx context.Context, key string, target interface{}) error {
	return r.poolManager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
		data, err := client.Get(ctx, key).Result()
		if err != nil {
			return err
		}

		if target == nil {
			return nil
		}

		// Handle different target types
		switch t := target.(type) {
		case *string:
			*t = data
			return nil
		case *[]byte:
			*t = []byte(data)
			return nil
		default:
			return json.Unmarshal([]byte(data), target)
		}
	})
}

// IncrementCounter increments a counter with automatic expiry
func (r *RedisManager) IncrementCounter(ctx context.Context, key string, expiry time.Duration) (int64, error) {
	var result int64
	err := r.poolManager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
		pipe := client.Pipeline()
		incrCmd := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, expiry)
		
		_, err := pipe.Exec(ctx)
		if err != nil {
			return err
		}
		
		result = incrCmd.Val()
		return nil
	})
	
	return result, err
}

// QueueOperations provides optimized queue operations
type QueueOperations struct {
	manager *RedisManager
}

// GetQueueOps returns queue operations helper
func (r *RedisManager) GetQueueOps() *QueueOperations {
	return &QueueOperations{manager: r}
}

// ZAddBatch adds multiple items to a sorted set in a single operation
func (q *QueueOperations) ZAddBatch(ctx context.Context, key string, members []*redis.Z) error {
	if len(members) == 0 {
		return nil
	}

	return q.manager.poolManager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
		// Split into batches of 100 to avoid overwhelming Redis
		const batchSize = 100
		
		for i := 0; i < len(members); i += batchSize {
			end := i + batchSize
			if end > len(members) {
				end = len(members)
			}
			
			batch := members[i:end]
			if err := client.ZAdd(ctx, key, batch...).Err(); err != nil {
				return fmt.Errorf("failed to add batch %d-%d: %w", i, end-1, err)
			}
		}
		
		return nil
	})
}

// ZPopMinBatch pops multiple items from sorted sets
func (q *QueueOperations) ZPopMinBatch(ctx context.Context, keys []string, count int64) (map[string][]redis.Z, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	results := make(map[string][]redis.Z)
	
	err := q.manager.poolManager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
		pipe := client.Pipeline()
		cmds := make(map[string]*redis.ZSliceCmd)
		
		// Add all ZPopMin operations to pipeline
		for _, key := range keys {
			cmds[key] = pipe.ZPopMin(ctx, key, count)
		}
		
		_, err := pipe.Exec(ctx)
		if err != nil && err != redis.Nil {
			return err
		}
		
		// Collect results
		for key, cmd := range cmds {
			if result, err := cmd.Result(); err == nil && len(result) > 0 {
				results[key] = result
			}
		}
		
		return nil
	})
	
	return results, err
}

// KeyOperations provides optimized key operations
type KeyOperations struct {
	manager *RedisManager
}

// GetKeyOps returns key operations helper
func (r *RedisManager) GetKeyOps() *KeyOperations {
	return &KeyOperations{manager: r}
}

// ScanPattern scans keys matching a pattern with optimal batch size
func (k *KeyOperations) ScanPattern(ctx context.Context, pattern string) ([]string, error) {
	var allKeys []string
	
	err := k.manager.poolManager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
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

// DeleteBatch deletes multiple keys in batches
func (k *KeyOperations) DeleteBatch(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	return k.manager.poolManager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
		// Split into batches to avoid overwhelming Redis
		const batchSize = 100
		
		for i := 0; i < len(keys); i += batchSize {
			end := i + batchSize
			if end > len(keys) {
				end = len(keys)
			}
			
			batch := keys[i:end]
			if err := client.Del(ctx, batch...).Err(); err != nil {
				return fmt.Errorf("failed to delete batch %d-%d: %w", i, end-1, err)
			}
		}
		
		return nil
	})
}

// GetHealth returns the health status of Redis connections
func (r *RedisManager) GetHealth() HealthStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.poolManager.GetHealthStatus()
}

// GetDetailedMetrics returns comprehensive Redis metrics
func (r *RedisManager) GetDetailedMetrics() RedisPoolMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.poolManager.GetMetrics()
}

// IsHealthy returns true if Redis is healthy
func (r *RedisManager) IsHealthy() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.poolManager.IsHealthy()
}

// Shutdown gracefully closes all Redis connections
func (r *RedisManager) Shutdown() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.poolManager != nil {
		return r.poolManager.Close()
	}
	
	return nil
}

// GetMemoryOptimizedClient returns a client configured for memory efficiency
func (r *RedisManager) GetMemoryOptimizedClient(ctx context.Context) *redis.Client {
	// For memory optimization, we return the shared client
	// All optimizations are handled at the pool level
	return r.GetClient()
}

// CleanupExpiredKeys performs cleanup of expired keys to optimize memory
func (r *RedisManager) CleanupExpiredKeys(ctx context.Context, patterns []string) error {
	return r.poolManager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
		for _, pattern := range patterns {
			// Scan for keys matching pattern
			var cursor uint64
			var keys []string
			var err error
			
			for {
				keys, cursor, err = client.Scan(ctx, cursor, pattern, 100).Result()
				if err != nil {
					return fmt.Errorf("failed to scan pattern %s: %w", pattern, err)
				}
				
				// Check TTL for each key and clean up if needed
				for _, key := range keys {
					ttl, err := client.TTL(ctx, key).Result()
					if err != nil {
						continue
					}
					
					// Remove keys that should have expired (TTL = -1 means no expiry, -2 means expired)
					if ttl == -2*time.Second {
						client.Del(ctx, key)
						log.Debug().Str("key", key).Msg("Cleaned up expired key")
					}
				}
				
				if cursor == 0 {
					break
				}
			}
		}
		
		return nil
	})
}

// SetMemoryOptimizations configures Redis for memory efficiency
func (r *RedisManager) SetMemoryOptimizations(ctx context.Context) error {
	return r.poolManager.ExecuteWithRetry(ctx, func(client *redis.Client) error {
		// Configure Redis for memory optimization
		optimizations := map[string]string{
			"maxmemory-policy": "allkeys-lru",  // Use LRU eviction
			"save": "",                         // Disable background saves for memory
			"rdbcompression": "yes",            // Enable RDB compression
			"list-max-ziplist-entries": "512",  // Optimize list storage
			"hash-max-ziplist-entries": "512",  // Optimize hash storage
			"set-max-intset-entries": "512",    // Optimize set storage
			"zset-max-ziplist-entries": "128",  // Optimize sorted set storage
		}
		
		for key, value := range optimizations {
			err := client.ConfigSet(ctx, key, value).Err()
			if err != nil {
				log.Warn().
					Err(err).
					Str("config", key).
					Str("value", value).
					Msg("Failed to set Redis memory optimization")
			}
		}
		
		return nil
	})
}