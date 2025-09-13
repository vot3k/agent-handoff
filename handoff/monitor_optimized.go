package handoff

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

// OptimizedHandoffMonitor provides monitoring and metrics for handoff system with optimized Redis operations
type OptimizedHandoffMonitor struct {
	redisManager *RedisManager
	metrics      *HandoffMetrics
	metricsMutex sync.RWMutex
	alertRules   []AlertRule
	subscribers  map[string][]chan AlertEvent
	subMutex     sync.RWMutex
}

// NewOptimizedHandoffMonitor creates a new optimized handoff monitor
func NewOptimizedHandoffMonitor(redisManager *RedisManager) *OptimizedHandoffMonitor {
	return &OptimizedHandoffMonitor{
		redisManager: redisManager,
		metrics:      &HandoffMetrics{LastUpdated: time.Now()},
		alertRules:   make([]AlertRule, 0),
		subscribers:  make(map[string][]chan AlertEvent),
	}
}

// AddAlertRule adds a new alert rule
func (m *OptimizedHandoffMonitor) AddAlertRule(rule AlertRule) {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	
	m.alertRules = append(m.alertRules, rule)
	
	log.Info().
		Str("rule_name", rule.Name).
		Str("type", string(rule.Type)).
		Float64("threshold", rule.Threshold).
		Msg("Alert rule added")
}

// SubscribeToAlerts subscribes to alert events of a specific type
func (m *OptimizedHandoffMonitor) SubscribeToAlerts(alertType AlertType) chan AlertEvent {
	m.subMutex.Lock()
	defer m.subMutex.Unlock()
	
	ch := make(chan AlertEvent, 100) // Buffered to prevent blocking
	typeStr := string(alertType)
	
	if m.subscribers[typeStr] == nil {
		m.subscribers[typeStr] = make([]chan AlertEvent, 0)
	}
	
	m.subscribers[typeStr] = append(m.subscribers[typeStr], ch)
	return ch
}

// StartMonitoring starts the monitoring loop with optimized Redis operations
func (m *OptimizedHandoffMonitor) StartMonitoring(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	log.Info().
		Dur("interval", interval).
		Msg("Starting optimized handoff monitoring")
	
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Optimized handoff monitoring stopped")
			return
		case <-ticker.C:
			if err := m.collectOptimizedMetrics(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to collect metrics")
			}
			
			m.evaluateAlerts()
		}
	}
}

