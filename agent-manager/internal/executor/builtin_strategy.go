package executor

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/vot3k/agent-handoff/agent-manager/internal/agents"
	"github.com/vot3k/agent-handoff/agent-manager/internal/tools"
)

// BuiltInAgentStrategy implements agents with native Go logic
type BuiltInAgentStrategy struct{}

// NewBuiltInAgentStrategy creates a new built-in agent strategy
func NewBuiltInAgentStrategy() *BuiltInAgentStrategy {
	return &BuiltInAgentStrategy{}
}

func (b *BuiltInAgentStrategy) Name() string {
	return "BuiltInAgent"
}

func (b *BuiltInAgentStrategy) Priority() int {
	return 80 // High priority, but lower than tool detection
}

func (b *BuiltInAgentStrategy) CanHandle(agentName string, projectPath string, toolSet *tools.ToolSet) bool {
	// Built-in agents we can handle natively
	builtInAgents := []string{
		"project-manager",
		"architecture-analyzer",
		"architecture-expert",
		"agent-manager",
	}

	for _, agent := range builtInAgents {
		if agent == agentName {
			return true
		}
	}

	return false
}

func (b *BuiltInAgentStrategy) Execute(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	log.Printf("🔧 BuiltInAgentStrategy executing %s", req.AgentName)

	switch req.AgentName {
	case "project-manager":
		return b.executeProjectManager(ctx, req)
	case "architecture-analyzer", "architecture-expert":
		return b.executeArchitectureAnalyzer(ctx, req)
	case "agent-manager":
		return b.executeAgentManager(ctx, req)
	default:
		return nil, fmt.Errorf("unknown built-in agent: %s", req.AgentName)
	}
}

// executeProjectManager implements native project management logic
func (b *BuiltInAgentStrategy) executeProjectManager(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	log.Printf("📋 Executing built-in Project Manager")

	// Create project analyzer
	analyzer := agents.NewProjectAnalyzer(req.ProjectPath)

	// Analyze the project
	analysis, err := analyzer.AnalyzeProject()
	if err != nil {
		return &AgentExecutionResponse{
			Success: false,
			Error:   fmt.Sprintf("Project analysis failed: %v", err),
		}, err
	}

	// Generate insights and recommendations
	insights := b.generateProjectInsights(analysis, req)

	// Create follow-up handoffs based on analysis
	nextHandoffs := b.createProjectHandoffs(insights, analysis)

	output := b.formatProjectManagerOutput(analysis, insights)

	return &AgentExecutionResponse{
		Success:      true,
		Output:       output,
		Artifacts:    b.generateProjectArtifacts(analysis),
		NextHandoffs: nextHandoffs,
		Metadata: map[string]string{
			"project_type":   analysis.ProjectType,
			"total_files":    fmt.Sprintf("%d", analysis.FileCount),
			"languages":      strings.Join(analysis.Languages, ", "),
			"insights_count": fmt.Sprintf("%d", len(insights)),
		},
	}, nil
}

// executeArchitectureAnalyzer implements native architecture analysis
func (b *BuiltInAgentStrategy) executeArchitectureAnalyzer(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	log.Printf("🏗️ Executing built-in Architecture Analyzer")

	analyzer := agents.NewArchitectureAnalyzer(req.ProjectPath)

	analysis, err := analyzer.AnalyzeArchitecture()
	if err != nil {
		return &AgentExecutionResponse{
			Success: false,
			Error:   fmt.Sprintf("Architecture analysis failed: %v", err),
		}, err
	}

	recommendations := b.generateArchitectureRecommendations(analysis, req)
	nextHandoffs := b.createArchitectureHandoffs(analysis, recommendations)

	output := b.formatArchitectureOutput(analysis, recommendations)

	return &AgentExecutionResponse{
		Success:      true,
		Output:       output,
		Artifacts:    b.generateArchitectureArtifacts(analysis),
		NextHandoffs: nextHandoffs,
		Metadata: map[string]string{
			"architecture_pattern": analysis.Pattern,
			"component_count":      fmt.Sprintf("%d", len(analysis.Components)),
			"complexity_score":     fmt.Sprintf("%.2f", analysis.ComplexityScore),
		},
	}, nil
}

