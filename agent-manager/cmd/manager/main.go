package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/vot3k/agent-handoff/agent-manager/internal/executor"
)

// HandoffPayload represents the structure of messages from the queue.
// This matches the structure used in the existing handoff system.
type HandoffPayload struct {
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
	ctx := context.Background()

	// Parse command line flags
	mode := flag.String("mode", "dispatcher", "Operation mode: dispatcher|executor")
	agentName := flag.String("agent", "", "Agent name (for executor mode)")
	payloadFile := flag.String("payload-file", "", "Payload JSON file")
	payloadStdin := flag.Bool("payload-stdin", false, "Read payload from stdin")
	projectName := flag.String("project", "", "Project name")
	flag.Parse()

	// Handle different execution modes
	switch *mode {
	case "executor":
		runAgentExecutor(*agentName, *projectName, *payloadFile, *payloadStdin)
		return
	case "dispatcher":
		// Continue with dispatcher mode below
	default:
		log.Fatalf("Unknown mode: %s. Use dispatcher or executor", *mode)
	}

	// Dispatcher mode setup
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// Initialize agent executor - only built-in executor now
	agentExecutor, err := executor.NewAgentExecutor(executor.ModeExecutor)
	if err != nil {
		log.Fatalf("❌ Failed to initialize built-in executor: %v", err)
	}
	log.Printf("✅ Using built-in agent executor with tool-agnostic execution")

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	// Test Redis connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis at %s: %v", redisAddr, err)
	}

	log.Printf("Agent Manager service started. Listening for tasks...")
	log.Printf("Redis address: %s", redisAddr)

	for {
		// Scan for all project-specific queues
		queuePattern := "handoff:project:*:queue:*"
		queues, err := rdb.Keys(ctx, queuePattern).Result()
		if err != nil {
			log.Printf("Error scanning for queues with pattern %s: %v", queuePattern, err)
			time.Sleep(5 * time.Second) // Wait before retrying scan
			continue
		}

		if len(queues) == 0 {
			time.Sleep(2 * time.Second) // No active queues, wait a bit
			continue
		}

		// Check each queue for messages
		for _, queueName := range queues {
			// Pop from priority queue (lowest score first)
			result, err := rdb.ZPopMin(ctx, queueName, 1).Result()
			if err != nil {
				if err != redis.Nil {
					log.Printf("Error checking queue %s: %v", queueName, err)
				}
				continue
			}

			if len(result) == 0 {
				continue
			}

			// result[0].Member contains the handoff ID
			handoffID := result[0].Member.(string)
			log.Printf("Received task from queue: %s, handoff ID: %s", queueName, handoffID)

			// Extract project and agent name from queue name
			projectName, agentName := extractProjectAndAgentName(queueName)
			if agentName == "" || projectName == "" {
				log.Printf("Could not extract project/agent name from queue: %s", queueName)
				continue
			}

			// Retrieve the full handoff data from Redis
			handoffKey := fmt.Sprintf("handoff:%s", handoffID)
			taskPayload, err := rdb.Get(ctx, handoffKey).Result()
			if err != nil {
				if err == redis.Nil {
					log.Printf("Handoff data not found for ID: %s", handoffID)
				} else {
					log.Printf("Error retrieving handoff %s: %v", handoffID, err)
				}
				continue
			}

			// Dispatch the task in a new goroutine using built-in executor
			go dispatchWithBuiltInExecutor(projectName, agentName, taskPayload, agentExecutor)
		}

		// Small delay to prevent busy-waiting if all queues were empty
		time.Sleep(100 * time.Millisecond)
	}
}

// extractProjectAndAgentName extracts the project and agent name from a queue name
func extractProjectAndAgentName(queueName string) (string, string) {
	// Expected format: "handoff:project:{projectName}:queue:{agentName}"
	parts := strings.Split(queueName, ":")
	if len(parts) == 5 && parts[0] == "handoff" && parts[1] == "project" && parts[3] == "queue" {
		return parts[2], parts[4]
	}
	return "", ""
}

// archiveHandoff saves the successful handoff payload to the file system.
func archiveHandoff(payload string, handoffData *HandoffPayload, handoffID string) error {
	var ts time.Time
	if !handoffData.Metadata.Timestamp.IsZero() {
		ts = handoffData.Metadata.Timestamp
	} else {
		ts = time.Now()
	}

	datePath := ts.UTC().Format("2006-01-02")
	// Include project name in archive path
	projectName := handoffData.Metadata.ProjectName
	if projectName == "" {
		projectName = "unknown-project"
	}

	fileName := fmt.Sprintf("%s-%s-%s.json",
		ts.UTC().Format("20060102T150405Z"),
		handoffData.Metadata.ToAgent,
		handoffID[:8])

	archiveDir := filepath.Join("archive", projectName, datePath)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	filePath := filepath.Join(archiveDir, fileName)
	log.Printf("[Archive] Saving handoff to %s", filePath)

	return os.WriteFile(filePath, []byte(payload), 0644)
}

