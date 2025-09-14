package tools

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ToolInfo contains information about an available tool
type ToolInfo struct {
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
	Priority     int      `json:"priority"`
}

// ToolSet contains all detected tools
type ToolSet struct {
	Available map[string]ToolInfo `json:"available"`
	Project   ProjectInfo         `json:"project"`
}

// ProjectInfo contains information about the project context
type ProjectInfo struct {
	Type         string            `json:"type"`         // go, node, python, generic
	Path         string            `json:"path"`
	Dependencies []string          `json:"dependencies"`
	BuildSystem  string            `json:"build_system"` // make, go, npm, etc.
	Features     map[string]string `json:"features"`     // additional project features
}

// DetectAvailableTools scans for available development tools
func DetectAvailableTools(projectPath string) (*ToolSet, error) {
	toolSet := &ToolSet{
		Available: make(map[string]ToolInfo),
		Project:   detectProjectInfo(projectPath),
	}

	// Detect AI coding assistants (highest priority)
	detectAITools(toolSet)

	// Detect language tools
	detectLanguageTools(toolSet)

	// Detect build tools
	detectBuildTools(toolSet)

	// Detect project-specific tools
	detectProjectTools(toolSet, projectPath)

	log.Printf("🔍 Detected %d tools for project type: %s", len(toolSet.Available), toolSet.Project.Type)

	return toolSet, nil
}

// detectAITools looks for AI coding assistants
func detectAITools(toolSet *ToolSet) {
	// Claude Code (highest priority)
	if path, err := exec.LookPath("claude"); err == nil {
		version := getToolVersion("claude", "--version")
		toolSet.Available["claude"] = ToolInfo{
			Name:         "claude",
			Path:         path,
			Version:      version,
			Priority:     100,
			Capabilities: []string{"task", "code-gen", "analysis", "refactor", "test"},
		}
		log.Printf("✅ Found Claude Code at: %s", path)
	}

	// Cursor (high priority)
	if path, err := exec.LookPath("cursor"); err == nil {
		toolSet.Available["cursor"] = ToolInfo{
			Name:         "cursor",
			Path:         path,
			Priority:     90,
			Capabilities: []string{"edit", "generate", "refactor"},
		}
		log.Printf("✅ Found Cursor at: %s", path)
	}

	// VS Code (medium priority)
	if path, err := exec.LookPath("code"); err == nil {
		toolSet.Available["vscode"] = ToolInfo{
			Name:         "vscode",
			Path:         path,
			Priority:     80,
			Capabilities: []string{"edit", "debug", "extension"},
		}
		log.Printf("✅ Found VS Code at: %s", path)
	}
}

// detectLanguageTools looks for programming language tools
func detectLanguageTools(toolSet *ToolSet) {
	// Go tools
	if path, err := exec.LookPath("go"); err == nil {
		version := getToolVersion("go", "version")
		toolSet.Available["go"] = ToolInfo{
			Name:         "go",
			Path:         path,
			Version:      version,
			Priority:     70,
			Capabilities: []string{"build", "test", "mod", "run", "fmt", "vet"},
		}
	}

	// Node.js tools
	if path, err := exec.LookPath("node"); err == nil {
		version := getToolVersion("node", "--version")
		toolSet.Available["node"] = ToolInfo{
			Name:         "node",
			Path:         path,
			Version:      version,
			Priority:     70,
			Capabilities: []string{"run", "debug"},
		}
	}

	if path, err := exec.LookPath("npm"); err == nil {
		version := getToolVersion("npm", "--version")
		toolSet.Available["npm"] = ToolInfo{
			Name:         "npm",
			Path:         path,
			Version:      version,
			Priority:     65,
			Capabilities: []string{"install", "test", "build", "run"},
		}
	}

	// Python tools
	if path, err := exec.LookPath("python3"); err == nil {
		version := getToolVersion("python3", "--version")
		toolSet.Available["python"] = ToolInfo{
			Name:         "python",
			Path:         path,
			Version:      version,
			Priority:     70,
			Capabilities: []string{"run", "test"},
		}
	}

	if path, err := exec.LookPath("pip"); err == nil {
		toolSet.Available["pip"] = ToolInfo{
			Name:         "pip",
			Path:         path,
			Priority:     65,
			Capabilities: []string{"install"},
		}
	}
}

// detectBuildTools looks for build and development tools
func detectBuildTools(toolSet *ToolSet) {
	// Make
	if path, err := exec.LookPath("make"); err == nil {
		toolSet.Available["make"] = ToolInfo{
			Name:         "make",
			Path:         path,
			Priority:     60,
			Capabilities: []string{"build", "test", "install"},
		}
	}

	// Docker
	if path, err := exec.LookPath("docker"); err == nil {
		toolSet.Available["docker"] = ToolInfo{
			Name:         "docker",
			Path:         path,
			Priority:     55,
			Capabilities: []string{"build", "run", "compose"},
		}
	}

	// Git
	if path, err := exec.LookPath("git"); err == nil {
		version := getToolVersion("git", "--version")
		toolSet.Available["git"] = ToolInfo{
			Name:         "git",
			Path:         path,
			Version:      version,
			Priority:     50,
			Capabilities: []string{"commit", "branch", "merge", "push", "pull"},
		}
	}
}

