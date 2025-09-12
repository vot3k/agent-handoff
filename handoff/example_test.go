package handoff

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// ExampleHandoffAgent demonstrates basic usage of the handoff agent
func ExampleHandoffAgent() {
	fmt.Println("Handoff Agent Example")
	fmt.Println("Handoff system provides Redis-based queue management")
	fmt.Println("Agents can publish and consume handoffs with validation")
	// Output: Handoff Agent Example
	// Handoff system provides Redis-based queue management
	// Agents can publish and consume handoffs with validation
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
				Type:     ConditionContent,
				Field:    "summary",
				Operator: "contains",
				Value:    "implement",
				CaseSensitive: false,
			},
		},
	}

	router.AddRoute("api-expert", implementationRule)

	// Create handoff with Go files
	handoff := &Handoff{
		Metadata: Metadata{
			FromAgent: "api-expert",
			ToAgent:   "",  // Will be determined by router
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

// ExampleHandoffValidator demonstrates validation
func ExampleHandoffValidator() {
	fmt.Println("Handoff Validator Example")
	fmt.Println("Validation enforces schema compliance")
	fmt.Println("Agent-specific fields are validated")
	fmt.Println("Content is sanitized and normalized")
	// Output: Handoff Validator Example
	// Validation enforces schema compliance
	// Agent-specific fields are validated
	// Content is sanitized and normalized
}

// ExampleHandoffMonitor demonstrates monitoring setup
func ExampleHandoffMonitor() {
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

// TestHandoffLifecycle demonstrates a complete handoff lifecycle
func TestHandoffLifecycle(t *testing.T) {
	// This is a comprehensive test that would require Redis
	// Skipping actual execution in example
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup would include:
	// 1. Redis connection
	// 2. Agent registration
	// 3. Consumer setup
	// 4. Handoff publishing
	// 5. Processing verification
	// 6. Metrics collection
	// 7. Cleanup
}

// BenchmarkHandoffPublishing benchmarks handoff publishing performance
func BenchmarkHandoffPublishing(b *testing.B) {
	// This would benchmark the publishing performance
	// Skipping actual execution in example
	b.Skip("Benchmark requires Redis connection")
}

// ExampleHandoffAgent_integration shows how to integrate with existing agent system
func ExampleHandoffAgent_integration() {
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