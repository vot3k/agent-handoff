package handoff

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// init sets up structured logging for examples
func init() {
	zerolog.TimeFieldFormat = time.RFC3339
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// ExampleHandoffRouter demonstrates intelligent routing
func ExampleHandoffRouter() {
	router := NewHandoffRouter("default-agent")

	// Add routing rules
	implementationRule := RouteRule{
		Name:        "route-to-golang",
		TargetAgent: "golang-expert",
		Priority:    100,
		Conditions: []RouteCondition{
			{
				Type:     ConditionComplexQuery,
				Field:    "has_go_files",
				Operator: "equals",
				Value:    true,
			},
			{
				Type:          ConditionContent,
				Field:         "summary",
				Operator:      "contains",
				Value:         "implement",
				CaseSensitive: false,
			},
		},
	}

	router.AddRoute("api-expert", implementationRule)

	// Create handoff with Go files
	handoff := &Handoff{
		Metadata: Metadata{
			FromAgent: "api-expert",
			ToAgent:   "", // Will be determined by router
		},
		Content: Content{
			Summary: "Implement user service in Go",
			Artifacts: Artifacts{
				Created: []string{"user.go", "user_test.go"},
			},
		},
	}

	// Route handoff
	targetAgent, err := router.RouteHandoff(context.Background(), handoff)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Routed to agent: %s\n", targetAgent)
	// Output: Routed to agent: golang-expert
}

// Example_monitoringAlerts demonstrates monitoring setup
func Example_monitoringAlerts() {
	// Add alert rules
	queueAlert := AlertRule{
		Name:      "high-queue-depth",
		Type:      AlertQueueDepth,
		Condition: "greater_than",
		Threshold: 50,
		Duration:  time.Minute,
		Enabled:   true,
		Cooldown:  5 * time.Minute,
	}

	failureAlert := AlertRule{
		Name:      "high-failure-rate",
		Type:      AlertFailureRate,
		Condition: "greater_than",
		Threshold: 10.0, // 10%
		Duration:  5 * time.Minute,
		Enabled:   true,
		Cooldown:  10 * time.Minute,
	}

	fmt.Printf("Queue alert: %s (threshold: %.0f)\n", queueAlert.Name, queueAlert.Threshold)
	fmt.Printf("Failure alert: %s (threshold: %.1f%%)\n", failureAlert.Name, failureAlert.Threshold)
	// Output: Queue alert: high-queue-depth (threshold: 50)
	// Failure alert: high-failure-rate (threshold: 10.0%)
}

// Example_agentCapabilities shows how to integrate with existing agent system
func Example_agentCapabilities() {
	// Configuration that matches existing patterns
	agents := []AgentCapabilities{
		{Name: "api-expert", QueueName: "handoff:queue:api-expert", MaxConcurrent: 5},
		{Name: "golang-expert", QueueName: "handoff:queue:golang-expert", MaxConcurrent: 3},
		{Name: "typescript-expert", QueueName: "handoff:queue:typescript-expert", MaxConcurrent: 3},
		{Name: "test-expert", QueueName: "handoff:queue:test-expert", MaxConcurrent: 2},
		{Name: "devops-expert", QueueName: "handoff:queue:devops-expert", MaxConcurrent: 2},
	}

	fmt.Printf("Handoff system initialized with %d agents\n", len(agents))
	// Output: Handoff system initialized with 5 agents
}

// TestExampleRedisPoolConfiguration shows how to configure Redis pool for different environments
func TestExampleRedisPoolConfiguration(t *testing.T) {
	// This test demonstrates creating different pool configurations.
	// In a real scenario, you would select one based on the environment.

	// Development configuration - lighter resource usage
	devConfig := RedisPoolConfig{
		Addr:         "localhost:6379",
		PoolSize:     10,
		MinIdleConns: 2,
	}

	// Production configuration - optimized for high throughput
	prodConfig := RedisPoolConfig{
		Addr:         "redis-cluster.prod:6379",
		PoolSize:     50,
		MinIdleConns: 10,
	}

	t.Logf("Example dev config pool size: %d", devConfig.PoolSize)
	t.Logf("Example prod config pool size: %d", prodConfig.PoolSize)

	// This test doesn't connect, it just shows the configuration objects.
}

// TestFullHandoffFlow demonstrates a complete, optimized handoff from publishing to consuming.
func TestFullHandoffFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Initialize optimized Redis configuration
	redisConfig := DefaultRedisPoolConfig()
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		redisConfig.Addr = addr
	}

	// Create optimized agent configuration
	agentConfig := OptimizedConfig{
		RedisConfig: redisConfig,
		LogLevel:    "error", // Use error to avoid verbose logs in tests
	}

	// Create optimized handoff agent
	agent, err := NewOptimizedHandoffAgent(agentConfig)
	if err != nil {
		t.Fatalf("Failed to create optimized handoff agent: %v", err)
	}
	defer agent.Close()

	// Register agent capabilities
	if err := agent.RegisterAgent(AgentCapabilities{Name: "golang-expert", MaxConcurrent: 2}); err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}
	if err := agent.RegisterAgent(AgentCapabilities{Name: "test-expert", MaxConcurrent: 2}); err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var processedHandoffID string
	var wg sync.WaitGroup
	wg.Add(1)

	// Start consuming handoffs for golang-expert
	go func() {
		_ = agent.ConsumeHandoffs(ctx, "golang-expert", func(ctx context.Context, h *Handoff) error {
			t.Logf("Processing handoff: %s", h.Metadata.HandoffID)
			processedHandoffID = h.Metadata.HandoffID
			wg.Done() // Signal that processing is done
			return nil
		})
	}()

	// Allow time for consumer to start
	time.Sleep(500 * time.Millisecond)

	// Example: Publish a handoff
	handoff := &Handoff{
		Metadata: Metadata{
			ProjectName: "agent-handoff-optimization",
			FromAgent:   "architect-expert",
			ToAgent:     "golang-expert",
			TaskContext: "Optimize Redis connection pooling",
			Priority:    PriorityHigh,
		},
		Content: Content{
			Summary:      "Implement connection pooling",
			Requirements: []string{"Implement connection pooling for Redis"},
		},
	}

	// Publish the handoff
	if err := agent.PublishHandoff(ctx, handoff); err != nil {
		t.Fatalf("Failed to publish handoff: %v", err)
	}

	t.Logf("Successfully published handoff: %s", handoff.Metadata.HandoffID)

	// Wait for the handoff to be processed, with a timeout
	if waitTimeout(&wg, 5*time.Second) {
		t.Fatal("Timeout waiting for handoff to be processed")
	}

	if processedHandoffID != handoff.Metadata.HandoffID {
		t.Errorf("Expected processed handoff ID to be %s, got %s", handoff.Metadata.HandoffID, processedHandoffID)
	}
}

// waitTimeout waits for the waitgroup for the specified duration.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