// collectOptimizedMetrics collects system metrics from Redis using optimized operations
func (m *OptimizedHandoffMonitor) collectOptimizedMetrics(ctx context.Context) error {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	
	// Use optimized key operations to get queue depths
	keyOps := m.redisManager.GetKeyOps()
	keys, err := keyOps.ScanPattern(ctx, "handoff:queue:*")
	if err != nil {
		return fmt.Errorf("failed to scan queue keys: %w", err)
	}
	
	// Use batch operations to get queue depths
	var totalQueueDepth int64
	client := m.redisManager.GetClient()
	
	if len(keys) > 0 {
		// Create pipeline for batch operations
		pipe := client.Pipeline()
		cardCmds := make(map[string]*redis.IntCmd)
		
		for _, key := range keys {
			cardCmds[key] = pipe.ZCard(ctx, key)
		}
		
		_, err := pipe.Exec(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to execute queue depth pipeline")
		} else {
			for key, cmd := range cardCmds {
				depth, err := cmd.Result()
				if err != nil {
					log.Error().Err(err).Str("queue", key).Msg("Failed to get queue depth")
					continue
				}
				totalQueueDepth += depth
			}
		}
	}
	
	m.metrics.QueueDepth = totalQueueDepth
	
	// Get handoff counts from Redis counters using optimized operations
	var totalHandoffs, completedHandoffs, failedHandoffs int64
	
	// Use pipeline for batch metric retrieval
	pipe := client.Pipeline()
	totalCmd := pipe.Get(ctx, "handoff:metrics:total")
	completedCmd := pipe.Get(ctx, "handoff:metrics:completed")
	failedCmd := pipe.Get(ctx, "handoff:metrics:failed")
	activeAgentsCmd := pipe.SMembers(ctx, "handoff:active_agents")
	processingTimesCmd := pipe.LRange(ctx, "handoff:processing_times", 0, 99)
	
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		log.Error().Err(err).Msg("Failed to execute metrics pipeline")
	}
	
	// Process results
	if val, err := totalCmd.Int64(); err == nil {
		totalHandoffs = val
	}
	if val, err := completedCmd.Int64(); err == nil {
		completedHandoffs = val
	}
	if val, err := failedCmd.Int64(); err == nil {
		failedHandoffs = val
	}
	
	m.metrics.TotalHandoffs = totalHandoffs
	m.metrics.CompletedHandoffs = completedHandoffs
	m.metrics.FailedHandoffs = failedHandoffs
	
	// Get active agents
	if activeAgents, err := activeAgentsCmd.Result(); err == nil {
		m.metrics.ActiveAgents = activeAgents
	} else {
		m.metrics.ActiveAgents = []string{}
	}
	
	// Calculate average processing time from recent handoffs
	if processingTimes, err := processingTimesCmd.Result(); err == nil && len(processingTimes) > 0 {
		var totalTime time.Duration
		validTimes := 0
		
		for _, timeStr := range processingTimes {
			if duration, err := time.ParseDuration(timeStr); err == nil {
				totalTime += duration
				validTimes++
			}
		}
		
		if validTimes > 0 {
			m.metrics.AvgProcessingTime = totalTime / time.Duration(validTimes)
		}
	}
	
	m.metrics.LastUpdated = time.Now()
	
	// Store metrics snapshot in Redis for persistence using optimized operations
	if err := m.redisManager.SetWithOptimizedExpiry(ctx, "handoff:metrics:snapshot", m.metrics, time.Hour); err != nil {
		log.Error().Err(err).Msg("Failed to store metrics snapshot")
	}
	
	return nil
}

// evaluateAlerts evaluates all alert rules
func (m *OptimizedHandoffMonitor) evaluateAlerts() {
	m.metricsMutex.RLock()
	defer m.metricsMutex.RUnlock()
	
	for i, rule := range m.alertRules {
		if !rule.Enabled {
			continue
		}
		
		// Check cooldown
		if time.Since(rule.LastFired) < rule.Cooldown {
			continue
		}
		
		value, triggered := m.evaluateAlertRule(rule)
		if triggered {
			alert := AlertEvent{
				Rule:      rule,
				Value:     value,
				Timestamp: time.Now(),
				Message:   m.generateAlertMessage(rule, value),
				Severity:  m.calculateSeverity(rule, value),
			}
			
			// Update last fired time
			m.alertRules[i].LastFired = time.Now()
			
			// Send alert to subscribers
			m.sendAlert(rule.Type, alert)
			
			log.Warn().
				Str("rule", rule.Name).
				Float64("value", value).
				Float64("threshold", rule.Threshold).
				Str("severity", string(alert.Severity)).
				Msg("Alert triggered")
		}
	}
}

// evaluateAlertRule evaluates a single alert rule
func (m *OptimizedHandoffMonitor) evaluateAlertRule(rule AlertRule) (float64, bool) {
	var value float64
	
	switch rule.Type {
	case AlertQueueDepth:
		value = float64(m.metrics.QueueDepth)
	case AlertProcessingTime:
		value = float64(m.metrics.AvgProcessingTime / time.Millisecond)
	case AlertFailureRate:
		if m.metrics.TotalHandoffs > 0 {
			value = float64(m.metrics.FailedHandoffs) / float64(m.metrics.TotalHandoffs) * 100
		}
	case AlertAgentHealth:
		value = float64(len(m.metrics.ActiveAgents))
	case AlertSystemHealth:
		// System health score based on multiple factors including Redis health
		value = m.calculateSystemHealthScore()
	default:
		return 0, false
	}
	
	return value, m.checkThreshold(rule, value)
}

