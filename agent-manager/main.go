package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// HandoffPayload represents the structure of messages from the queue.
// This matches the structure used in the existing handoff system.
type HandoffPayload struct {
	Metadata struct {
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

// HandoffQueueMessage wraps the handoff payload as stored in Redis
type HandoffQueueMessage struct {
	HandoffID string         `json:"handoff_id"`
	Queue     string         `json:"queue"`
	Timestamp time.Time      `json:"timestamp"`
	Priority  string         `json:"priority"`
	Payload   HandoffPayload `json:"payload"`
}

func main() {
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

	// Define all the agent queues the manager will listen to.
	// This list should contain every agent that can receive a handoff.
	queues := []string{
		"handoff:queue:api-expert",
		"handoff:queue:golang-expert",
		"handoff:queue:typescript-expert",
		"handoff:queue:test-expert",
		"handoff:queue:tech-writer",
		"handoff:queue:project-optimizer",
		"handoff:queue:architecture-analyzer",
		"handoff:queue:architect-expert",
		"handoff:queue:agent-manager",
		"handoff:queue:devops-expert",
		"handoff:queue:security-expert",
		"handoff:queue:product-manager",
		"handoff:queue:project-manager",
	}

	log.Printf("Agent Manager service started. Listening for tasks on %d queues...", len(queues))
	log.Printf("Redis address: %s", redisAddr)
	log.Printf("Monitored queues: %v", queues)

	for {
		// Check each queue for messages (since we can't block on multiple sorted sets)
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

			// Extract agent name from queue name
			agentName := extractAgentName(queueName)
			if agentName == "" {
				log.Printf("Could not extract agent name from queue: %s", queueName)
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

			// Dispatch the task in a new goroutine so the manager can
			// immediately go back to listening for more tasks.
			go dispatchAndArchiveTask(agentName, taskPayload)
		}

		// Small delay to prevent busy-waiting
		time.Sleep(100 * time.Millisecond)
	}
}

// extractAgentName extracts the agent name from a queue name
func extractAgentName(queueName string) string {
	// Expected format: "handoff:queue:agent-name"
	parts := strings.Split(queueName, ":")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// dispatchAndArchiveTask handles the execution and archival of a single task.
func dispatchAndArchiveTask(agentName, payload string) {
	log.Printf("[Dispatch] Processing task for agent '%s'", agentName)

	// Parse the payload as HandoffPayload directly
	var handoff HandoffPayload
	if err := json.Unmarshal([]byte(payload), &handoff); err != nil {
		log.Printf("[ERROR] Failed to decode task payload: %v", err)
		log.Printf("[ERROR] Payload: %s", payload)
		return
	}

	handoffID := handoff.Metadata.HandoffID
	log.Printf("[Debug] Parsed HandoffPayload, ID: %s", handoffID)

	if handoffID == "" {
		log.Printf("[ERROR] Missing handoff ID in payload")
		log.Printf("[DEBUG] Payload content: %s", payload)
		return
	}

	log.Printf("[Dispatch] Invoking agent '%s' for handoff '%s'", agentName, handoffID)

	// Invoke the Agent Executor. This script is the bridge to your agent tooling.
	// We pass the agent's name and its task payload.
	cmd := exec.Command("./run-agent.sh", agentName, payload)
	cmd.Dir = "." // Ensure we're in the right directory

	// Execute the command and capture its output (stdout and stderr).
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[FAILURE] Agent '%s' failed: %v\n--- Output ---\n%s\n--------------", agentName, err, string(output))
		// In a production system, you would push the payload to a DLQ for failed tasks
		return
	}

	log.Printf("[SUCCESS] Agent '%s' completed for handoff '%s'.", agentName, handoffID)
	log.Printf("[OUTPUT]\n%s", string(output))

	// If the agent was successful, archive the original handoff payload.
	if err := archiveHandoff(payload, &handoff, handoffID); err != nil {
		log.Printf("[CRITICAL] Agent '%s' succeeded but failed to archive: %v", agentName, err)
	}
}

// archiveHandoff saves the successful handoff payload to the file system.
func archiveHandoff(payload string, handoffData *HandoffPayload, handoffID string) error {
	// Use the timestamp and agent name to create a unique, chronological filename.
	var ts time.Time
	if !handoffData.Metadata.Timestamp.IsZero() {
		ts = handoffData.Metadata.Timestamp
	} else if !handoffData.CreatedAt.IsZero() {
		ts = handoffData.CreatedAt
	} else {
		ts = time.Now()
	}

	datePath := ts.UTC().Format("2006-01-02")
	fileName := fmt.Sprintf("%s-%s-%s.json", 
		ts.UTC().Format("20060102T150405Z"), 
		handoffData.Metadata.ToAgent,
		handoffID[:8]) // Use first 8 chars of handoff ID

	archiveDir := filepath.Join("archive", datePath)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	filePath := filepath.Join(archiveDir, fileName)
	log.Printf("[Archive] Saving handoff to %s", filePath)
	
	if err := os.WriteFile(filePath, []byte(payload), 0644); err != nil {
		return fmt.Errorf("failed to write archive file: %w", err)
	}

	return nil
}