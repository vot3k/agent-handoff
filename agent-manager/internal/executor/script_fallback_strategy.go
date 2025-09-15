package executor

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/vot3k/agent-handoff/agent-manager/internal/tools"
)

// ScriptFallbackStrategy executes agents using traditional run-agent.sh scripts
type ScriptFallbackStrategy struct {
	scriptPath string
}

// NewScriptFallbackStrategy creates a new script fallback strategy
func NewScriptFallbackStrategy() *ScriptFallbackStrategy {
	return &ScriptFallbackStrategy{}
}

func (s *ScriptFallbackStrategy) Name() string {
	return "ScriptFallback"
}

func (s *ScriptFallbackStrategy) Priority() int {
	return 50 // Lowest priority - fallback option
}

func (s *ScriptFallbackStrategy) CanHandle(agentName string, projectPath string, toolSet *tools.ToolSet) bool {
	// Try to find run-agent.sh script
	scriptPath, err := s.findRunAgentScript(projectPath)
	if err != nil {
		log.Printf("⚠️ ScriptFallbackStrategy: %v", err)
		return false
	}

	s.scriptPath = scriptPath
	log.Printf("📝 ScriptFallbackStrategy: Found script at %s", scriptPath)
	return true
}

func (s *ScriptFallbackStrategy) Execute(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	log.Printf("📝 ScriptFallbackStrategy executing %s with script: %s", req.AgentName, s.scriptPath)

	if s.scriptPath == "" {
		scriptPath, err := s.findRunAgentScript(req.ProjectPath)
		if err != nil {
			return nil, fmt.Errorf("no run-agent.sh script found: %w", err)
		}
		s.scriptPath = scriptPath
	}

	// Execute the script
	cmd := exec.CommandContext(ctx, s.scriptPath, req.AgentName, req.Payload)

	// Set working directory and environment
	scriptDir := filepath.Dir(s.scriptPath)
	cmd.Dir = scriptDir

	env := os.Environ()
	for key, value := range req.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Env = env

	log.Printf("🔍 Executing: %s %s [payload] from directory: %s", s.scriptPath, req.AgentName, scriptDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &AgentExecutionResponse{
			Success: false,
			Output:  string(output),
			Error:   fmt.Sprintf("Script execution failed: %v", err),
		}, err
	}

	log.Printf("✅ Script execution completed successfully")

	return &AgentExecutionResponse{
		Success:   true,
		Output:    string(output),
		Artifacts: []string{"Script execution completed", "Legacy agent workflow processed"},
		Metadata: map[string]string{
			"script_path": s.scriptPath,
			"script_dir":  scriptDir,
		},
	}, nil
}

// findRunAgentScript locates the run-agent.sh script using multiple strategies
func (s *ScriptFallbackStrategy) findRunAgentScript(projectPath string) (string, error) {
	// Strategy 1: Check environment variable first
	if scriptPath := os.Getenv("RUN_AGENT_SCRIPT_PATH"); scriptPath != "" {
		if _, err := os.Stat(scriptPath); err == nil {
			log.Printf("📝 Using run-agent.sh from environment variable: %s", scriptPath)
			return scriptPath, nil
		}
		log.Printf("⚠️ RUN_AGENT_SCRIPT_PATH points to non-existent file: %s", scriptPath)
	}

	// Strategy 2: Check project-specific script
	if projectPath != "" {
		projectScript := filepath.Join(projectPath, "run-agent.sh")
		if _, err := os.Stat(projectScript); err == nil {
			absPath, _ := filepath.Abs(projectScript)
			log.Printf("📝 Found project-specific run-agent.sh: %s", absPath)
			return absPath, nil
		}
	}

	// Strategy 3: Find relative to executable location
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)

		// Try same directory as executable
		scriptPath := filepath.Join(execDir, "run-agent.sh")
		if _, err := os.Stat(scriptPath); err == nil {
			log.Printf("📝 Found run-agent.sh relative to executable: %s", scriptPath)
			return scriptPath, nil
		}

		// Try agent-manager directory (if executable is in a subdirectory)
		agentManagerDir := filepath.Join(execDir, "..", "agent-manager")
		scriptPath = filepath.Join(agentManagerDir, "run-agent.sh")
		if _, err := os.Stat(scriptPath); err == nil {
			absPath, _ := filepath.Abs(scriptPath)
			log.Printf("📝 Found run-agent.sh in agent-manager directory: %s", absPath)
			return absPath, nil
		}
	}

	// Strategy 4: Search common locations
	searchPaths := []string{
		"./run-agent.sh",                // Current directory
		"./agent-manager/run-agent.sh",  // Agent-manager subdirectory
		"../agent-manager/run-agent.sh", // Parent then agent-manager
		"/Users/jimmy/Dev/ai-platforms/agent-handoff/agent-manager/run-agent.sh", // Absolute fallback
	}

	for _, path := range searchPaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				log.Printf("📝 Found run-agent.sh at: %s", absPath)
				return absPath, nil
			}
		}
	}

	return "", fmt.Errorf("run-agent.sh script not found in any expected location")
}