// calculateSystemHealthScore calculates an overall system health score including Redis metrics
func (m *OptimizedHandoffMonitor) calculateSystemHealthScore() float64 {
	score := 100.0
	
	// Deduct points for high queue depth
	if m.metrics.QueueDepth > 50 {
		score -= float64(m.metrics.QueueDepth-50) * 0.5
	}
	
	// Deduct points for high failure rate
	if m.metrics.TotalHandoffs > 0 {
		failureRate := float64(m.metrics.FailedHandoffs) / float64(m.metrics.TotalHandoffs) * 100
		if failureRate > 5 {
			score -= (failureRate - 5) * 2
		}
	}
	
	// Deduct points for slow processing
	avgProcessingMs := float64(m.metrics.AvgProcessingTime / time.Millisecond)
	if avgProcessingMs > 5000 { // 5 seconds
		score -= (avgProcessingMs - 5000) * 0.01
	}
	
	// Deduct points for inactive agents
	if len(m.metrics.ActiveAgents) == 0 {
		score -= 50
	}
	
	// Factor in Redis health
	if !m.redisManager.IsHealthy() {
		score -= 30 // Major deduction for unhealthy Redis
	}
	
	// Factor in Redis connection pool metrics
	redisMetrics := m.redisManager.GetDetailedMetrics()
	
	// Deduct points for high connection pool usage
	if redisMetrics.TotalConns > 0 {
		poolUsage := float64(redisMetrics.TotalConns-redisMetrics.IdleConns) / float64(redisMetrics.TotalConns) * 100
		if poolUsage > 80 {
			score -= (poolUsage - 80) * 0.5
		}
	}
	
	// Deduct points for high failure rate in Redis operations
	if redisMetrics.TotalRequests > 0 {
		redisFailureRate := float64(redisMetrics.FailedRequests) / float64(redisMetrics.TotalRequests) * 100
		if redisFailureRate > 1 {
			score -= (redisFailureRate - 1) * 5
		}
	}
	
	// Deduct points for high Redis latency
	avgLatencyMs := float64(redisMetrics.AvgLatency / time.Millisecond)
	if avgLatencyMs > 50 {
		score -= (avgLatencyMs - 50) * 0.1
	}
	
	if score < 0 {
		score = 0
	}
	
	return score
}

// checkThreshold checks if the value crosses the threshold
func (m *OptimizedHandoffMonitor) checkThreshold(rule AlertRule, value float64) bool {
	switch rule.Condition {
	case "greater_than", ">":
		return value > rule.Threshold
	case "less_than", "<":
		return value < rule.Threshold
	case "equals", "=":
		return value == rule.Threshold
	case "greater_equal", ">=":
		return value >= rule.Threshold
	case "less_equal", "<=":
		return value <= rule.Threshold
	default:
		return value > rule.Threshold // Default behavior
	}
}

// generateAlertMessage generates a human-readable alert message
func (m *OptimizedHandoffMonitor) generateAlertMessage(rule AlertRule, value float64) string {
	switch rule.Type {
	case AlertQueueDepth:
		return fmt.Sprintf("Queue depth is %.0f (threshold: %.0f)", value, rule.Threshold)
	case AlertProcessingTime:
		return fmt.Sprintf("Average processing time is %.0fms (threshold: %.0fms)", value, rule.Threshold)
	case AlertFailureRate:
		return fmt.Sprintf("Failure rate is %.1f%% (threshold: %.1f%%)", value, rule.Threshold)
	case AlertAgentHealth:
		return fmt.Sprintf("Active agents: %.0f (threshold: %.0f)", value, rule.Threshold)
	case AlertSystemHealth:
		return fmt.Sprintf("System health score is %.1f (threshold: %.1f)", value, rule.Threshold)
	default:
		return fmt.Sprintf("Alert %s triggered: %.2f", rule.Name, value)
	}
}