// executeAgentManager implements agent orchestration logic
func (b *BuiltInAgentStrategy) executeAgentManager(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionResponse, error) {
	log.Printf("🤖 Executing built-in Agent Manager")

	// Parse the request to understand what orchestration is needed
	orchestrationPlan := b.createOrchestrationPlan(req)

	// Generate next handoffs based on the plan
	nextHandoffs := b.createOrchestrationHandoffs(orchestrationPlan)

	output := b.formatAgentManagerOutput(orchestrationPlan)

	return &AgentExecutionResponse{
		Success:      true,
		Output:       output,
		Artifacts:    []string{"Orchestration plan created", "Agent sequence defined"},
		NextHandoffs: nextHandoffs,
		Metadata: map[string]string{
			"orchestration_type": orchestrationPlan.Type,
			"agents_count":       fmt.Sprintf("%d", len(orchestrationPlan.Agents)),
			"estimated_duration": orchestrationPlan.EstimatedDuration,
		},
	}, nil
}

// Helper methods for project manager
func (b *BuiltInAgentStrategy) generateProjectInsights(analysis *agents.ProjectAnalysis, req AgentExecutionRequest) []agents.ProjectInsight {
	var insights []agents.ProjectInsight

	// Analyze based on project type
	switch analysis.ProjectType {
	case "go":
		insights = append(insights, b.generateGoInsights(analysis)...)
	case "node":
		insights = append(insights, b.generateNodeInsights(analysis)...)
	case "python":
		insights = append(insights, b.generatePythonInsights(analysis)...)
	}

	// General insights
	insights = append(insights, b.generateGeneralInsights(analysis, req)...)

	return insights
}

func (b *BuiltInAgentStrategy) generateGoInsights(analysis *agents.ProjectAnalysis) []agents.ProjectInsight {
	var insights []agents.ProjectInsight

	// Check Go module structure
	if analysis.HasGoMod {
		insights = append(insights, agents.ProjectInsight{
			Type:        "positive",
			Category:    "go-modules",
			Description: "Project uses Go modules for dependency management",
			Impact:      "medium",
		})
	} else {
		insights = append(insights, agents.ProjectInsight{
			Type:        "improvement",
			Category:    "go-modules",
			Description: "Consider migrating to Go modules for better dependency management",
			Impact:      "high",
		})
	}

	// Check test coverage
	if analysis.TestCoverage < 50 {
		insights = append(insights, agents.ProjectInsight{
			Type:        "issue",
			Category:    "testing",
			Description: fmt.Sprintf("Low test coverage: %.1f%%. Consider improving test coverage", analysis.TestCoverage),
			Impact:      "high",
		})
	} else if analysis.TestCoverage > 80 {
		insights = append(insights, agents.ProjectInsight{
			Type:        "positive",
			Category:    "testing",
			Description: fmt.Sprintf("Excellent test coverage: %.1f%%", analysis.TestCoverage),
			Impact:      "low",
		})
	}

	return insights
}

func (b *BuiltInAgentStrategy) generateNodeInsights(analysis *agents.ProjectAnalysis) []agents.ProjectInsight {
	var insights []agents.ProjectInsight

	if analysis.HasPackageJSON {
		insights = append(insights, agents.ProjectInsight{
			Type:        "positive",
			Category:    "package-management",
			Description: "Project has proper package.json configuration",
			Impact:      "medium",
		})
	}

	return insights
}

func (b *BuiltInAgentStrategy) generatePythonInsights(analysis *agents.ProjectAnalysis) []agents.ProjectInsight {
	var insights []agents.ProjectInsight

	if analysis.HasRequirements {
		insights = append(insights, agents.ProjectInsight{
			Type:        "positive",
			Category:    "dependencies",
			Description: "Project has requirements.txt for dependency management",
			Impact:      "medium",
		})
	}

	return insights
}

func (b *BuiltInAgentStrategy) generateGeneralInsights(analysis *agents.ProjectAnalysis, req AgentExecutionRequest) []agents.ProjectInsight {
	var insights []agents.ProjectInsight

	// Check documentation
	if !analysis.HasReadme {
		insights = append(insights, agents.ProjectInsight{
			Type:        "issue",
			Category:    "documentation",
			Description: "Missing README.md file. Consider adding project documentation",
			Impact:      "medium",
		})
	}

	// Check for CI/CD
	if !analysis.HasCICD {
		insights = append(insights, agents.ProjectInsight{
			Type:        "improvement",
			Category:    "automation",
			Description: "No CI/CD configuration found. Consider adding automated workflows",
			Impact:      "medium",
		})
	}

	// Security analysis
	if analysis.SecurityIssues > 0 {
		insights = append(insights, agents.ProjectInsight{
			Type:        "issue",
			Category:    "security",
			Description: fmt.Sprintf("Found %d potential security issues", analysis.SecurityIssues),
			Impact:      "high",
		})
	}

	return insights
}

