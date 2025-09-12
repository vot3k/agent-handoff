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
		ProjectName string    `json:"project_name"`
		FromAgent   string    `json:"from_agent"`
		ToAgent     string    `json:"to_agent"`
		Timestamp   time.Time `json:"timestamp"`
		TaskContext string    `json:"task_context"`
		Priority    string    `json:"priority"`
		HandoffID   string    `json:"handoff_id"`
	}`json:"metadata"`
	Content struct {
		Summary          string                 `json:"summary"`
		Requirements     []string               `json:"requirements"`
		Artifacts        map[string][]string    `json:"artifacts"`
		TechnicalDetails map[string]interface{} `json:"technical_details"`
		NextSteps        []string               `json:"next_steps"`
	}`json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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

			// Dispatch the task in a new goroutine
			go dispatchAndArchiveTask(projectName, agentName, taskPayload)
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

// dispatchAndArchiveTask handles the execution and archival of a single task.
func dispatchAndArchiveTask(projectName, agentName, payload string) {
	log.Printf("[Dispatch] Processing task for project '%s', agent '%s'", projectName, agentName)

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

	log.Printf("[Dispatch] Invoking agent '%s' for handoff '%s' in project '%s'", agentName, handoffID, projectName)

	// Set environment variable for the agent
	env := os.Environ()
	env = append(env, fmt.Sprintf("AGENT_PROJECT_NAME=%s", projectName))

	cmd := exec.Command("./run-agent.sh", agentName, payload)
	cmd.Dir = "." // Ensure we're in the right directory
	cmd.Env = env // Pass the environment with the project name

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[FAILURE] Agent '%s' failed: %v\n--- Output ---\n%s\n--------------", agentName, err, string(output))
		return
	}

	log.Printf("[SUCCESS] Agent '%s' completed for handoff '%s'.", agentName, handoffID)
	log.Printf("[OUTPUT]\n%s", string(output))

	if err := archiveHandoff(payload, &handoff, handoffID); err != nil {
		log.Printf("[CRITICAL] Agent '%s' succeeded but failed to archive: %v", agentName, err)
	}
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