// detectProjectTools looks for project-specific tools
func detectProjectTools(toolSet *ToolSet, projectPath string) {
	if projectPath == "" {
		return
	}

	// Check for project-specific package managers
	if _, err := os.Stat(filepath.Join(projectPath, "yarn.lock")); err == nil {
		if path, err := exec.LookPath("yarn"); err == nil {
			toolSet.Available["yarn"] = ToolInfo{
				Name:         "yarn",
				Path:         path,
				Priority:     67,
				Capabilities: []string{"install", "test", "build", "run"},
			}
		}
	}

	// Check for Rust tools
	if _, err := os.Stat(filepath.Join(projectPath, "Cargo.toml")); err == nil {
		if path, err := exec.LookPath("cargo"); err == nil {
			toolSet.Available["cargo"] = ToolInfo{
				Name:         "cargo",
				Path:         path,
				Priority:     70,
				Capabilities: []string{"build", "test", "run"},
			}
		}
	}
}

// detectProjectInfo analyzes the project to determine its type and characteristics
func detectProjectInfo(projectPath string) ProjectInfo {
	info := ProjectInfo{
		Type:         "generic",
		Path:         projectPath,
		Dependencies: []string{},
		Features:     make(map[string]string),
	}

	if projectPath == "" {
		return info
	}

	// Detect Go project
	if _, err := os.Stat(filepath.Join(projectPath, "go.mod")); err == nil {
		info.Type = "go"
		info.BuildSystem = "go"
		info.Features["modules"] = "enabled"
		
		// Check for specific Go features
		if _, err := os.Stat(filepath.Join(projectPath, "Makefile")); err == nil {
			info.BuildSystem = "make"
		}
	}

	// Detect Node.js project
	if _, err := os.Stat(filepath.Join(projectPath, "package.json")); err == nil {
		info.Type = "node"
		info.BuildSystem = "npm"
		
		if _, err := os.Stat(filepath.Join(projectPath, "yarn.lock")); err == nil {
			info.BuildSystem = "yarn"
		}
	}

	// Detect Python project
	if _, err := os.Stat(filepath.Join(projectPath, "requirements.txt")); err == nil {
		info.Type = "python"
		info.BuildSystem = "pip"
	}
	if _, err := os.Stat(filepath.Join(projectPath, "setup.py")); err == nil {
		info.Type = "python"
		info.BuildSystem = "setup.py"
	}

	// Detect Rust project
	if _, err := os.Stat(filepath.Join(projectPath, "Cargo.toml")); err == nil {
		info.Type = "rust"
		info.BuildSystem = "cargo"
	}

	return info
}

// getToolVersion attempts to get the version of a tool
func getToolVersion(tool, versionFlag string) string {
	cmd := exec.Command(tool, versionFlag)
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	
	version := strings.TrimSpace(string(output))
	// Take only the first line for cleaner output
	if lines := strings.Split(version, "\n"); len(lines) > 0 {
		return lines[0]
	}
	
	return version
}

// Has checks if a tool is available
func (t *ToolSet) Has(toolName string) bool {
	_, exists := t.Available[toolName]
	return exists
}

// Get returns information about a specific tool
func (t *ToolSet) Get(toolName string) (ToolInfo, bool) {
	tool, exists := t.Available[toolName]
	return tool, exists
}

// ListAvailable returns a list of available tool names
func (t *ToolSet) ListAvailable() []string {
	var tools []string
	for name := range t.Available {
		tools = append(tools, name)
	}
	return tools
}

// GetByCapability returns tools that have a specific capability
func (t *ToolSet) GetByCapability(capability string) []ToolInfo {
	var matching []ToolInfo
	
	for _, tool := range t.Available {
		for _, cap := range tool.Capabilities {
			if cap == capability {
				matching = append(matching, tool)
				break
			}
		}
	}
	
	return matching
}

// GetBestToolFor returns the highest priority tool for a specific capability
func (t *ToolSet) GetBestToolFor(capability string) *ToolInfo {
	matching := t.GetByCapability(capability)
	if len(matching) == 0 {
		return nil
	}
	
	best := &matching[0]
	for i := 1; i < len(matching); i++ {
		if matching[i].Priority > best.Priority {
			best = &matching[i]
		}
	}
	
	return best
}

// String returns a human-readable representation of the toolset
func (t *ToolSet) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Project: %s (%s)", t.Project.Type, t.Project.BuildSystem))
	
	if len(t.Available) > 0 {
		tools := t.ListAvailable()
		parts = append(parts, fmt.Sprintf("Tools: %s", strings.Join(tools, ", ")))
	}
	
	return strings.Join(parts, " | ")
}