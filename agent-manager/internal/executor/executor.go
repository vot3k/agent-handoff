package executor

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

	"github.com/vot3k/agent-handoff/agent-manager/internal/tools"
)

// ExecutionMode defines different ways to execute agents
type ExecutionMode int

const (
	ModeDispatcher ExecutionMode = iota // Current dispatcher mode
	ModeExecutor                        // Direct agent execution mode
	ModeHybrid                          // Hybrid mode with fallbacks
)

// AgentExecutionRequest contains all data needed to execute an agent
type AgentExecutionRequest struct {
	AgentName    string            `json:"agent_name"`
	ProjectName  string            `json:"project_name"`
	ProjectPath  string            `json:"project_path"`
	Payload      string            `json:"payload"`
	HandoffID    string            `json:"handoff_id"`
	FromAgent    string            `json:"from_agent"`
	Environment  map[string]string `json:"environment"`
	TaskContext  string            `json:"task_context"`
	Summary      string            `json:"summary"`
	Requirements []string          `json:"requirements"`
}

// AgentExecutionResponse contains the result of agent execution
type AgentExecutionResponse struct {
	Success      bool              `json:"success"`
	Output       string            `json:"output"`
	Error        string            `json:"error,omitempty"`
	Duration     time.Duration     `json:"duration"`
	Artifacts    []string          `json:"artifacts,omitempty"`
	NextHandoffs []NextHandoff     `json:"next_handoffs,omitempty"`
	Metadata     map[string]string `json:"metadata"`
}

// NextHandoff represents a follow-up handoff to be created
type NextHandoff struct {
	ToAgent  string `json:"to_agent"`
	Summary  string `json:"summary"`
	Context  string `json:"context"`
	Priority string `json:"priority"`
}

// ExecutionStrategy defines how to execute agents
type ExecutionStrategy interface {
	Execute(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error)
	CanHandle(agentName string, projectPath string, toolSet *tools.ToolSet) bool
	Priority() int
	Name() string
}

// AgentExecutor manages agent execution with multiple strategies
type AgentExecutor struct {
	strategies []ExecutionStrategy
	toolSet    *tools.ToolSet
	mode       ExecutionMode
}

// NewAgentExecutor creates a new agent executor with default strategies
func NewAgentExecutor(mode ExecutionMode) (*AgentExecutor, error) {
	toolSet, err := tools.DetectAvailableTools("")
	if err != nil {
		return nil, fmt.Errorf("failed to detect tools: %w", err)
	}

	executor := &AgentExecutor{
		toolSet: toolSet,
		mode:    mode,
	}

	// Register execution strategies in priority order
	executor.strategies = []ExecutionStrategy{
		NewToolDetectionStrategy(toolSet),
		NewBuiltInAgentStrategy(),
		NewScriptFallbackStrategy(),
	}

	log.Printf("✅ AgentExecutor initialized with %d strategies", len(executor.strategies))
	log.Printf("🔍 Available tools: %v", toolSet.ListAvailable())

	return executor, nil
}

// Execute runs an agent using the best available strategy
func (e *AgentExecutor) Execute(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	start := time.Now()

	log.Printf("🚀 Executing agent '%s' for project '%s'", req.AgentName, req.ProjectName)
	log.Printf("📄 Task: %s", req.Summary)

	// Find the best strategy for this request
	strategy := e.selectStrategy(req)
	if strategy == nil {
		return nil, fmt.Errorf("no suitable execution strategy found for agent '%s'", req.AgentName)
	}

	log.Printf("🔧 Using strategy: %s", strategy.Name())

	// Execute using the selected strategy
	response, err := strategy.Execute(ctx, req)
	if err != nil {
		return &AgentExecutionResponse{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
			Metadata: map[string]string{
				"strategy": strategy.Name(),
				"agent":    req.AgentName,
				"project":  req.ProjectName,
			},
		}, err
	}

	// Enhance response with metadata
	response.Duration = time.Since(start)
	if response.Metadata == nil {
		response.Metadata = make(map[string]string)
	}
	response.Metadata["strategy"] = strategy.Name()
	response.Metadata["agent"] = req.AgentName
	response.Metadata["project"] = req.ProjectName

	log.Printf("✅ Agent '%s' completed successfully in %v", req.AgentName, response.Duration)

	return response, nil
}

// selectStrategy finds the best execution strategy for the request
func (e *AgentExecutor) selectStrategy(req AgentExecutionRequest) ExecutionStrategy {
	var bestStrategy ExecutionStrategy
	highestPriority := -1

	for _, strategy := range e.strategies {
		if strategy.CanHandle(req.AgentName, req.ProjectPath, e.toolSet) {
			if strategy.Priority() > highestPriority {
				bestStrategy = strategy
				highestPriority = strategy.Priority()
			}
		}
	}

	return bestStrategy
}

