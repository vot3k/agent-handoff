package repository

import (
	"fmt"
	"strings"
)

// Redis key constants
const (
	HandoffPrefix     = "handoff"
	ProjectPrefix     = "project"
	QueuePrefix       = "queue"
	HandoffKeyPattern = "handoff:%s"
	QueueKeyPattern   = "handoff:project:%s:queue:%s"
)

// GetHandoffKey generates the Redis key for a handoff
func GetHandoffKey(handoffID string) string {
	return fmt.Sprintf(HandoffKeyPattern, handoffID)
}

// GetQueueKey generates the Redis key for a queue
func GetQueueKey(projectName, agentName string) string {
	return fmt.Sprintf(QueueKeyPattern, projectName, agentName)
}

// GetHandoffIDFromKey extracts handoff ID from a Redis key
func GetHandoffIDFromKey(key string) string {
	parts := strings.Split(key, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// GetProjectAndAgentFromQueueKey extracts project name and agent name from queue key
func GetProjectAndAgentFromQueueKey(queueKey string) (string, string) {
	parts := strings.Split(queueKey, ":")
	if len(parts) == 5 && parts[0] == HandoffPrefix && parts[1] == ProjectPrefix && parts[3] == QueuePrefix {
		return parts[2], parts[4]
	}
	return "", ""
}

// GetHandoffListKey generates the key for listing all handoffs (using a set)
func GetHandoffListKey(projectName string) string {
	if projectName != "" {
		return fmt.Sprintf("handoff:project:%s:list", projectName)
	}
	return "handoff:list"
}

// GetHandoffProjectSetKey generates the key for a set of handoffs belonging to a project
func GetHandoffProjectSetKey(projectName string) string {
	return fmt.Sprintf("handoff:project:%s:set", projectName)
}
