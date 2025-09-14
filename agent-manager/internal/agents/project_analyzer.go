package agents

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ProjectAnalyzer analyzes projects to provide management insights
type ProjectAnalyzer struct {
	projectPath string
}

// ProjectAnalysis contains the results of project analysis
type ProjectAnalysis struct {
	ProjectType       string            `json:"project_type"`
	ProjectPath       string            `json:"project_path"`
	FileCount         int               `json:"file_count"`
	Languages         []string          `json:"languages"`
	TestCoverage      float64           `json:"test_coverage"`
	HasReadme         bool              `json:"has_readme"`
	HasCICD           bool              `json:"has_cicd"`
	HasGoMod          bool              `json:"has_go_mod"`
	HasPackageJSON    bool              `json:"has_package_json"`
	HasRequirements   bool              `json:"has_requirements"`
	SecurityIssues    int               `json:"security_issues"`
	Dependencies      []Dependency      `json:"dependencies"`
	TestFiles         []string          `json:"test_files"`
	DocumentationSize int64             `json:"documentation_size"`
	LastModified      time.Time         `json:"last_modified"`
	BuildSystem       string            `json:"build_system"`
	Features          map[string]string `json:"features"`
}

// Dependency represents a project dependency
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"` // "direct", "indirect", "dev"
}

// ProjectInsight represents an insight about the project
type ProjectInsight struct {
	Type        string `json:"type"`        // "positive", "issue", "improvement"
	Category    string `json:"category"`    // "testing", "security", "performance", etc.
	Description string `json:"description"`
	Impact      string `json:"impact"`      // "low", "medium", "high"
	Suggestion  string `json:"suggestion,omitempty"`
}

// NewProjectAnalyzer creates a new project analyzer
func NewProjectAnalyzer(projectPath string) *ProjectAnalyzer {
	return &ProjectAnalyzer{
		projectPath: projectPath,
	}
}

// AnalyzeProject performs comprehensive project analysis
func (p *ProjectAnalyzer) AnalyzeProject() (*ProjectAnalysis, error) {
	if p.projectPath == "" {
		return nil, fmt.Errorf("project path is empty")
	}

	log.Printf("🔍 Analyzing project at: %s", p.projectPath)

	analysis := &ProjectAnalysis{
		ProjectPath:  p.projectPath,
		Languages:    []string{},
		Dependencies: []Dependency{},
		TestFiles:    []string{},
		Features:     make(map[string]string),
	}

	// Check if path exists
	if _, err := os.Stat(p.projectPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("project path does not exist: %s", p.projectPath)
	}

	// Analyze project structure
	if err := p.analyzeProjectStructure(analysis); err != nil {
		log.Printf("⚠️ Error analyzing project structure: %v", err)
	}

	// Detect project type
	p.detectProjectType(analysis)

	// Analyze files
	if err := p.analyzeFiles(analysis); err != nil {
		log.Printf("⚠️ Error analyzing files: %v", err)
	}

	// Analyze dependencies
	if err := p.analyzeDependencies(analysis); err != nil {
		log.Printf("⚠️ Error analyzing dependencies: %v", err)
	}

	// Analyze tests
	if err := p.analyzeTests(analysis); err != nil {
		log.Printf("⚠️ Error analyzing tests: %v", err)
	}

	// Security analysis
	if err := p.analyzeSecurityIssues(analysis); err != nil {
		log.Printf("⚠️ Error analyzing security: %v", err)
	}

	// Detect features
	p.detectFeatures(analysis)

	log.Printf("✅ Project analysis completed: %s project with %d files", analysis.ProjectType, analysis.FileCount)

	return analysis, nil
}

// analyzeProjectStructure analyzes the basic project structure
func (p *ProjectAnalyzer) analyzeProjectStructure(analysis *ProjectAnalysis) error {
	// Check for common files
	commonFiles := map[string]*bool{
		"README.md":         &analysis.HasReadme,
		"readme.md":         &analysis.HasReadme,
		"README.txt":        &analysis.HasReadme,
		"go.mod":            &analysis.HasGoMod,
		"package.json":      &analysis.HasPackageJSON,
		"requirements.txt":  &analysis.HasRequirements,
	}

	for file, flag := range commonFiles {
		if _, err := os.Stat(filepath.Join(p.projectPath, file)); err == nil {
			*flag = true
		}
	}

	// Check for CI/CD configurations
	cicdFiles := []string{
		".github/workflows",
		".gitlab-ci.yml",
		"Jenkinsfile",
		".travis.yml",
		"azure-pipelines.yml",
		".circleci",
	}

	for _, cicdFile := range cicdFiles {
		if _, err := os.Stat(filepath.Join(p.projectPath, cicdFile)); err == nil {
			analysis.HasCICD = true
			break
		}
	}

	return nil
}