// ExtractExecutionRequest creates an execution request from handoff payload
func ExtractExecutionRequest(payload string, projectName string) (*AgentExecutionRequest, error) {
	var handoffData map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &handoffData); err != nil {
		return nil, fmt.Errorf("failed to parse handoff payload: %w", err)
	}

	req := &AgentExecutionRequest{
		Payload:     payload,
		ProjectName: projectName,
		Environment: make(map[string]string),
	}

	// Extract metadata
	if metadata, ok := handoffData["metadata"].(map[string]interface{}); ok {
		if handoffID, ok := metadata["handoff_id"].(string); ok {
			req.HandoffID = handoffID
		}
		if fromAgent, ok := metadata["from_agent"].(string); ok {
			req.FromAgent = fromAgent
		}
		if toAgent, ok := metadata["to_agent"].(string); ok {
			req.AgentName = toAgent
		}
		if taskContext, ok := metadata["task_context"].(string); ok {
			req.TaskContext = taskContext
		}
	}

	// Extract content
	if content, ok := handoffData["content"].(map[string]interface{}); ok {
		if summary, ok := content["summary"].(string); ok {
			req.Summary = summary
		}
		if requirements, ok := content["requirements"].([]interface{}); ok {
			for _, req_item := range requirements {
				if reqStr, ok := req_item.(string); ok {
					req.Requirements = append(req.Requirements, reqStr)
				}
			}
		}
	}

	// Set environment variables
	req.Environment["AGENT_PROJECT_NAME"] = projectName
	req.Environment["HANDOFF_ID"] = req.HandoffID
	req.Environment["FROM_AGENT"] = req.FromAgent

	// Detect project path
	req.ProjectPath = DetectProjectPath(projectName)

	return req, nil
}

// DetectProjectPath attempts to find the project directory
func DetectProjectPath(projectName string) string {
	// Strategy 1: Check environment variable
	if path := os.Getenv("PROJECT_ROOT"); path != "" {
		return path
	}

	// Strategy 2: Common development paths (configurable via environment)
	var commonPaths []string

	// Check for user-configured project paths
	if devPath := os.Getenv("AGENT_DEV_PATH"); devPath != "" {
		commonPaths = append(commonPaths, fmt.Sprintf("%s/%s", devPath, projectName))
	}

	// Add standard relative paths
	commonPaths = append(commonPaths, []string{
		fmt.Sprintf("../%s", projectName),
		fmt.Sprintf("/tmp/projects/%s", projectName),
		fmt.Sprintf("./%s", projectName),
	}...)

	for _, path := range commonPaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				log.Printf("📁 Detected project path: %s", absPath)
				return absPath
			}
		}
	}

	// Strategy 3: Current working directory as fallback
	if cwd, err := os.Getwd(); err == nil {
		log.Printf("📁 Using current directory as project path: %s", cwd)
		return cwd
	}

	return ""
}

// CreateTempPayloadFile creates a temporary file containing the handoff payload
func CreateTempPayloadFile(payload string) (string, error) {
	tempFile, err := os.CreateTemp("", "handoff-payload-*.json")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	if _, err := tempFile.WriteString(payload); err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

// ExtractTaskDescription creates a meaningful task description from handoff payload
func ExtractTaskDescription(payload map[string]interface{}) string {
	// Try to extract summary from content
	if content, ok := payload["content"].(map[string]interface{}); ok {
		if summary, ok := content["summary"].(string); ok && summary != "" {
			return summary
		}
	}

	// Try to extract from top-level summary
	if summary, ok := payload["summary"].(string); ok && summary != "" {
		return summary
	}

	// Try to extract from metadata
	if metadata, ok := payload["metadata"].(map[string]interface{}); ok {
		if fromAgent, ok := metadata["from_agent"].(string); ok {
			return fmt.Sprintf("Process handoff from %s agent", fromAgent)
		}
	}

	// Fallback description
	return "Process agent handoff task"
}

// SetupCommand configures a command with environment and working directory
func SetupCommand(cmd *exec.Cmd, req AgentExecutionRequest) {
	// Set working directory
	if req.ProjectPath != "" {
		cmd.Dir = req.ProjectPath
	}

	// Set environment variables
	env := os.Environ()
	for key, value := range req.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Env = env

	log.Printf("🔍 Command: %s", strings.Join(cmd.Args, " "))
	log.Printf("📁 Working directory: %s", cmd.Dir)
	log.Printf("🌍 Environment: %v", req.Environment)
}