// calculateSeverity determines alert severity based on how far the value exceeds the threshold
func (m *OptimizedHandoffMonitor) calculateSeverity(rule AlertRule, value float64) Severity {
	ratio := value / rule.Threshold
	
	switch rule.Type {
	case AlertQueueDepth:
		if ratio >= 3.0 {
			return SeverityCritical
		} else if ratio >= 2.0 {
			return SeverityError
		} else if ratio >= 1.5 {
			return SeverityWarning
		}
		return SeverityInfo
	case AlertFailureRate:
		if value >= 50 {
			return SeverityCritical
		} else if value >= 25 {
			return SeverityError
		} else if value >= 10 {
			return SeverityWarning
		}
		return SeverityInfo
	case AlertSystemHealth:
		if value <= 25 {
			return SeverityCritical
		} else if value <= 50 {
			return SeverityError
		} else if value <= 75 {
			return SeverityWarning
		}
		return SeverityInfo
	default:
		if ratio >= 2.0 {
			return SeverityError
		} else if ratio >= 1.5 {
			return SeverityWarning
		}
		return SeverityInfo
	}
}

// sendAlert sends an alert to all subscribers
func (m *OptimizedHandoffMonitor) sendAlert(alertType AlertType, alert AlertEvent) {
	m.subMutex.RLock()
	defer m.subMutex.RUnlock()
	
	typeStr := string(alertType)
	subscribers := m.subscribers[typeStr]
	
	for _, ch := range subscribers {
		select {
		case ch <- alert:
		default:
			// Channel is full, log warning
			log.Warn().
				Str("alert_type", typeStr).
				Msg("Alert channel is full, dropping alert")
		}
	}
	
	// Also send to "all" subscribers
	allSubscribers := m.subscribers["all"]
	for _, ch := range allSubscribers {
		select {
		case ch <- alert:
		default:
			log.Warn().Msg("All alerts channel is full, dropping alert")
		}
	}
}

// GetMetrics returns current metrics
func (m *OptimizedHandoffMonitor) GetMetrics() HandoffMetrics {
	m.metricsMutex.RLock()
	defer m.metricsMutex.RUnlock()
	return *m.metrics
}

// GetDetailedMetrics returns both handoff and Redis metrics
func (m *OptimizedHandoffMonitor) GetDetailedMetrics() (HandoffMetrics, RedisPoolMetrics) {
	m.metricsMutex.RLock()
	defer m.metricsMutex.RUnlock()
	
	handoffMetrics := *m.metrics
	redisMetrics := m.redisManager.GetDetailedMetrics()
	
	return handoffMetrics, redisMetrics
}

// GetAlertRules returns all alert rules
func (m *OptimizedHandoffMonitor) GetAlertRules() []AlertRule {
	m.metricsMutex.RLock()
	defer m.metricsMutex.RUnlock()
	
	rules := make([]AlertRule, len(m.alertRules))
	copy(rules, m.alertRules)
	return rules
}

// UpdateAlertRule updates an existing alert rule
func (m *OptimizedHandoffMonitor) UpdateAlertRule(name string, rule AlertRule) error {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	
	for i, existingRule := range m.alertRules {
		if existingRule.Name == name {
			rule.Name = name // Ensure name doesn't change
			m.alertRules[i] = rule
			log.Info().Str("rule_name", name).Msg("Alert rule updated")
			return nil
		}
	}
	
	return fmt.Errorf("alert rule %s not found", name)
}

// RemoveAlertRule removes an alert rule
func (m *OptimizedHandoffMonitor) RemoveAlertRule(name string) error {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	
	for i, rule := range m.alertRules {
		if rule.Name == name {
			// Remove rule by slicing
			m.alertRules = append(m.alertRules[:i], m.alertRules[i+1:]...)
			log.Info().Str("rule_name", name).Msg("Alert rule removed")
			return nil
		}
	}
	
	return fmt.Errorf("alert rule %s not found", name)
}

// RecordHandoffMetrics records metrics for a completed handoff using optimized operations
func (m *OptimizedHandoffMonitor) RecordHandoffMetrics(ctx context.Context, handoff *Handoff, processingTime time.Duration, success bool) {
	// Use optimized batch operations for recording metrics
	operations := []func(redis.Pipeliner) error{
		func(pipe redis.Pipeliner) error {
			pipe.Incr(ctx, "handoff:metrics:total")
			if success {
				pipe.Incr(ctx, "handoff:metrics:completed")
			} else {
				pipe.Incr(ctx, "handoff:metrics:failed")
			}
			return nil
		},
		func(pipe redis.Pipeliner) error {
			// Record processing time
			pipe.LPush(ctx, "handoff:processing_times", processingTime.String())
			pipe.LTrim(ctx, "handoff:processing_times", 0, 99) // Keep last 100 processing times
			return nil
		},
		func(pipe redis.Pipeliner) error {
			// Set expiry on counters (they'll be recreated if needed)
			pipe.Expire(ctx, "handoff:metrics:total", 24*time.Hour)
			pipe.Expire(ctx, "handoff:metrics:completed", 24*time.Hour)
			pipe.Expire(ctx, "handoff:metrics:failed", 24*time.Hour)
			return nil
		},
	}
	
	if err := m.redisManager.ExecuteBatch(ctx, operations); err != nil {
		log.Error().Err(err).Msg("Failed to record handoff metrics")
	}
}