func (b *BuiltInAgentStrategy) createProjectHandoffs(insights []agents.ProjectInsight, analysis *agents.ProjectAnalysis) []NextHandoff {
	var handoffs []NextHandoff

	// Create handoffs based on insights
	for _, insight := range insights {
		if insight.Impact == "high" {
			switch insight.Category {
			case "testing":
				handoffs = append(handoffs, NextHandoff{
					ToAgent:  "test-expert",
					Summary:  fmt.Sprintf("Improve test coverage: %s", insight.Description),
					Context:  "Project analysis identified testing issues",
					Priority: "high",
				})
			case "security":
				handoffs = append(handoffs, NextHandoff{
					ToAgent:  "security-expert",
					Summary:  fmt.Sprintf("Address security issues: %s", insight.Description),
					Context:  "Project analysis found security concerns",
					Priority: "high",
				})
			case "go-modules", "package-management":
				handoffs = append(handoffs, NextHandoff{
					ToAgent:  "golang-expert",
					Summary:  fmt.Sprintf("Improve dependency management: %s", insight.Description),
					Context:  "Project analysis identified dependency issues",
					Priority: "medium",
				})
			}
		}
	}

	return handoffs
}

// Helper methods for architecture analyzer
func (b *BuiltInAgentStrategy) generateArchitectureRecommendations(analysis *agents.ArchitectureAnalysis, req AgentExecutionRequest) []agents.ArchitectureRecommendation {
	var recommendations []agents.ArchitectureRecommendation

	// Analyze complexity
	if analysis.ComplexityScore > 7.0 {
		recommendations = append(recommendations, agents.ArchitectureRecommendation{
			Type:        "refactor",
			Priority:    "high",
			Description: "High complexity detected. Consider breaking down large components",
			Impact:      "Improved maintainability and testability",
		})
	}

	// Analyze coupling
	if analysis.CouplingScore > 6.0 {
		recommendations = append(recommendations, agents.ArchitectureRecommendation{
			Type:        "decouple",
			Priority:    "medium",
			Description: "High coupling between components. Consider introducing interfaces",
			Impact:      "Better modularity and flexibility",
		})
	}

	return recommendations
}

func (b *BuiltInAgentStrategy) createArchitectureHandoffs(analysis *agents.ArchitectureAnalysis, recommendations []agents.ArchitectureRecommendation) []NextHandoff {
	var handoffs []NextHandoff

	for _, rec := range recommendations {
		if rec.Priority == "high" {
			switch rec.Type {
			case "refactor":
				handoffs = append(handoffs, NextHandoff{
					ToAgent:  "golang-expert",
					Summary:  fmt.Sprintf("Refactor high complexity components: %s", rec.Description),
					Context:  "Architecture analysis identified refactoring opportunities",
					Priority: "high",
				})
			case "decouple":
				handoffs = append(handoffs, NextHandoff{
					ToAgent:  "architect-expert",
					Summary:  fmt.Sprintf("Improve component decoupling: %s", rec.Description),
					Context:  "Architecture analysis found high coupling",
					Priority: "medium",
				})
			}
		}
	}

	return handoffs
}

// Helper methods for agent manager
func (b *BuiltInAgentStrategy) createOrchestrationPlan(req AgentExecutionRequest) *agents.OrchestrationPlan {
	// Analyze the request to determine what needs orchestration
	plan := &agents.OrchestrationPlan{
		Type:              b.determineOrchestrationType(req),
		Agents:            []string{},
		EstimatedDuration: "30-60 minutes",
		Phases:            []agents.OrchestrationPhase{},
	}

	// Create orchestration plan based on request
	if strings.Contains(req.Summary, "full-stack") || strings.Contains(req.Summary, "complete") {
		plan = b.createFullStackPlan(req)
	} else if strings.Contains(req.Summary, "refactor") {
		plan = b.createRefactorPlan(req)
	} else if strings.Contains(req.Summary, "test") {
		plan = b.createTestingPlan(req)
	} else {
		plan = b.createGenericPlan(req)
	}

	return plan
}

func (b *BuiltInAgentStrategy) determineOrchestrationType(req AgentExecutionRequest) string {
	if strings.Contains(req.Summary, "implement") {
		return "implementation"
	} else if strings.Contains(req.Summary, "refactor") {
		return "refactoring"
	} else if strings.Contains(req.Summary, "test") {
		return "testing"
	} else if strings.Contains(req.Summary, "deploy") {
		return "deployment"
	}
	return "analysis"
}

