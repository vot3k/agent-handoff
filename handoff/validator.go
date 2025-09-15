package handoff

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// HandoffValidator provides validation for handoffs
type HandoffValidator struct {
	knownAgents     map[string]bool
	schemaVersion   string
	maxSummaryLen   int
	maxRequirements int
	maxNextSteps    int
}

// NewHandoffValidator creates a new validator instance
func NewHandoffValidator() *HandoffValidator {
	return &HandoffValidator{
		knownAgents:     make(map[string]bool),
		schemaVersion:   "1.0",
		maxSummaryLen:   1000,
		maxRequirements: 50,
		maxNextSteps:    20,
	}
}

// RegisterAgent registers an agent for validation
func (v *HandoffValidator) RegisterAgent(agentName string) {
	v.knownAgents[agentName] = true
}

// ValidateHandoff performs comprehensive handoff validation
func (v *HandoffValidator) ValidateHandoff(handoff *Handoff) error {
	if err := v.validateMetadata(&handoff.Metadata); err != nil {
		return fmt.Errorf("metadata validation failed: %w", err)
	}

	if err := v.validateContent(&handoff.Content); err != nil {
		return fmt.Errorf("content validation failed: %w", err)
	}

	if err := v.validateValidation(&handoff.Validation); err != nil {
		return fmt.Errorf("validation section failed: %w", err)
	}

	if err := v.validateArtifacts(&handoff.Content.Artifacts); err != nil {
		return fmt.Errorf("artifacts validation failed: %w", err)
	}

	return nil
}

// validateMetadata validates the metadata section
func (v *HandoffValidator) validateMetadata(metadata *Metadata) error {
	if metadata.FromAgent == "" {
		return fmt.Errorf("from_agent is required")
	}

	if metadata.ToAgent == "" {
		return fmt.Errorf("to_agent is required")
	}

	if metadata.FromAgent == metadata.ToAgent {
		return fmt.Errorf("from_agent and to_agent cannot be the same")
	}

	// Validate agent names format (lowercase, alphanumeric with dashes)
	agentNameRegex := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !agentNameRegex.MatchString(metadata.FromAgent) {
		return fmt.Errorf("from_agent must be lowercase alphanumeric with dashes only")
	}

	if !agentNameRegex.MatchString(metadata.ToAgent) {
		return fmt.Errorf("to_agent must be lowercase alphanumeric with dashes only")
	}

	// Check if agents are registered (optional check)
	if len(v.knownAgents) > 0 {
		if !v.knownAgents[metadata.FromAgent] {
			return fmt.Errorf("from_agent %s is not registered", metadata.FromAgent)
		}
		if !v.knownAgents[metadata.ToAgent] {
			return fmt.Errorf("to_agent %s is not registered", metadata.ToAgent)
		}
	}

	if metadata.Timestamp.IsZero() {
		metadata.Timestamp = time.Now()
	}

	// Validate timestamp is not too far in the future or past
	now := time.Now()
	if metadata.Timestamp.After(now.Add(time.Hour)) {
		return fmt.Errorf("timestamp cannot be more than 1 hour in the future")
	}
	if metadata.Timestamp.Before(now.Add(-24 * time.Hour)) {
		return fmt.Errorf("timestamp cannot be more than 24 hours in the past")
	}

	if metadata.TaskContext == "" {
		return fmt.Errorf("task_context is required")
	}

	// Validate priority
	switch string(metadata.Priority) {
	case string(PriorityLow), string(PriorityNormal), string(PriorityHigh), string(PriorityCritical):
		// Valid priority
	case "":
		metadata.Priority = PriorityNormal // Default
	default:
		return fmt.Errorf("invalid priority: %s", metadata.Priority)
	}

	return nil
}

// validateContent validates the content section
func (v *HandoffValidator) validateContent(content *Content) error {
	if content.Summary == "" {
		return fmt.Errorf("summary is required")
	}

	if len(content.Summary) > v.maxSummaryLen {
		return fmt.Errorf("summary too long (max %d characters)", v.maxSummaryLen)
	}

	// Summary should be descriptive
	if len(strings.TrimSpace(content.Summary)) < 10 {
		return fmt.Errorf("summary too short (minimum 10 characters)")
	}

	if len(content.Requirements) == 0 {
		return fmt.Errorf("at least one requirement is needed")
	}

	if len(content.Requirements) > v.maxRequirements {
		return fmt.Errorf("too many requirements (max %d)", v.maxRequirements)
	}

	// Validate requirements are not empty
	for i, req := range content.Requirements {
		if strings.TrimSpace(req) == "" {
			return fmt.Errorf("requirement %d cannot be empty", i+1)
		}
	}

	if len(content.NextSteps) > v.maxNextSteps {
		return fmt.Errorf("too many next steps (max %d)", v.maxNextSteps)
	}

	// Validate next steps are not empty
	for i, step := range content.NextSteps {
		if strings.TrimSpace(step) == "" {
			return fmt.Errorf("next step %d cannot be empty", i+1)
		}
	}

	// Validate technical details structure
	if content.TechnicalDetails == nil {
		content.TechnicalDetails = make(map[string]interface{})
	}

	return nil
}

