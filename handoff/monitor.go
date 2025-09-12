package handoff

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

// HandoffMonitor provides monitoring and metrics for handoff system
type HandoffMonitor struct {
	redis       *redis.Client
	metrics     *HandoffMetrics
	metricsMutex sync.RWMutex
	alertRules  []AlertRule
	subscribers map[string][]chan AlertEvent
	subMutex    sync.RWMutex
}

// AlertRule defines conditions that trigger alerts
type AlertRule struct {
	Name        string        `json:"name"`
	Type        AlertType     `json:"type"`
	Condition   string        `json:"condition"`   // e.g., "queue_depth > 100"
	Threshold   float64       `json:"threshold"`
	Duration    time.Duration `json:"duration"`    // How long condition must persist
	Enabled     bool          `json:"enabled"`
	LastFired   time.Time     `json:"last_fired"`
	Cooldown    time.Duration `json:"cooldown"`    // Minimum time between alerts
}

// AlertType defines the type of alert
type AlertType string

const (
	AlertQueueDepth      AlertType = "queue_depth"
	AlertProcessingTime  AlertType = "processing_time"
	AlertFailureRate     AlertType = "failure_rate"
	AlertAgentHealth     AlertType = "agent_health"
	AlertSystemHealth    AlertType = "system_health"
)

// AlertEvent represents an alert that was triggered
type AlertEvent struct {
	Rule        AlertRule `json:"rule"`
	Value       float64   `json:"value"`
	Timestamp   time.Time `json:"timestamp"`
	Message     string    `json:"message"`
	Severity    Severity  `json:"severity"`
}

// Severity defines alert severity levels
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// NewHandoffMonitor creates a new handoff monitor
func NewHandoffMonitor(redisClient *redis.Client) *HandoffMonitor {
	return &HandoffMonitor{
		redis:       redisClient,
		metrics:     &HandoffMetrics{LastUpdated: time.Now()},
		alertRules:  make([]AlertRule, 0),
		subscribers: make(map[string][]chan AlertEvent),
	}
}

// AddAlertRule adds a new alert rule
func (m *HandoffMonitor) AddAlertRule(rule AlertRule) {
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
func (m *HandoffMonitor) SubscribeToAlerts(alertType AlertType) chan AlertEvent {
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

// StartMonitoring starts the monitoring loop
func (m *HandoffMonitor) StartMonitoring(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	log.Info().
		Dur("interval", interval).
		Msg("Starting handoff monitoring")
	
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Handoff monitoring stopped")
			return
		case <-ticker.C:
			if err := m.collectMetrics(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to collect metrics")
			}
			
			m.evaluateAlerts()
		}
	}
}

// collectMetrics collects system metrics from Redis
func (m *HandoffMonitor) collectMetrics(ctx context.Context) error {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	
	// Get queue depths
	var totalQueueDepth int64
	keys, err := m.redis.Keys(ctx, "handoff:queue:*").Result()
	if err != nil {
		return fmt.Errorf("failed to get queue keys: %w", err)
	}
	
	for _, key := range keys {
		depth, err := m.redis.ZCard(ctx, key).Result()
		if err != nil {
			log.Error().Err(err).Str("queue", key).Msg("Failed to get queue depth")
			continue
		}
		totalQueueDepth += depth
	}
	
	m.metrics.QueueDepth = totalQueueDepth
	
	// Get handoff counts from Redis counters
	totalHandoffs, _ := m.redis.Get(ctx, "handoff:metrics:total").Int64()
	completedHandoffs, _ := m.redis.Get(ctx, "handoff:metrics:completed").Int64()
	failedHandoffs, _ := m.redis.Get(ctx, "handoff:metrics:failed").Int64()
	
	m.metrics.TotalHandoffs = totalHandoffs
	m.metrics.CompletedHandoffs = completedHandoffs
	m.metrics.FailedHandoffs = failedHandoffs
	
	// Get active agents
	activeAgents, err := m.redis.SMembers(ctx, "handoff:active_agents").Result()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get active agents")
		activeAgents = []string{}
	}
	m.metrics.ActiveAgents = activeAgents
	
	// Calculate average processing time from recent handoffs
	processingTimes, err := m.redis.LRange(ctx, "handoff:processing_times", 0, 99).Result()
	if err == nil && len(processingTimes) > 0 {
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
	
	// Store metrics in Redis for persistence
	metricsJSON, _ := json.Marshal(m.metrics)
	m.redis.Set(ctx, "handoff:metrics:snapshot", metricsJSON, time.Hour)
	
	return nil
}

// evaluateAlerts evaluates all alert rules
func (m *HandoffMonitor) evaluateAlerts() {
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
func (m *HandoffMonitor) evaluateAlertRule(rule AlertRule) (float64, bool) {
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
		// System health score based on multiple factors
		value = m.calculateSystemHealthScore()
	default:
		return 0, false
	}
	
	return value, m.checkThreshold(rule, value)
}