// runAgentExecutor executes a single agent directly (executor mode)
func runAgentExecutor(agentName, projectName, payloadFile string, payloadStdin bool) {
	if agentName == "" {
		log.Fatal("❌ Agent name is required in executor mode")
	}

	// Get payload
	payload := getPayload(payloadFile, payloadStdin)
	if payload == "" {
		log.Fatal("❌ Payload is required (use --payload-file or --payload-stdin)")
	}

	// Initialize executor
	agentExecutor, err := executor.NewAgentExecutor(executor.ModeExecutor)
	if err != nil {
		log.Fatalf("❌ Failed to initialize executor: %v", err)
	}

	// Create execution request
	req, err := executor.ExtractExecutionRequest(payload, projectName)
	if err != nil {
		log.Fatalf("❌ Failed to parse execution request: %v", err)
	}

	// Ensure agent name is set
	if req.AgentName == "" {
		req.AgentName = agentName
	}

	log.Printf("🚀 Executing agent '%s' for project '%s'", req.AgentName, req.ProjectName)

	// Execute agent
	ctx := context.Background()
	response, err := agentExecutor.Execute(ctx, *req)
	if err != nil {
		log.Fatalf("❌ Agent execution failed: %v", err)
	}

	if response.Success {
		log.Printf("✅ Agent '%s' completed successfully in %v", req.AgentName, response.Duration)
		log.Printf("📄 Output:\n%s", response.Output)

		if len(response.Artifacts) > 0 {
			log.Printf("📦 Artifacts: %v", response.Artifacts)
		}

		if len(response.NextHandoffs) > 0 {
			log.Printf("🔄 Next handoffs: %d pending", len(response.NextHandoffs))
			for i, handoff := range response.NextHandoffs {
				log.Printf("   %d. %s: %s", i+1, handoff.ToAgent, handoff.Summary)
			}
		}

		os.Exit(0)
	} else {
		log.Printf("❌ Agent '%s' failed: %s", req.AgentName, response.Error)
		os.Exit(1)
	}
}

// dispatchWithBuiltInExecutor dispatches using the built-in executor
func dispatchWithBuiltInExecutor(projectName, agentName, payload string, agentExecutor *executor.AgentExecutor) {
	log.Printf("[Dispatch] Processing task for project '%s', agent '%s' (built-in)", projectName, agentName)

	var handoff HandoffPayload
	if err := json.Unmarshal([]byte(payload), &handoff); err != nil {
		log.Printf("[ERROR] Failed to decode task payload: %v", err)
		return
	}

	handoffID := handoff.Metadata.HandoffID
	if handoffID == "" {
		log.Printf("[ERROR] Missing handoff ID in payload")
		return
	}

	log.Printf("[Dispatch] Invoking built-in agent '%s' for handoff '%s' in project '%s'", agentName, handoffID, projectName)

	// Create execution request
	req, err := executor.ExtractExecutionRequest(payload, projectName)
	if err != nil {
		log.Printf("[ERROR] Failed to create execution request: %v", err)
		return
	}

	// Execute using built-in executor
	ctx := context.Background()
	response, err := agentExecutor.Execute(ctx, *req)
	if err != nil {
		log.Printf("[FAILURE] Built-in agent '%s' failed: %v", agentName, err)
		return
	}

	if response.Success {
		log.Printf("[SUCCESS] Built-in agent '%s' completed for handoff '%s' in %v", agentName, handoffID, response.Duration)
		log.Printf("[OUTPUT]\n%s", response.Output)

		if len(response.Artifacts) > 0 {
			log.Printf("[ARTIFACTS] %v", response.Artifacts)
		}

		// Handle next handoffs
		if len(response.NextHandoffs) > 0 {
			log.Printf("[HANDOFFS] Creating %d follow-up handoffs", len(response.NextHandoffs))
			// TODO: Implement follow-up handoff creation
		}

		if err := archiveHandoff(payload, &handoff, handoffID); err != nil {
			log.Printf("[CRITICAL] Agent '%s' succeeded but failed to archive: %v", agentName, err)
		}
	} else {
		log.Printf("[FAILURE] Built-in agent '%s' failed: %s", agentName, response.Error)
	}
}

// getPayload reads payload from file or stdin
func getPayload(payloadFile string, payloadStdin bool) string {
	if payloadStdin {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("❌ Failed to read payload from stdin: %v", err)
		}
		return strings.TrimSpace(string(content))
	}

	if payloadFile != "" {
		content, err := os.ReadFile(payloadFile)
		if err != nil {
			log.Fatalf("❌ Failed to read payload file '%s': %v", payloadFile, err)
		}
		return strings.TrimSpace(string(content))
	}

	return ""
}