func (b *BuiltInAgentStrategy) createFullStackPlan(req AgentExecutionRequest) *agents.OrchestrationPlan {
	return &agents.OrchestrationPlan{
		Type:              "full-stack-implementation",
		Agents:            []string{"api-expert", "golang-expert", "typescript-expert", "test-expert", "devops-expert"},
		EstimatedDuration: "2-4 hours",
		Phases: []agents.OrchestrationPhase{
			{Name: "Analysis", Agents: []string{"api-expert"}, Duration: "15 minutes"},
			{Name: "Backend", Agents: []string{"golang-expert"}, Duration: "60 minutes"},
			{Name: "Frontend", Agents: []string{"typescript-expert"}, Duration: "45 minutes"},
			{Name: "Testing", Agents: []string{"test-expert"}, Duration: "30 minutes"},
			{Name: "Deployment", Agents: []string{"devops-expert"}, Duration: "20 minutes"},
		},
	}
}

func (b *BuiltInAgentStrategy) createRefactorPlan(req AgentExecutionRequest) *agents.OrchestrationPlan {
	return &agents.OrchestrationPlan{
		Type:              "refactoring",
		Agents:            []string{"architecture-analyzer", "golang-expert", "test-expert"},
		EstimatedDuration: "1-2 hours",
		Phases: []agents.OrchestrationPhase{
			{Name: "Analysis", Agents: []string{"architecture-analyzer"}, Duration: "20 minutes"},
			{Name: "Refactoring", Agents: []string{"golang-expert"}, Duration: "60 minutes"},
			{Name: "Testing", Agents: []string{"test-expert"}, Duration: "30 minutes"},
		},
	}
}

func (b *BuiltInAgentStrategy) createTestingPlan(req AgentExecutionRequest) *agents.OrchestrationPlan {
	return &agents.OrchestrationPlan{
		Type:              "testing",
		Agents:            []string{"test-expert", "golang-expert"},
		EstimatedDuration: "45 minutes",
		Phases: []agents.OrchestrationPhase{
			{Name: "Test Analysis", Agents: []string{"test-expert"}, Duration: "15 minutes"},
			{Name: "Test Implementation", Agents: []string{"golang-expert"}, Duration: "30 minutes"},
		},
	}
}

func (b *BuiltInAgentStrategy) createGenericPlan(req AgentExecutionRequest) *agents.OrchestrationPlan {
	return &agents.OrchestrationPlan{
		Type:              "generic-analysis",
		Agents:            []string{"project-manager"},
		EstimatedDuration: "15-30 minutes",
		Phases: []agents.OrchestrationPhase{
			{Name: "Analysis", Agents: []string{"project-manager"}, Duration: "20 minutes"},
		},
	}
}

func (b *BuiltInAgentStrategy) createOrchestrationHandoffs(plan *agents.OrchestrationPlan) []NextHandoff {
	var handoffs []NextHandoff

	// Create handoffs for each phase
	for i, phase := range plan.Phases {
		for _, agent := range phase.Agents {
			handoffs = append(handoffs, NextHandoff{
				ToAgent: agent,
				Summary: fmt.Sprintf("Execute %s phase: %s", phase.Name, phase.Duration),
				Context: fmt.Sprintf("Orchestration plan phase %d of %d", i+1, len(plan.Phases)),
				Priority: func() string {
					if i == 0 {
						return "high"
					}
					return "normal"
				}(),
			})
		}
	}

	return handoffs
}

// Output formatting methods
func (b *BuiltInAgentStrategy) formatProjectManagerOutput(analysis *agents.ProjectAnalysis, insights []agents.ProjectInsight) string {
	var output strings.Builder

	output.WriteString("# 📋 Project Management Analysis\n\n")
	output.WriteString(fmt.Sprintf("**Project Type:** %s\n", analysis.ProjectType))
	output.WriteString(fmt.Sprintf("**Total Files:** %d\n", analysis.FileCount))
	output.WriteString(fmt.Sprintf("**Languages:** %s\n", strings.Join(analysis.Languages, ", ")))
	output.WriteString(fmt.Sprintf("**Test Coverage:** %.1f%%\n\n", analysis.TestCoverage))

	output.WriteString("## 🔍 Key Insights\n\n")
	for i, insight := range insights {
		icon := "ℹ️"
		switch insight.Type {
		case "positive":
			icon = "✅"
		case "issue":
			icon = "⚠️"
		case "improvement":
			icon = "🔧"
		}

		output.WriteString(fmt.Sprintf("%d. %s **%s**: %s (Impact: %s)\n",
			i+1, icon, insight.Category, insight.Description, insight.Impact))
	}

	output.WriteString("\n## 📊 Project Health Score\n\n")
	score := b.calculateProjectHealthScore(analysis, insights)
	output.WriteString(fmt.Sprintf("**Overall Score:** %.1f/10\n", score))

	return output.String()
}

