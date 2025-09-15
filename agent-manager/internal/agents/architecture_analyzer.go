package agents

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ArchitectureAnalyzer analyzes software architecture
type ArchitectureAnalyzer struct {
	projectPath string
}

// ArchitectureAnalysis contains the results of architecture analysis
type ArchitectureAnalysis struct {
	Pattern         string                      `json:"pattern"`          // microservices, monolith, layered, etc.
	Components      []ArchitectureComponent     `json:"components"`
	Dependencies    []ComponentDependency       `json:"dependencies"`
	ComplexityScore float64                     `json:"complexity_score"` // 0-10
	CouplingScore   float64                     `json:"coupling_score"`   // 0-10
	Layers          []ArchitectureLayer         `json:"layers"`
	Interfaces      []InterfaceDefinition       `json:"interfaces"`
	DataFlow        []DataFlowConnection        `json:"data_flow"`
	Recommendations []ArchitectureRecommendation `json:"recommendations"`
}

// ArchitectureComponent represents a component in the architecture
type ArchitectureComponent struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`         // service, package, module, class
	Path         string            `json:"path"`
	Responsibilities []string       `json:"responsibilities"`
	PublicAPI    []string          `json:"public_api"`
	Dependencies []string          `json:"dependencies"`
	Size         int               `json:"size"`         // lines of code or file count
	Complexity   float64           `json:"complexity"`
	Metadata     map[string]string `json:"metadata"`
}

// ComponentDependency represents a dependency between components
type ComponentDependency struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Type       string `json:"type"`       // import, call, data
	Strength   int    `json:"strength"`   // 1-5, how tightly coupled
	Direction  string `json:"direction"`  // bidirectional, unidirectional
}

// ArchitectureLayer represents a layer in the architecture
type ArchitectureLayer struct {
	Name        string   `json:"name"`
	Level       int      `json:"level"`      // 0 = lowest level
	Components  []string `json:"components"`
	Description string   `json:"description"`
}

// InterfaceDefinition represents an interface or contract
type InterfaceDefinition struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`        // API, interface, contract
	Methods     []string `json:"methods"`
	Components  []string `json:"components"`  // components that implement/use this
	Stability   string   `json:"stability"`   // stable, evolving, experimental
}

// DataFlowConnection represents data flow between components
type DataFlowConnection struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Type   string `json:"type"`   // sync, async, event, data
	Data   string `json:"data"`   // type of data flowing
}