// checkThreshold checks if the value crosses the threshold
func (m *HandoffMonitor) checkThreshold(rule AlertRule, value float64) bool {
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

// calculateSystemHealthScore calculates an overall system health score
func (m *HandoffMonitor) calculateSystemHealthScore() float64 {
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
	
	if score < 0 {
		score = 0
	}
	
	return score
}

// generateAlertMessage generates a human-readable alert message
func (m *HandoffMonitor) generateAlertMessage(rule AlertRule, value float64) string {
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
func (m *HandoffMonitor) calculateSeverity(rule AlertRule, value float64) Severity {
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
func (m *HandoffMonitor) sendAlert(alertType AlertType, alert AlertEvent) {
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
func (m *HandoffMonitor) GetMetrics() HandoffMetrics {
	m.metricsMutex.RLock()
	defer m.metricsMutex.RUnlock()
	return *m.metrics
}

// GetAlertRules returns all alert rules
func (m *HandoffMonitor) GetAlertRules() []AlertRule {
	m.metricsMutex.RLock()
	defer m.metricsMutex.RUnlock()
	
	rules := make([]AlertRule, len(m.alertRules))
	copy(rules, m.alertRules)
	return rules
}

// UpdateAlertRule updates an existing alert rule
func (m *HandoffMonitor) UpdateAlertRule(name string, rule AlertRule) error {
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
func (m *HandoffMonitor) RemoveAlertRule(name string) error {
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

// RecordHandoffMetrics records metrics for a completed handoff
func (m *HandoffMonitor) RecordHandoffMetrics(ctx context.Context, handoff *Handoff, processingTime time.Duration, success bool) {
	// Increment counters
	pipe := m.redis.Pipeline()
	pipe.Incr(ctx, "handoff:metrics:total")
	
	if success {
		pipe.Incr(ctx, "handoff:metrics:completed")
	} else {
		pipe.Incr(ctx, "handoff:metrics:failed")
	}
	
	// Record processing time
	pipe.LPush(ctx, "handoff:processing_times", processingTime.String())
	pipe.LTrim(ctx, "handoff:processing_times", 0, 99) // Keep last 100 processing times
	
	// Set expiry on counters (they'll be recreated if needed)
	pipe.Expire(ctx, "handoff:metrics:total", 24*time.Hour)
	pipe.Expire(ctx, "handoff:metrics:completed", 24*time.Hour)
	pipe.Expire(ctx, "handoff:metrics:failed", 24*time.Hour)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to record handoff metrics")
	}
}

// SetAgentActive marks an agent as active
func (m *HandoffMonitor) SetAgentActive(ctx context.Context, agentName string) {
	if err := m.redis.SAdd(ctx, "handoff:active_agents", agentName).Err(); err != nil {
		log.Error().Err(err).Str("agent", agentName).Msg("Failed to mark agent as active")
	}
	m.redis.Expire(ctx, "handoff:active_agents", 5*time.Minute) // Expire after 5 minutes of inactivity
}

// SetAgentInactive removes an agent from the active list
func (m *HandoffMonitor) SetAgentInactive(ctx context.Context, agentName string) {
	if err := m.redis.SRem(ctx, "handoff:active_agents", agentName).Err(); err != nil {
		log.Error().Err(err).Str("agent", agentName).Msg("Failed to mark agent as inactive")
	}
}

// GetQueueStatus returns detailed queue status for all agents
func (m *HandoffMonitor) GetQueueStatus(ctx context.Context) (map[string]QueueStatus, error) {
	keys, err := m.redis.Keys(ctx, "handoff:queue:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get queue keys: %w", err)
	}
	
	status := make(map[string]QueueStatus)
	
	for _, key := range keys {
		agentName := strings.TrimPrefix(key, "handoff:queue:")
		
		// Get queue depth
		depth, err := m.redis.ZCard(ctx, key).Result()
		if err != nil {
			log.Error().Err(err).Str("queue", key).Msg("Failed to get queue depth")
			continue
		}
		
		// Get oldest item timestamp
		var oldestTimestamp time.Time
		if depth > 0 {
			items, err := m.redis.ZRangeWithScores(ctx, key, 0, 0).Result()
			if err == nil && len(items) > 0 {
				// Score contains timestamp as fractional part
				score := items[0].Score
				oldestTimestamp = time.Unix(int64(score), int64((score-float64(int64(score)))*1e9))
			}
		}
		
		status[agentName] = QueueStatus{
			AgentName:       agentName,
			QueueDepth:      int(depth),
			OldestItem:      oldestTimestamp,
			QueueName:       key,
		}
	}
	
	return status, nil
}

// QueueStatus contains status information for a queue
type QueueStatus struct {
	AgentName   string    `json:"agent_name"`
	QueueDepth  int       `json:"queue_depth"`
	OldestItem  time.Time `json:"oldest_item"`
	QueueName   string    `json:"queue_name"`
}