// detectProjectType determines the project type based on files present
func (p *ProjectAnalyzer) detectProjectType(analysis *ProjectAnalysis) {
	analysis.ProjectType = "generic" // default

	// Go project detection
	if analysis.HasGoMod {
		analysis.ProjectType = "go"
		analysis.BuildSystem = "go"
		if _, err := os.Stat(filepath.Join(p.projectPath, "Makefile")); err == nil {
			analysis.BuildSystem = "make"
		}
		return
	}

	// Node.js project detection
	if analysis.HasPackageJSON {
		analysis.ProjectType = "node"
		analysis.BuildSystem = "npm"
		if _, err := os.Stat(filepath.Join(p.projectPath, "yarn.lock")); err == nil {
			analysis.BuildSystem = "yarn"
		}
		return
	}

	// Python project detection
	if analysis.HasRequirements {
		analysis.ProjectType = "python"
		analysis.BuildSystem = "pip"
		if _, err := os.Stat(filepath.Join(p.projectPath, "setup.py")); err == nil {
			analysis.BuildSystem = "setup.py"
		}
		return
	}

	// Rust project detection
	if _, err := os.Stat(filepath.Join(p.projectPath, "Cargo.toml")); err == nil {
		analysis.ProjectType = "rust"
		analysis.BuildSystem = "cargo"
		return
	}

	// Java project detection
	if _, err := os.Stat(filepath.Join(p.projectPath, "pom.xml")); err == nil {
		analysis.ProjectType = "java"
		analysis.BuildSystem = "maven"
		return
	}

	if _, err := os.Stat(filepath.Join(p.projectPath, "build.gradle")); err == nil {
		analysis.ProjectType = "java"
		analysis.BuildSystem = "gradle"
		return
	}
}

// analyzeFiles analyzes all files in the project
func (p *ProjectAnalyzer) analyzeFiles(analysis *ProjectAnalysis) error {
	var totalSize int64
	languageFiles := make(map[string]int)
	var latestModTime time.Time

	err := filepath.WalkDir(p.projectPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors and continue
		}

		// Skip hidden directories and common ignore patterns
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}

		// Count files and analyze
		analysis.FileCount++

		// Get file info
		info, err := d.Info()
		if err != nil {
			return nil // Continue on error
		}

		totalSize += info.Size()

		// Track latest modification time
		if info.ModTime().After(latestModTime) {
			latestModTime = info.ModTime()
		}

		// Detect language by extension
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".go":
			languageFiles["go"]++
		case ".js", ".jsx":
			languageFiles["javascript"]++
		case ".ts", ".tsx":
			languageFiles["typescript"]++
		case ".py":
			languageFiles["python"]++
		case ".java":
			languageFiles["java"]++
		case ".rs":
			languageFiles["rust"]++
		case ".cpp", ".cc", ".cxx":
			languageFiles["cpp"]++
		case ".c":
			languageFiles["c"]++
		case ".cs":
			languageFiles["csharp"]++
		case ".rb":
			languageFiles["ruby"]++
		case ".php":
			languageFiles["php"]++
		case ".sh", ".bash":
			languageFiles["shell"]++
		case ".yml", ".yaml":
			languageFiles["yaml"]++
		case ".json":
			languageFiles["json"]++
		case ".md":
			analysis.DocumentationSize += info.Size()
		}

		// Identify test files
		if p.isTestFile(path) {
			analysis.TestFiles = append(analysis.TestFiles, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking project directory: %w", err)
	}

	// Extract primary languages (those with more than 5 files or >10% of files)
	threshold := max(5, analysis.FileCount/10)
	for lang, count := range languageFiles {
		if count >= threshold {
			analysis.Languages = append(analysis.Languages, lang)
		}
	}

	// If no primary language found, add the most common one
	if len(analysis.Languages) == 0 && len(languageFiles) > 0 {
		var maxLang string
		var maxCount int
		for lang, count := range languageFiles {
			if count > maxCount {
				maxLang = lang
				maxCount = count
			}
		}
		if maxLang != "" {
			analysis.Languages = append(analysis.Languages, maxLang)
		}
	}

	analysis.LastModified = latestModTime

	return nil
}

// analyzeDependencies analyzes project dependencies
func (p *ProjectAnalyzer) analyzeDependencies(analysis *ProjectAnalysis) error {
	switch analysis.ProjectType {
	case "go":
		return p.analyzeGoDependencies(analysis)
	case "node":
		return p.analyzeNodeDependencies(analysis)
	case "python":
		return p.analyzePythonDependencies(analysis)
	}
	return nil
}