// ArchitectureRecommendation represents an architecture improvement recommendation
type ArchitectureRecommendation struct {
	Type        string `json:"type"`        // refactor, decouple, extract, merge
	Priority    string `json:"priority"`    // high, medium, low
	Component   string `json:"component"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Effort      string `json:"effort"`      // low, medium, high
}

// OrchestrationPlan represents a plan for orchestrating multiple agents
type OrchestrationPlan struct {
	Type              string                `json:"type"`
	Agents            []string              `json:"agents"`
	EstimatedDuration string                `json:"estimated_duration"`
	Phases            []OrchestrationPhase  `json:"phases"`
}

// OrchestrationPhase represents a phase in orchestration
type OrchestrationPhase struct {
	Name     string   `json:"name"`
	Agents   []string `json:"agents"`
	Duration string   `json:"duration"`
}

// NewArchitectureAnalyzer creates a new architecture analyzer
func NewArchitectureAnalyzer(projectPath string) *ArchitectureAnalyzer {
	return &ArchitectureAnalyzer{
		projectPath: projectPath,
	}
}

// AnalyzeArchitecture performs comprehensive architecture analysis
func (a *ArchitectureAnalyzer) AnalyzeArchitecture() (*ArchitectureAnalysis, error) {
	if a.projectPath == "" {
		return nil, fmt.Errorf("project path is empty")
	}

	log.Printf("🏗️ Analyzing architecture at: %s", a.projectPath)

	analysis := &ArchitectureAnalysis{
		Components:      []ArchitectureComponent{},
		Dependencies:    []ComponentDependency{},
		Layers:          []ArchitectureLayer{},
		Interfaces:      []InterfaceDefinition{},
		DataFlow:        []DataFlowConnection{},
		Recommendations: []ArchitectureRecommendation{},
	}

	// Check if path exists
	if _, err := os.Stat(a.projectPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("project path does not exist: %s", a.projectPath)
	}

	// Detect architecture pattern
	a.detectArchitecturePattern(analysis)

	// Analyze components
	if err := a.analyzeComponents(analysis); err != nil {
		log.Printf("⚠️ Error analyzing components: %v", err)
	}

	// Analyze dependencies
	if err := a.analyzeDependencies(analysis); err != nil {
		log.Printf("⚠️ Error analyzing dependencies: %v", err)
	}

	// Analyze layers
	if err := a.analyzeLayers(analysis); err != nil {
		log.Printf("⚠️ Error analyzing layers: %v", err)
	}

	// Calculate complexity and coupling scores
	a.calculateScores(analysis)

	// Generate recommendations
	a.generateRecommendations(analysis)

	log.Printf("✅ Architecture analysis completed: %s pattern with %d components", 
		analysis.Pattern, len(analysis.Components))

	return analysis, nil
}

// detectArchitecturePattern detects the overall architecture pattern
func (a *ArchitectureAnalyzer) detectArchitecturePattern(analysis *ArchitectureAnalysis) {
	analysis.Pattern = "unknown"

	// Check for microservices patterns
	if a.hasMicroservicesPattern() {
		analysis.Pattern = "microservices"
		return
	}

	// Check for layered architecture
	if a.hasLayeredPattern() {
		analysis.Pattern = "layered"
		return
	}

	// Check for modular monolith
	if a.hasModularPattern() {
		analysis.Pattern = "modular-monolith"
		return
	}

	// Default to monolith
	analysis.Pattern = "monolith"
}

// analyzeComponents discovers and analyzes architecture components
func (a *ArchitectureAnalyzer) analyzeComponents(analysis *ArchitectureAnalysis) error {
	err := filepath.Walk(a.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		// Skip hidden directories and common ignore patterns
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}

			// Analyze directory as a potential component
			if a.isComponentDirectory(path, name) {
				component := a.analyzeComponentDirectory(path, name)
				if component != nil {
					analysis.Components = append(analysis.Components, *component)
				}
			}
		}

		return nil
	})

	return err
}

// analyzeDependencies analyzes dependencies between components
func (a *ArchitectureAnalyzer) analyzeDependencies(analysis *ArchitectureAnalysis) error {
	// For each component, analyze its dependencies
	for _, component := range analysis.Components {
		deps, err := a.findComponentDependencies(component, analysis.Components)
		if err != nil {
			log.Printf("⚠️ Error finding dependencies for %s: %v", component.Name, err)
			continue
		}
		analysis.Dependencies = append(analysis.Dependencies, deps...)
	}

	return nil
}

// analyzeLayers identifies architectural layers
func (a *ArchitectureAnalyzer) analyzeLayers(analysis *ArchitectureAnalysis) error {
	// Common layer patterns
	layers := a.identifyLayers(analysis.Components)
	analysis.Layers = layers
	return nil
}

// calculateScores calculates complexity and coupling scores
func (a *ArchitectureAnalyzer) calculateScores(analysis *ArchitectureAnalysis) {
	// Calculate complexity score (0-10)
	totalComplexity := 0.0
	for _, component := range analysis.Components {
		totalComplexity += component.Complexity
	}
	
	if len(analysis.Components) > 0 {
		analysis.ComplexityScore = totalComplexity / float64(len(analysis.Components))
	}

	// Calculate coupling score (0-10)
	if len(analysis.Components) > 1 {
		totalConnections := len(analysis.Dependencies)
		maxPossibleConnections := len(analysis.Components) * (len(analysis.Components) - 1)
		if maxPossibleConnections > 0 {
			analysis.CouplingScore = float64(totalConnections) / float64(maxPossibleConnections) * 10
		}
	}
}

// generateRecommendations generates architecture improvement recommendations
func (a *ArchitectureAnalyzer) generateRecommendations(analysis *ArchitectureAnalysis) {
	var recommendations []ArchitectureRecommendation

	// High complexity recommendation
	if analysis.ComplexityScore > 7.0 {
		recommendations = append(recommendations, ArchitectureRecommendation{
			Type:        "refactor",
			Priority:    "high",
			Description: "High complexity detected. Consider breaking down large components into smaller, focused modules",
			Impact:      "Improved maintainability and testability",
			Effort:      "high",
		})
	}

	// High coupling recommendation
	if analysis.CouplingScore > 6.0 {
		recommendations = append(recommendations, ArchitectureRecommendation{
			Type:        "decouple",
			Priority:    "medium",
			Description: "High coupling between components. Consider introducing interfaces and dependency injection",
			Impact:      "Better modularity and flexibility",
			Effort:      "medium",
		})
	}

	// Component-specific recommendations
	for _, component := range analysis.Components {
		if component.Complexity > 8.0 {
			recommendations = append(recommendations, ArchitectureRecommendation{
				Type:        "extract",
				Priority:    "medium",
				Component:   component.Name,
				Description: fmt.Sprintf("Component '%s' is highly complex. Consider extracting responsibilities", component.Name),
				Impact:      "Improved single responsibility and testability",
				Effort:      "medium",
			})
		}

		if len(component.Dependencies) > 10 {
			recommendations = append(recommendations, ArchitectureRecommendation{
				Type:        "decouple",
				Priority:    "medium",
				Component:   component.Name,
				Description: fmt.Sprintf("Component '%s' has many dependencies. Consider reducing coupling", component.Name),
				Impact:      "Better isolation and testability",
				Effort:      "medium",
			})
		}
	}

	analysis.Recommendations = recommendations
}

// Helper methods for pattern detection
func (a *ArchitectureAnalyzer) hasMicroservicesPattern() bool {
	// Look for microservices indicators
	patterns := []string{
		"cmd/*/main.go",  // Multiple main files
		"services/*/",    // Services directory structure
		"docker-compose.yml", // Container orchestration
		"k8s/",          // Kubernetes configs
	}

	count := 0
	for _, pattern := range patterns {
		if a.pathExists(pattern) {
			count++
		}
	}

	return count >= 2
}

func (a *ArchitectureAnalyzer) hasLayeredPattern() bool {
	// Look for layered architecture indicators
	layers := []string{
		"handler", "controller", "api",     // Presentation layer
		"service", "business", "domain",    // Business layer  
		"repository", "dao", "data",       // Data layer
		"model", "entity",                 // Data models
	}

	count := 0
	for _, layer := range layers {
		if a.hasDirectoryWithName(layer) {
			count++
		}
	}

	return count >= 3
}

func (a *ArchitectureAnalyzer) hasModularPattern() bool {
	// Look for modular structure
	return a.hasMultiplePackages() && !a.hasMicroservicesPattern()
}

// Helper methods for component analysis
func (a *ArchitectureAnalyzer) isComponentDirectory(path, name string) bool {
	// Skip root directory
	if path == a.projectPath {
		return false
	}

	// Component indicators
	componentPatterns := []string{
		"internal/", "pkg/", "cmd/", "api/", "service/", "handler/",
		"repository/", "model/", "config/", "middleware/", "util/",
	}

	relPath, _ := filepath.Rel(a.projectPath, path)
	for _, pattern := range componentPatterns {
		if strings.Contains(relPath, pattern) {
			return true
		}
	}

	// Check if directory has Go files (or other source files)
	return a.hasSourceFiles(path)
}

func (a *ArchitectureAnalyzer) analyzeComponentDirectory(path, name string) *ArchitectureComponent {
	relPath, _ := filepath.Rel(a.projectPath, path)
	
	component := &ArchitectureComponent{
		Name:            name,
		Type:            a.determineComponentType(relPath),
		Path:            relPath,
		Responsibilities: a.inferResponsibilities(relPath, name),
		Dependencies:    []string{},
		Metadata:        make(map[string]string),
	}

	// Count files and calculate complexity
	fileCount, complexity := a.calculateComponentMetrics(path)
	component.Size = fileCount
	component.Complexity = complexity

	// Analyze public API
	component.PublicAPI = a.analyzePublicAPI(path)

	return component
}

func (a *ArchitectureAnalyzer) determineComponentType(relPath string) string {
	if strings.Contains(relPath, "cmd/") {
		return "service"
	}
	if strings.Contains(relPath, "api/") || strings.Contains(relPath, "handler/") {
		return "api"
	}
	if strings.Contains(relPath, "service/") || strings.Contains(relPath, "business/") {
		return "service"
	}
	if strings.Contains(relPath, "repository/") || strings.Contains(relPath, "dao/") {
		return "data"
	}
	if strings.Contains(relPath, "model/") || strings.Contains(relPath, "entity/") {
		return "model"
	}
	return "package"
}

func (a *ArchitectureAnalyzer) inferResponsibilities(relPath, name string) []string {
	var responsibilities []string

	// Based on path patterns
	if strings.Contains(relPath, "handler") || strings.Contains(relPath, "controller") {
		responsibilities = append(responsibilities, "HTTP request handling", "Input validation", "Response formatting")
	} else if strings.Contains(relPath, "service") {
		responsibilities = append(responsibilities, "Business logic", "Domain operations")
	} else if strings.Contains(relPath, "repository") {
		responsibilities = append(responsibilities, "Data persistence", "Database operations")
	} else if strings.Contains(relPath, "model") {
		responsibilities = append(responsibilities, "Data structures", "Domain entities")
	} else {
		responsibilities = append(responsibilities, fmt.Sprintf("%s functionality", name))
	}

	return responsibilities
}

func (a *ArchitectureAnalyzer) calculateComponentMetrics(path string) (int, float64) {
	fileCount := 0
	totalLines := 0

	filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if a.isSourceFile(filePath) {
			fileCount++
			lines := a.countLines(filePath)
			totalLines += lines
		}

		return nil
	})

	// Simple complexity calculation
	complexity := float64(totalLines) / 100.0 // Rough heuristic
	if complexity > 10.0 {
		complexity = 10.0
	}

	return fileCount, complexity
}

func (a *ArchitectureAnalyzer) analyzePublicAPI(path string) []string {
	var publicAPI []string
	
	// This would analyze exported functions/methods
	// Simplified for demo
	filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(filePath, ".go") {
			return nil
		}

		// Look for exported functions (starting with capital letter)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "func ") && len(line) > 5 {
				funcName := strings.Fields(line[5:])[0]
				if strings.Contains(funcName, "(") {
					funcName = strings.Split(funcName, "(")[0]
				}
				if len(funcName) > 0 && funcName[0] >= 'A' && funcName[0] <= 'Z' {
					publicAPI = append(publicAPI, funcName)
				}
			}
		}

		return nil
	})

	return publicAPI
}

func (a *ArchitectureAnalyzer) findComponentDependencies(component ArchitectureComponent, allComponents []ArchitectureComponent) ([]ComponentDependency, error) {
	var dependencies []ComponentDependency
	
	// Analyze import statements to find dependencies
	componentPath := filepath.Join(a.projectPath, component.Path)
	
	filepath.Walk(componentPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		imports := a.extractImports(path)
		for _, imp := range imports {
			// Check if import refers to another component in our project
			for _, otherComp := range allComponents {
				if strings.Contains(imp, otherComp.Path) {
					dependencies = append(dependencies, ComponentDependency{
						From:      component.Name,
						To:        otherComp.Name,
						Type:      "import",
						Strength:  3,
						Direction: "unidirectional",
					})
				}
			}
		}

		return nil
	})

	return dependencies, nil
}

func (a *ArchitectureAnalyzer) identifyLayers(components []ArchitectureComponent) []ArchitectureLayer {
	layers := []ArchitectureLayer{
		{Name: "Presentation", Level: 3, Components: []string{}, Description: "HTTP handlers and API endpoints"},
		{Name: "Business", Level: 2, Components: []string{}, Description: "Business logic and domain operations"},
		{Name: "Data", Level: 1, Components: []string{}, Description: "Data access and persistence"},
		{Name: "Foundation", Level: 0, Components: []string{}, Description: "Utilities and common functionality"},
	}

	// Assign components to layers
	for _, component := range components {
		switch component.Type {
		case "api":
			layers[0].Components = append(layers[0].Components, component.Name)
		case "service":
			layers[1].Components = append(layers[1].Components, component.Name)
		case "data":
			layers[2].Components = append(layers[2].Components, component.Name)
		default:
			layers[3].Components = append(layers[3].Components, component.Name)
		}
	}

	return layers
}

// Utility methods
func (a *ArchitectureAnalyzer) pathExists(pattern string) bool {
	fullPath := filepath.Join(a.projectPath, pattern)
	_, err := os.Stat(fullPath)
	return err == nil
}

func (a *ArchitectureAnalyzer) hasDirectoryWithName(name string) bool {
	found := false
	filepath.Walk(a.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && strings.Contains(strings.ToLower(info.Name()), name) {
			found = true
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

func (a *ArchitectureAnalyzer) hasMultiplePackages() bool {
	packageCount := 0
	filepath.Walk(a.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		
		if a.hasSourceFiles(path) {
			packageCount++
		}
		
		return nil
	})
	return packageCount > 3
}

func (a *ArchitectureAnalyzer) hasSourceFiles(dirPath string) bool {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}
	
	for _, file := range files {
		if !file.IsDir() && a.isSourceFile(file.Name()) {
			return true
		}
	}
	return false
}

func (a *ArchitectureAnalyzer) isSourceFile(filename string) bool {
	extensions := []string{".go", ".java", ".py", ".js", ".ts", ".rs", ".cpp", ".c"}
	for _, ext := range extensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}

func (a *ArchitectureAnalyzer) countLines(filePath string) int {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}
	return strings.Count(string(content), "\n")
}

func (a *ArchitectureAnalyzer) extractImports(filePath string) []string {
	var imports []string
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		return imports
	}

	lines := strings.Split(string(content), "\n")
	inImportBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "import (") {
			inImportBlock = true
			continue
		}
		
		if inImportBlock && line == ")" {
			inImportBlock = false
			continue
		}

		if strings.HasPrefix(line, "import ") || inImportBlock {
			// Extract import path
			if strings.Contains(line, "\"") {
				start := strings.Index(line, "\"")
				end := strings.LastIndex(line, "\"")
				if start != end && start >= 0 && end > start {
					importPath := line[start+1 : end]
					imports = append(imports, importPath)
				}
			}
		}
	}

	return imports
}