// GetScriptPath returns the current script path (for debugging/info purposes)
func (s *ScriptFallbackStrategy) GetScriptPath() string {
	return s.scriptPath
}

// ValidateScript checks if the script exists and is executable
func (s *ScriptFallbackStrategy) ValidateScript() error {
	if s.scriptPath == "" {
		return fmt.Errorf("no script path set")
	}

	// Check if file exists
	info, err := os.Stat(s.scriptPath)
	if err != nil {
		return fmt.Errorf("script not accessible: %w", err)
	}

	// Check if file is executable
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("script is not executable: %s", s.scriptPath)
	}

	// Try to execute with help flag to validate
	cmd := exec.Command(s.scriptPath)
	err = cmd.Start()
	if err == nil {
		cmd.Process.Kill() // Kill immediately since we're just testing
		return nil
	}

	return fmt.Errorf("script validation failed: %w", err)
}

// createTemporaryScript creates a basic fallback script if none exists
func (s *ScriptFallbackStrategy) createTemporaryScript() (string, error) {
	tempFile, err := os.CreateTemp("", "run-agent-fallback-*.sh")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary script: %w", err)
	}
	defer tempFile.Close()

	scriptContent := `#!/bin/bash
# Temporary fallback run-agent.sh script
# Generated by ScriptFallbackStrategy

set -e

AGENT_NAME="$1"
PAYLOAD="$2"

if [ -z "$AGENT_NAME" ] || [ -z "$PAYLOAD" ]; then
  echo "Usage: $0 <agent_name> <json_payload>"
  echo "ERROR: Missing required arguments"
  exit 1
fi

echo "=== Temporary Agent Executor: Running agent '$AGENT_NAME' ==="
echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "Project Context: $AGENT_PROJECT_NAME"
echo "Agent: $AGENT_NAME"

# Parse handoff ID from payload
HANDOFF_ID=$(echo "$PAYLOAD" | jq -r '.handoff_id // .metadata.handoff_id // "unknown"' 2>/dev/null || echo "unknown")
echo "Handoff ID: $HANDOFF_ID"

echo ""
echo "⚠️  Using temporary fallback script - consider implementing project-specific run-agent.sh"
echo ""

# Simple agent execution simulation
case "$AGENT_NAME" in
    "project-manager")
        echo "📋 Project Manager: Analyzing project structure..."
        sleep 1
        echo "📋 Project Manager: Generating insights and recommendations..."
        echo "✅ Project Manager: Analysis completed"
        ;;
    "golang-expert")
        echo "🐹 Go Expert: Reviewing Go code..."
        sleep 1
        echo "🐹 Go Expert: Applying Go best practices..."
        echo "✅ Go Expert: Go improvements completed"
        ;;
    "test-expert")
        echo "🧪 Test Expert: Analyzing test coverage..."
        sleep 1
        echo "🧪 Test Expert: Generating test recommendations..."
        echo "✅ Test Expert: Test analysis completed"
        ;;
    *)
        echo "🤖 Generic Agent: Processing handoff for '$AGENT_NAME'..."
        sleep 1
        echo "🤖 Generic Agent: Task completed"
        echo "✅ Generic Agent: Handoff processed"
        ;;
esac

echo ""
echo "--- Temporary Agent Processing Completed ---"
echo "Consider implementing a proper run-agent.sh script for better functionality"

exit 0
`

	if _, err := tempFile.WriteString(scriptContent); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write script content: %w", err)
	}

	// Make script executable
	if err := os.Chmod(tempFile.Name(), 0755); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to make script executable: %w", err)
	}

	log.Printf("📝 Created temporary fallback script: %s", tempFile.Name())
	return tempFile.Name(), nil
}

// GetAvailableScripts returns a list of potential script locations
func (s *ScriptFallbackStrategy) GetAvailableScripts() []string {
	var scripts []string

	searchPaths := []string{
		"./run-agent.sh",
		"./agent-manager/run-agent.sh",
		"../agent-manager/run-agent.sh",
		"/Users/jimmy/Dev/ai-platforms/agent-handoff/agent-manager/run-agent.sh",
	}

	// Add environment variable if set
	if envPath := os.Getenv("RUN_AGENT_SCRIPT_PATH"); envPath != "" {
		searchPaths = append([]string{envPath}, searchPaths...)
	}

	// Add executable-relative paths
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		searchPaths = append(searchPaths,
			filepath.Join(execDir, "run-agent.sh"),
			filepath.Join(execDir, "..", "agent-manager", "run-agent.sh"))
	}

	for _, path := range searchPaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				scripts = append(scripts, absPath)
			}
		}
	}

	return scripts
}

// String returns a human-readable representation of the strategy
func (s *ScriptFallbackStrategy) String() string {
	if s.scriptPath != "" {
		return fmt.Sprintf("ScriptFallback(script=%s)", s.scriptPath)
	}
	return "ScriptFallback(no script found)"
}