// validateValidation validates the validation section
func (v *HandoffValidator) validateValidation(validation *Validation) error {
	if validation.SchemaVersion == "" {
		validation.SchemaVersion = v.schemaVersion
	}

	// Check supported schema versions
	supportedVersions := []string{"1.0", "1.1"}
	versionSupported := false
	for _, version := range supportedVersions {
		if validation.SchemaVersion == version {
			versionSupported = true
			break
		}
	}

	if !versionSupported {
		return fmt.Errorf("unsupported schema version: %s (supported: %v)",
			validation.SchemaVersion, supportedVersions)
	}

	if validation.Checksum == "" {
		return fmt.Errorf("checksum is required")
	}

	// Checksum should be hex string (64 characters for SHA256)
	checksumRegex := regexp.MustCompile(`^[a-f0-9]{64}$`)
	if !checksumRegex.MatchString(validation.Checksum) {
		return fmt.Errorf("checksum must be a valid 64-character hex string")
	}

	return nil
}

// validateArtifacts validates the artifacts section
func (v *HandoffValidator) validateArtifacts(artifacts *Artifacts) error {
	// Validate file paths
	pathRegex := regexp.MustCompile(`^[a-zA-Z0-9/_.-]+$`)

	for _, path := range artifacts.Created {
		if !pathRegex.MatchString(path) {
			return fmt.Errorf("invalid file path in created artifacts: %s", path)
		}
	}

	for _, path := range artifacts.Modified {
		if !pathRegex.MatchString(path) {
			return fmt.Errorf("invalid file path in modified artifacts: %s", path)
		}
	}

	for _, path := range artifacts.Reviewed {
		if !pathRegex.MatchString(path) {
			return fmt.Errorf("invalid file path in reviewed artifacts: %s", path)
		}
	}

	// Check for duplicates across categories
	allPaths := make(map[string]string)

	for _, path := range artifacts.Created {
		if category, exists := allPaths[path]; exists {
			return fmt.Errorf("duplicate artifact path %s in %s and created", path, category)
		}
		allPaths[path] = "created"
	}

	for _, path := range artifacts.Modified {
		if category, exists := allPaths[path]; exists {
			return fmt.Errorf("duplicate artifact path %s in %s and modified", path, category)
		}
		allPaths[path] = "modified"
	}

	for _, path := range artifacts.Reviewed {
		if category, exists := allPaths[path]; exists {
			return fmt.Errorf("duplicate artifact path %s in %s and reviewed", path, category)
		}
		allPaths[path] = "reviewed"
	}

	return nil
}

// ValidateAgentSpecificFields validates technical_details based on agent type
func (v *HandoffValidator) ValidateAgentSpecificFields(handoff *Handoff) error {
	toAgent := handoff.Metadata.ToAgent
	techDetails := handoff.Content.TechnicalDetails

	switch toAgent {
	case "golang-expert":
		return v.validateGolangFields(techDetails)
	case "typescript-expert":
		return v.validateTypescriptFields(techDetails)
	case "api-expert":
		return v.validateAPIFields(techDetails)
	case "test-expert":
		return v.validateTestFields(techDetails)
	case "devops-expert":
		return v.validateDevOpsFields(techDetails)
	default:
		// No specific validation for unknown agents
		return nil
	}
}

// validateGolangFields validates golang-expert specific fields
func (v *HandoffValidator) validateGolangFields(details map[string]interface{}) error {
	expectedFields := []string{"handlers", "services", "models", "repositories"}

	for _, field := range expectedFields {
		if val, exists := details[field]; exists {
			// Should be a string array
			if arr, ok := val.([]interface{}); ok {
				for _, item := range arr {
					if _, ok := item.(string); !ok {
						return fmt.Errorf("golang field %s should contain string values", field)
					}
				}
			} else {
				return fmt.Errorf("golang field %s should be an array", field)
			}
		}
	}

	// Validate test coverage if present
	if coverage, exists := details["test_coverage"]; exists {
		if coverageNum, ok := coverage.(float64); ok {
			if coverageNum < 0 || coverageNum > 100 {
				return fmt.Errorf("test_coverage must be between 0 and 100")
			}
		} else {
			return fmt.Errorf("test_coverage must be a number")
		}
	}

	return nil
}

