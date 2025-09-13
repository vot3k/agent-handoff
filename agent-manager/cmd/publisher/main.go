package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-redis/redis/v8"
)

// TestHandoff creates a test handoff message
type TestHandoff struct {
	Metadata struct {
		ProjectName string    `json:"project_name"`
		FromAgent   string    `json:"from_agent"`
		ToAgent     string    `json:"to_agent"`
		Timestamp   time.Time `json:"timestamp"`
		TaskContext string    `json:"task_context"`
		Priority    string    `json:"priority"`
		HandoffID   string    `json:"handoff_id"`
	} `json:"metadata"`
	Content struct {
		Summary          string                 `json:"summary"`
		Requirements     []string               `json:"requirements"`
		Artifacts        map[string][]string    `json:"artifacts"`
		TechnicalDetails map[string]interface{} `json:"technical_details"`
		NextSteps        []string               `json:"next_steps"`
	} `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <from_agent> <to_agent> [message]\n", os.Args[0])
		fmt.Println("Example: go run test-publisher.go architect-expert api-expert")
		os.Exit(1)
	}

	fromAgent := os.Args[1]
	toAgent := os.Args[2]
	message := "Test handoff message"
	if len(os.Args) > 3 {
		message = os.Args[3]
	}

	// Get project name from current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}
	projectName := filepath.Base(wd)

	ctx := context.Background()
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	// Test Redis connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis at %s: %v", redisAddr, err)
	}

	// Create test handoff
	handoff := TestHandoff{
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	handoff.Metadata.ProjectName = projectName
	handoff.Metadata.FromAgent = fromAgent
	handoff.Metadata.ToAgent = toAgent
	handoff.Metadata.Timestamp = time.Now()
	handoff.Metadata.TaskContext = "test-workflow"
	handoff.Metadata.Priority = "normal"
	handoff.Metadata.HandoffID = fmt.Sprintf("test-%d", time.Now().UnixNano())

	handoff.Content.Summary = message
	handoff.Content.Requirements = []string{
		"Process the incoming request",
		"Generate appropriate output",
		"Ensure proper error handling",
	}
	handoff.Content.Artifacts = map[string][]string{
		"created":  {},
		"modified": {},
		"reviewed": {},
	}
	handoff.Content.TechnicalDetails = map[string]interface{}{
		"test_mode":   true,
		"environment": "development",
		"source":      "test-publisher",
		"timestamp":   time.Now().Unix(),
	}
	handoff.Content.NextSteps = []string{
		"Validate input requirements",
		"Execute agent-specific logic",
		"Generate output artifacts",
		"Update handoff status",
	}

	// Serialize handoff
	payload, err := json.MarshalIndent(handoff, "", "  ")
	if err != nil {
		log.Fatalf("Failed to serialize handoff: %v", err)
	}

	// Determine target queue
	queueName := fmt.Sprintf("handoff:project:%s:queue:%s", projectName, toAgent)

	// Store handoff data in Redis with expiration (24 hours)
	handoffKey := fmt.Sprintf("handoff:%s", handoff.Metadata.HandoffID)
	if err := rdb.Set(ctx, handoffKey, payload, 24*time.Hour).Err(); err != nil {
		log.Fatalf("Failed to store handoff data: %v", err)
	}

	// Push to priority queue (using sorted set like the existing system)
	// Priority: normal = 3, with timestamp for FIFO within same priority
	score := 3.0 + float64(time.Now().UnixNano())/1e18
	if err := rdb.ZAdd(ctx, queueName, &redis.Z{
		Score:  score,
		Member: handoff.Metadata.HandoffID,
	}).Err(); err != nil {
		log.Fatalf("Failed to queue handoff: %v", err)
	}

	fmt.Printf("✅ Published handoff to queue: %s\n", queueName)
	fmt.Printf("Project: %s\n", projectName)
	fmt.Printf("📨 Handoff ID: %s\n", handoff.Metadata.HandoffID)
	fmt.Printf("🔄 From: %s → To: %s\n", fromAgent, toAgent)
	fmt.Printf("📝 Summary: %s\n", message)
	fmt.Printf("📊 Queue depth after push: ")

	// Check queue depth (using ZCard for sorted sets)
	depth, err := rdb.ZCard(ctx, queueName).Result()
	if err != nil {
		fmt.Printf("(error: %v)\n", err)
	} else {
		fmt.Printf("%d\n", depth)
	}

	fmt.Println("\n🚀 You can now run the agent-manager to process this handoff:")
	fmt.Println("   go run ./cmd/manager/main.go")
}