func (b *BuiltInAgentStrategy) formatArchitectureOutput(analysis *agents.ArchitectureAnalysis, recommendations []agents.ArchitectureRecommendation) string {
	var output strings.Builder

	output.WriteString("# 🏗️ Architecture Analysis Report\n\n")
	output.WriteString(fmt.Sprintf("**Architecture Pattern:** %s\n", analysis.Pattern))
	output.WriteString(fmt.Sprintf("**Components:** %d\n", len(analysis.Components)))
	output.WriteString(fmt.Sprintf("**Complexity Score:** %.2f/10\n", analysis.ComplexityScore))
	output.WriteString(fmt.Sprintf("**Coupling Score:** %.2f/10\n\n", analysis.CouplingScore))

	output.WriteString("## 🎯 Recommendations\n\n")
	for i, rec := range recommendations {
		priority := "📌"
		switch rec.Priority {
		case "high":
			priority = "🚨"
		case "medium":
			priority = "⚠️"
		case "low":
			priority = "💡"
		}

		output.WriteString(fmt.Sprintf("%d. %s **%s Priority**: %s\n",
			i+1, priority, rec.Priority, rec.Description))
		output.WriteString(fmt.Sprintf("   - **Impact**: %s\n\n", rec.Impact))
	}

	return output.String()
}

func (b *BuiltInAgentStrategy) formatAgentManagerOutput(plan *agents.OrchestrationPlan) string {
	var output strings.Builder

	output.WriteString("# 🤖 Agent Orchestration Plan\n\n")
	output.WriteString(fmt.Sprintf("**Orchestration Type:** %s\n", plan.Type))
	output.WriteString(fmt.Sprintf("**Estimated Duration:** %s\n", plan.EstimatedDuration))
	output.WriteString(fmt.Sprintf("**Total Agents:** %d\n\n", len(plan.Agents)))

	output.WriteString("## 📋 Execution Phases\n\n")
	for i, phase := range plan.Phases {
		output.WriteString(fmt.Sprintf("%d. **%s** (%s)\n", i+1, phase.Name, phase.Duration))
		output.WriteString(fmt.Sprintf("   - Agents: %s\n\n", strings.Join(phase.Agents, ", ")))
	}

	output.WriteString("## 🎯 Next Steps\n\n")
	output.WriteString("The following agents will be engaged in sequence according to the orchestration plan.\n")
	output.WriteString("Each agent will receive the context and results from previous phases.\n")

	return output.String()
}

// Helper methods
func (b *BuiltInAgentStrategy) generateProjectArtifacts(analysis *agents.ProjectAnalysis) []string {
	artifacts := []string{
		"Project analysis report",
		"Insights and recommendations",
	}

	if analysis.TestCoverage > 0 {
		artifacts = append(artifacts, "Test coverage analysis")
	}

	if len(analysis.Languages) > 1 {
		artifacts = append(artifacts, "Multi-language project assessment")
	}

	return artifacts
}

func (b *BuiltInAgentStrategy) generateArchitectureArtifacts(analysis *agents.ArchitectureAnalysis) []string {
	return []string{
		"Architecture analysis report",
		"Component relationship diagram",
		"Complexity metrics",
		"Refactoring recommendations",
	}
}

func (b *BuiltInAgentStrategy) calculateProjectHealthScore(analysis *agents.ProjectAnalysis, insights []agents.ProjectInsight) float64 {
	score := 5.0 // Base score

	// Adjust based on test coverage
	if analysis.TestCoverage > 80 {
		score += 2.0
	} else if analysis.TestCoverage > 50 {
		score += 1.0
	} else if analysis.TestCoverage < 20 {
		score -= 2.0
	}

	// Adjust based on insights
	for _, insight := range insights {
		switch insight.Type {
		case "positive":
			if insight.Impact == "high" {
				score += 1.0
			} else {
				score += 0.5
			}
		case "issue":
			if insight.Impact == "high" {
				score -= 1.5
			} else {
				score -= 0.5
			}
		}
	}

	// Ensure score is within bounds
	if score > 10.0 {
		score = 10.0
	} else if score < 0.0 {
		score = 0.0
	}

	return score
}