// validateTypescriptFields validates typescript-expert specific fields
func (v *HandoffValidator) validateTypescriptFields(details map[string]interface{}) error {
	if components, exists := details["components"]; exists {
		if _, ok := components.([]interface{}); !ok {
			return fmt.Errorf("typescript components field should be an array")
		}
	}

	if hooks, exists := details["hooks"]; exists {
		if _, ok := hooks.([]interface{}); !ok {
			return fmt.Errorf("typescript hooks field should be an array")
		}
	}

	return nil
}

// validateAPIFields validates api-expert specific fields
func (v *HandoffValidator) validateAPIFields(details map[string]interface{}) error {
	if endpoints, exists := details["endpoints"]; exists {
		if _, ok := endpoints.([]interface{}); !ok {
			return fmt.Errorf("api endpoints field should be an array")
		}
	}

	if schemas, exists := details["schemas"]; exists {
		if _, ok := schemas.([]interface{}); !ok {
			return fmt.Errorf("api schemas field should be an array")
		}
	}

	return nil
}

// validateTestFields validates test-expert specific fields
func (v *HandoffValidator) validateTestFields(details map[string]interface{}) error {
	if testSuites, exists := details["test_suites"]; exists {
		if _, ok := testSuites.([]interface{}); !ok {
			return fmt.Errorf("test_suites field should be an array")
		}
	}

	if coverage, exists := details["coverage_achieved"]; exists {
		if coverageNum, ok := coverage.(float64); ok {
			if coverageNum < 0 || coverageNum > 100 {
				return fmt.Errorf("coverage_achieved must be between 0 and 100")
			}
		} else {
			return fmt.Errorf("coverage_achieved must be a number")
		}
	}

	return nil
}

// validateDevOpsFields validates devops-expert specific fields
func (v *HandoffValidator) validateDevOpsFields(details map[string]interface{}) error {
	if deployments, exists := details["deployments"]; exists {
		if _, ok := deployments.([]interface{}); !ok {
			return fmt.Errorf("deployments field should be an array")
		}
	}

	if configs, exists := details["configurations"]; exists {
		if _, ok := configs.([]interface{}); !ok {
			return fmt.Errorf("configurations field should be an array")
		}
	}

	return nil
}

// SanitizeHandoff sanitizes and normalizes handoff data
func (v *HandoffValidator) SanitizeHandoff(handoff *Handoff) {
	// Trim whitespace from strings
	handoff.Metadata.FromAgent = strings.TrimSpace(handoff.Metadata.FromAgent)
	handoff.Metadata.ToAgent = strings.TrimSpace(handoff.Metadata.ToAgent)
	handoff.Metadata.TaskContext = strings.TrimSpace(handoff.Metadata.TaskContext)
	handoff.Content.Summary = strings.TrimSpace(handoff.Content.Summary)

	// Normalize requirements
	for i, req := range handoff.Content.Requirements {
		handoff.Content.Requirements[i] = strings.TrimSpace(req)
	}

	// Normalize next steps
	for i, step := range handoff.Content.NextSteps {
		handoff.Content.NextSteps[i] = strings.TrimSpace(step)
	}

	// Remove empty requirements and next steps
	handoff.Content.Requirements = removeEmptyStrings(handoff.Content.Requirements)
	handoff.Content.NextSteps = removeEmptyStrings(handoff.Content.NextSteps)

	// Normalize artifact paths
	handoff.Content.Artifacts.Created = normalizePaths(handoff.Content.Artifacts.Created)
	handoff.Content.Artifacts.Modified = normalizePaths(handoff.Content.Artifacts.Modified)
	handoff.Content.Artifacts.Reviewed = normalizePaths(handoff.Content.Artifacts.Reviewed)
}

// removeEmptyStrings removes empty strings from a slice
func removeEmptyStrings(slice []string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if strings.TrimSpace(s) != "" {
			result = append(result, s)
		}
	}
	return result
}

// normalizePaths normalizes file paths
func normalizePaths(paths []string) []string {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path != "" {
			// Remove double slashes
			path = regexp.MustCompile(`/+`).ReplaceAllString(path, "/")
			// Remove trailing slash
			path = strings.TrimSuffix(path, "/")
			result = append(result, path)
		}
	}
	return result
}
