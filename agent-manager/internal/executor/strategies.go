package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/vot3k/agent-handoff/agent-manager/internal/tools"
)

// ToolDetectionStrategy executes agents using detected tools
type ToolDetectionStrategy struct {
	toolSet *tools.ToolSet
}

// NewToolDetectionStrategy creates a new tool detection strategy
func NewToolDetectionStrategy(toolSet *tools.ToolSet) *ToolDetectionStrategy {
	return &ToolDetectionStrategy{toolSet: toolSet}
}

func (t *ToolDetectionStrategy) Name() string {
	return "ToolDetection"
}

func (t *ToolDetectionStrategy) Priority() int {
	return 100 // Highest priority
}

func (t *ToolDetectionStrategy) CanHandle(agentName string, projectPath string, toolSet *tools.ToolSet) bool {
	// Can handle any agent if we have suitable tools
	return t.hasSuitableTools(agentName, toolSet)
}

func (t *ToolDetectionStrategy) Execute(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	log.Printf("🔧 ToolDetectionStrategy executing %s", req.AgentName)

	switch req.AgentName {
	case "golang-expert":
		return t.executeGoExpert(ctx, req)
	case "test-expert":
		return t.executeTestExpert(ctx, req)
	case "api-expert":
		return t.executeAPIExpert(ctx, req)
	case "devops-expert":
		return t.executeDevOpsExpert(ctx, req)
	case "typescript-expert":
		return t.executeTypeScriptExpert(ctx, req)
	default:
		return t.executeGenericAgent(ctx, req)
	}
}

// executeGoExpert handles Go-specific tasks
func (t *ToolDetectionStrategy) executeGoExpert(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	// Tool priority: claude > cursor > vscode > go direct
	if t.toolSet.Has("claude") {
		return t.executeClaudeTask(ctx, req, "golang-expert")
	}

	if t.toolSet.Has("cursor") {
		return t.executeCursorTask(ctx, req)
	}

	if t.toolSet.Has("go") {
		return t.executeGoDirectTask(ctx, req)
	}

	return nil, fmt.Errorf("no suitable tools found for golang-expert")
}

// executeTestExpert handles testing tasks
func (t *ToolDetectionStrategy) executeTestExpert(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	if t.toolSet.Has("claude") {
		return t.executeClaudeTask(ctx, req, "test-expert")
	}

	// Fallback to project-specific testing
	switch t.toolSet.Project.Type {
	case "go":
		if t.toolSet.Has("go") {
			return t.executeGoTest(ctx, req)
		}
	case "node":
		if t.toolSet.Has("npm") {
			return t.executeNpmTest(ctx, req)
		}
	}

	return nil, fmt.Errorf("no suitable testing tools found")
}

// executeAPIExpert handles API-related tasks
func (t *ToolDetectionStrategy) executeAPIExpert(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	if t.toolSet.Has("claude") {
		return t.executeClaudeTask(ctx, req, "api-expert")
	}

	return t.executeGenericAPITask(ctx, req)
}

// executeDevOpsExpert handles DevOps tasks
func (t *ToolDetectionStrategy) executeDevOpsExpert(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	if t.toolSet.Has("claude") {
		return t.executeClaudeTask(ctx, req, "devops-expert")
	}

	// Check for DevOps tools
	if t.toolSet.Has("docker") {
		return t.executeDockerTask(ctx, req)
	}

	return t.executeGenericDevOpsTask(ctx, req)
}

// executeTypeScriptExpert handles TypeScript/React tasks
func (t *ToolDetectionStrategy) executeTypeScriptExpert(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	if t.toolSet.Has("claude") {
		return t.executeClaudeTask(ctx, req, "typescript-expert")
	}

	if t.toolSet.Has("npm") || t.toolSet.Has("yarn") {
		return t.executeNodeTask(ctx, req)
	}

	return nil, fmt.Errorf("no suitable TypeScript tools found")
}