// SetAgentActive marks an agent as active using optimized operations
func (m *OptimizedHandoffMonitor) SetAgentActive(ctx context.Context, agentName string) {
	operations := []func(redis.Pipeliner) error{
		func(pipe redis.Pipeliner) error {
			pipe.SAdd(ctx, "handoff:active_agents", agentName)
			pipe.Expire(ctx, "handoff:active_agents", 5*time.Minute) // Expire after 5 minutes of inactivity
			return nil
		},
	}
	
	if err := m.redisManager.ExecuteBatch(ctx, operations); err != nil {
		log.Error().Err(err).Str("agent", agentName).Msg("Failed to mark agent as active")
	}
}

// SetAgentInactive removes an agent from the active list
func (m *OptimizedHandoffMonitor) SetAgentInactive(ctx context.Context, agentName string) {
	client := m.redisManager.GetClient()
	if err := client.SRem(ctx, "handoff:active_agents", agentName).Err(); err != nil {
		log.Error().Err(err).Str("agent", agentName).Msg("Failed to mark agent as inactive")
	}
}

// GetQueueStatus returns detailed queue status for all agents using optimized operations
func (m *OptimizedHandoffMonitor) GetQueueStatus(ctx context.Context) (map[string]QueueStatus, error) {
	keyOps := m.redisManager.GetKeyOps()
	keys, err := keyOps.ScanPattern(ctx, "handoff:queue:*")
	if err != nil {
		return nil, fmt.Errorf("failed to scan queue keys: %w", err)
	}
	
	status := make(map[string]QueueStatus)
	client := m.redisManager.GetClient()
	
	if len(keys) == 0 {
		return status, nil
	}
	
	// Use pipeline for batch operations
	pipe := client.Pipeline()
	cardCmds := make(map[string]*redis.IntCmd)
	rangeCmds := make(map[string]*redis.ZSliceCmd)
	
	for _, key := range keys {
		cardCmds[key] = pipe.ZCard(ctx, key)
		rangeCmds[key] = pipe.ZRangeWithScores(ctx, key, 0, 0)
	}
	
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to execute queue status pipeline: %w", err)
	}
	
	for _, key := range keys {
		agentName := strings.TrimPrefix(key, "handoff:queue:")
		
		// Get queue depth
		var depth int64
		if cmd, exists := cardCmds[key]; exists {
			if val, err := cmd.Result(); err == nil {
				depth = val
			}
		}
		
		// Get oldest item timestamp
		var oldestTimestamp time.Time
		if depth > 0 {
			if cmd, exists := rangeCmds[key]; exists {
				if items, err := cmd.Result(); err == nil && len(items) > 0 {
					// Score contains timestamp as fractional part
					score := items[0].Score
					oldestTimestamp = time.Unix(int64(score), int64((score-float64(int64(score)))*1e9))
				}
			}
		}
		
		status[agentName] = QueueStatus{
			AgentName:  agentName,
			QueueDepth: int(depth),
			OldestItem: oldestTimestamp,
			QueueName:  key,
		}
	}
	
	return status, nil
}

// GetRedisHealth returns the Redis connection health status
func (m *OptimizedHandoffMonitor) GetRedisHealth() HealthStatus {
	return m.redisManager.GetHealth()
}

// IsRedisHealthy returns true if Redis connection is healthy
func (m *OptimizedHandoffMonitor) IsRedisHealthy() bool {
	return m.redisManager.IsHealthy()
}