// analyzeGoDependencies analyzes Go module dependencies
func (p *ProjectAnalyzer) analyzeGoDependencies(analysis *ProjectAnalysis) error {
	goModPath := filepath.Join(p.projectPath, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return nil // Not an error, just no go.mod
	}

	lines := strings.Split(string(content), "\n")
	inRequireBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}
		
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		if strings.HasPrefix(line, "require ") || inRequireBlock {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[0]
				if inRequireBlock {
					name = strings.TrimSpace(parts[0])
				} else {
					name = parts[1] // Skip "require"
				}
				
				version := "latest"
				if len(parts) > 1 {
					versionPart := parts[len(parts)-1]
					if strings.HasPrefix(versionPart, "v") {
						version = versionPart
					}
				}

				analysis.Dependencies = append(analysis.Dependencies, Dependency{
					Name:    name,
					Version: version,
					Type:    "direct",
				})
			}
		}
	}

	return nil
}

// analyzeNodeDependencies analyzes npm/yarn dependencies
func (p *ProjectAnalyzer) analyzeNodeDependencies(analysis *ProjectAnalysis) error {
	// This would parse package.json - simplified for now
	return nil
}

// analyzePythonDependencies analyzes Python dependencies
func (p *ProjectAnalyzer) analyzePythonDependencies(analysis *ProjectAnalysis) error {
	// This would parse requirements.txt - simplified for now
	return nil
}

// analyzeTests analyzes test files and coverage
func (p *ProjectAnalyzer) analyzeTests(analysis *ProjectAnalysis) error {
	if len(analysis.TestFiles) == 0 {
		analysis.TestCoverage = 0.0
		return nil
	}

	// Simple test coverage estimation based on file ratio
	// In a real implementation, you'd run actual coverage tools
	testFileCount := len(analysis.TestFiles)
	sourceFileCount := analysis.FileCount - testFileCount

	if sourceFileCount > 0 {
		// Basic heuristic: assume each test file covers ~3 source files adequately
		coverage := float64(testFileCount*3) / float64(sourceFileCount) * 100
		if coverage > 100 {
			coverage = 100
		}
		analysis.TestCoverage = coverage
	}

	return nil
}

// analyzeSecurityIssues performs basic security analysis
func (p *ProjectAnalyzer) analyzeSecurityIssues(analysis *ProjectAnalysis) error {
	// Basic security issue detection
	securityPatterns := []string{
		"password",
		"secret",
		"api_key",
		"private_key",
		"token",
	}

	analysis.SecurityIssues = 0

	err := filepath.WalkDir(p.projectPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		// Skip binary files and large files
		if info, err := d.Info(); err == nil && info.Size() > 1024*1024 {
			return nil
		}

		// Check file content for security patterns
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		contentLower := strings.ToLower(string(content))
		for _, pattern := range securityPatterns {
			if strings.Contains(contentLower, pattern) {
				analysis.SecurityIssues++
				break // Count once per file
			}
		}

		return nil
	})

	return err
}

// detectFeatures detects additional project features
func (p *ProjectAnalyzer) detectFeatures(analysis *ProjectAnalysis) {
	// Check for Docker
	if _, err := os.Stat(filepath.Join(p.projectPath, "Dockerfile")); err == nil {
		analysis.Features["docker"] = "present"
	}

	// Check for Docker Compose
	composeFiles := []string{"docker-compose.yml", "docker-compose.yaml"}
	for _, file := range composeFiles {
		if _, err := os.Stat(filepath.Join(p.projectPath, file)); err == nil {
			analysis.Features["docker-compose"] = "present"
			break
		}
	}

	// Check for Kubernetes
	if _, err := os.Stat(filepath.Join(p.projectPath, "k8s")); err == nil {
		analysis.Features["kubernetes"] = "present"
	}

	// Check for databases
	if analysis.ProjectType == "go" {
		// Look for common Go database drivers in dependencies
		for _, dep := range analysis.Dependencies {
			if strings.Contains(dep.Name, "postgres") || strings.Contains(dep.Name, "pq") {
				analysis.Features["database"] = "postgresql"
			} else if strings.Contains(dep.Name, "mysql") {
				analysis.Features["database"] = "mysql"
			} else if strings.Contains(dep.Name, "redis") {
				analysis.Features["cache"] = "redis"
			}
		}
	}
}

// isTestFile determines if a file is a test file
func (p *ProjectAnalyzer) isTestFile(filePath string) bool {
	fileName := strings.ToLower(filepath.Base(filePath))
	
	// Go test files
	if strings.HasSuffix(fileName, "_test.go") {
		return true
	}

	// JavaScript test files
	testPatterns := []string{
		"test.js", "test.ts", "test.jsx", "test.tsx",
		"spec.js", "spec.ts", "spec.jsx", "spec.tsx",
	}

	for _, pattern := range testPatterns {
		if strings.Contains(fileName, pattern) {
			return true
		}
	}

	// Python test files
	if strings.HasPrefix(fileName, "test_") && strings.HasSuffix(fileName, ".py") {
		return true
	}

	// Check if file is in a test directory
	pathLower := strings.ToLower(filePath)
	testDirs := []string{"/test/", "/tests/", "/__tests__/", "/spec/"}
	for _, testDir := range testDirs {
		if strings.Contains(pathLower, testDir) {
			return true
		}
	}

	return false
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}