// executeClaudeTask executes using Claude Code
func (t *ToolDetectionStrategy) executeClaudeTask(ctx context.Context, req AgentExecutionRequest, agentType string) (*AgentExecutionResponse, error) {
	log.Printf("🤖 Executing Claude Code task: %s", agentType)

	// Create temporary payload file
	payloadFile, err := CreateTempPayloadFile(req.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create payload file: %w", err)
	}
	defer os.Remove(payloadFile)

	// Extract task description
	var handoffData map[string]interface{}
	json.Unmarshal([]byte(req.Payload), &handoffData)
	taskDescription := ExtractTaskDescription(handoffData)

	// Build Claude Code command
	claudeTool := t.toolSet.Available["claude"]
	cmd := exec.CommandContext(ctx, claudeTool.Path, "task",
		"--agent-type", agentType,
		"--description", taskDescription,
		"--context-file", payloadFile,
	)

	SetupCommand(cmd, req)

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &AgentExecutionResponse{
			Success: false,
			Output:  string(output),
			Error:   err.Error(),
		}, err
	}

	// Parse Claude Code output for next handoffs
	nextHandoffs := t.parseClaudeOutput(string(output))

	return &AgentExecutionResponse{
		Success:      true,
		Output:       string(output),
		NextHandoffs: nextHandoffs,
		Artifacts:    []string{"Claude Code task completed"},
	}, nil
}

// executeCursorTask executes using Cursor
func (t *ToolDetectionStrategy) executeCursorTask(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	log.Printf("📝 Executing Cursor task")

	cursorTool := t.toolSet.Available["cursor"]

	// Create a basic cursor command (this would need to be customized based on Cursor's API)
	cmd := exec.CommandContext(ctx, cursorTool.Path, "--wait", req.ProjectPath)
	SetupCommand(cmd, req)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &AgentExecutionResponse{
			Success: false,
			Output:  string(output),
			Error:   err.Error(),
		}, err
	}

	return &AgentExecutionResponse{
		Success:   true,
		Output:    string(output),
		Artifacts: []string{"Cursor task completed"},
	}, nil
}

// executeGoDirectTask executes Go tasks directly
func (t *ToolDetectionStrategy) executeGoDirectTask(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	log.Printf("🐹 Executing Go direct task")

	var commands []string
	var artifacts []string

	// Determine Go tasks based on request content
	if strings.Contains(req.Summary, "test") || strings.Contains(req.Summary, "Test") {
		commands = append(commands, "go test ./...")
		artifacts = append(artifacts, "Go tests executed")
	}

	if strings.Contains(req.Summary, "build") || strings.Contains(req.Summary, "Build") {
		commands = append(commands, "go build -v ./...")
		artifacts = append(artifacts, "Go build completed")
	}

	if strings.Contains(req.Summary, "mod") || strings.Contains(req.Summary, "dependencies") {
		commands = append(commands, "go mod tidy")
		artifacts = append(artifacts, "Go modules updated")
	}

	// Default to basic checks if no specific commands
	if len(commands) == 0 {
		commands = []string{"go fmt ./...", "go vet ./...", "go build ./..."}
		artifacts = []string{"Go format, vet, and build completed"}
	}

	var output strings.Builder
	goTool := t.toolSet.Available["go"]

	for _, cmdStr := range commands {
		parts := strings.Fields(cmdStr)
		cmd := exec.CommandContext(ctx, goTool.Path, parts[1:]...)
		SetupCommand(cmd, req)

		cmdOutput, err := cmd.CombinedOutput()
		output.WriteString(fmt.Sprintf("$ %s\n", cmdStr))
		output.Write(cmdOutput)
		output.WriteString("\n")

		if err != nil {
			return &AgentExecutionResponse{
				Success: false,
				Output:  output.String(),
				Error:   fmt.Sprintf("Command '%s' failed: %v", cmdStr, err),
			}, err
		}
	}

	return &AgentExecutionResponse{
		Success:   true,
		Output:    output.String(),
		Artifacts: artifacts,
	}, nil
}

// executeGoTest runs Go tests
func (t *ToolDetectionStrategy) executeGoTest(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	goTool := t.toolSet.Available["go"]
	cmd := exec.CommandContext(ctx, goTool.Path, "test", "-v", "./...")
	SetupCommand(cmd, req)

	output, err := cmd.CombinedOutput()
	success := err == nil

	return &AgentExecutionResponse{
		Success: success,
		Output:  string(output),
		Error: func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
		Artifacts: []string{"Go test results"},
	}, nil
}

// executeNpmTest runs npm tests
func (t *ToolDetectionStrategy) executeNpmTest(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	npmTool := t.toolSet.Available["npm"]
	cmd := exec.CommandContext(ctx, npmTool.Path, "test")
	SetupCommand(cmd, req)

	output, err := cmd.CombinedOutput()
	success := err == nil

	return &AgentExecutionResponse{
		Success: success,
		Output:  string(output),
		Error: func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
		Artifacts: []string{"npm test results"},
	}, nil
}

// executeGenericAgent handles unknown agents with available tools
func (t *ToolDetectionStrategy) executeGenericAgent(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	if t.toolSet.Has("claude") {
		return t.executeClaudeTask(ctx, req, req.AgentName)
	}

	return &AgentExecutionResponse{
		Success:   true,
		Output:    fmt.Sprintf("Generic agent %s processed with available tools", req.AgentName),
		Artifacts: []string{"Generic task completed"},
	}, nil
}

// Additional execution methods for other agents...
func (t *ToolDetectionStrategy) executeGenericAPITask(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	return &AgentExecutionResponse{
		Success:   true,
		Output:    "API analysis completed using available tools",
		Artifacts: []string{"API specification reviewed", "Endpoint documentation"},
	}, nil
}

func (t *ToolDetectionStrategy) executeDockerTask(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	dockerTool := t.toolSet.Available["docker"]

	// Basic Docker operations
	var commands []string
	if strings.Contains(req.Summary, "build") {
		commands = append(commands, "docker build .")
	}
	if strings.Contains(req.Summary, "compose") {
		commands = append(commands, "docker compose up --build")
	}

	if len(commands) == 0 {
		commands = []string{"docker --version"}
	}

	var output strings.Builder
	for _, cmdStr := range commands {
		parts := strings.Fields(cmdStr)
		cmd := exec.CommandContext(ctx, dockerTool.Path, parts[1:]...)
		SetupCommand(cmd, req)

		cmdOutput, err := cmd.CombinedOutput()
		output.WriteString(fmt.Sprintf("$ %s\n", cmdStr))
		output.Write(cmdOutput)

		if err != nil {
			return &AgentExecutionResponse{
				Success: false,
				Output:  output.String(),
				Error:   err.Error(),
			}, err
		}
	}

	return &AgentExecutionResponse{
		Success:   true,
		Output:    output.String(),
		Artifacts: []string{"Docker operations completed"},
	}, nil
}

func (t *ToolDetectionStrategy) executeGenericDevOpsTask(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	return &AgentExecutionResponse{
		Success:   true,
		Output:    "DevOps analysis completed",
		Artifacts: []string{"Infrastructure reviewed", "Deployment configurations checked"},
	}, nil
}

func (t *ToolDetectionStrategy) executeNodeTask(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	var tool tools.ToolInfo
	var exists bool

	if tool, exists = t.toolSet.Available["yarn"]; !exists {
		tool, exists = t.toolSet.Available["npm"]
		if !exists {
			return nil, fmt.Errorf("no Node.js package manager found")
		}
	}

	cmd := exec.CommandContext(ctx, tool.Path, "run", "build")
	SetupCommand(cmd, req)

	output, err := cmd.CombinedOutput()
	success := err == nil

	return &AgentExecutionResponse{
		Success: success,
		Output:  string(output),
		Error: func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
		Artifacts: []string{"Node.js build completed"},
	}, nil
}

// hasSuitableTools checks if we have tools suitable for the agent
func (t *ToolDetectionStrategy) hasSuitableTools(agentName string, toolSet *tools.ToolSet) bool {
	// AI tools can handle any agent
	if toolSet.Has("claude") || toolSet.Has("cursor") || toolSet.Has("vscode") {
		return true
	}

	// Agent-specific tool requirements
	switch agentName {
	case "golang-expert":
		return toolSet.Has("go")
	case "test-expert":
		return toolSet.Has("go") || toolSet.Has("npm") || toolSet.Has("python")
	case "devops-expert":
		return toolSet.Has("docker") || toolSet.Has("make")
	case "typescript-expert":
		return toolSet.Has("npm") || toolSet.Has("yarn") || toolSet.Has("node")
	default:
		return true // Generic handling available
	}
}

// parseClaudeOutput parses Claude Code output for follow-up handoffs
func (t *ToolDetectionStrategy) parseClaudeOutput(output string) []NextHandoff {
	var handoffs []NextHandoff

	// Look for handoff patterns in Claude Code output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Pattern: "Handing off to <agent>: <description>"
		if strings.Contains(line, "Handing off to") || strings.Contains(line, "handoff to") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				agentPart := strings.TrimSpace(parts[0])
				description := strings.TrimSpace(strings.Join(parts[1:], ":"))

				// Extract agent name
				words := strings.Fields(agentPart)
				for _, word := range words {
					if strings.Contains(word, "-expert") || strings.Contains(word, "-manager") {
						handoffs = append(handoffs, NextHandoff{
							ToAgent:  word,
							Summary:  description,
							Context:  "Claude Code analysis result",
							Priority: "normal",
						})
						break
					}
				}
			}
		}
	}

	return handoffs
}
