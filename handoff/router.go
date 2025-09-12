package handoff

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// HandoffRouter manages intelligent routing of handoffs to appropriate agents
type HandoffRouter struct {
	routes        map[string][]RouteRule
	fallbackAgent string
	routesMutex   sync.RWMutex
}

// RouteRule defines conditions for routing a handoff to an agent
type RouteRule struct {
	Name        string            `json:"name"`
	TargetAgent string            `json:"target_agent"`
	Priority    int               `json:"priority"` // Higher number = higher priority
	Conditions  []RouteCondition  `json:"conditions"`
	Transforms  []RouteTransform  `json:"transforms,omitempty"`
}

// RouteCondition defines a condition that must be met for the rule to apply
type RouteCondition struct {
	Type      ConditionType `json:"type"`
	Field     string        `json:"field"`
	Operator  string        `json:"operator"`
	Value     interface{}   `json:"value"`
	CaseSensitive bool      `json:"case_sensitive,omitempty"`
}

// RouteTransform defines how to modify a handoff before routing
type RouteTransform struct {
	Type   TransformType `json:"type"`
	Field  string        `json:"field"`
	Action string        `json:"action"`
	Value  interface{}   `json:"value,omitempty"`
}

// ConditionType defines the type of condition
type ConditionType string

const (
	ConditionContent      ConditionType = "content"      // Check content fields
	ConditionMetadata     ConditionType = "metadata"     // Check metadata fields
	ConditionTechnical    ConditionType = "technical"    // Check technical details
	ConditionArtifact     ConditionType = "artifact"     // Check artifact patterns
	ConditionComplexQuery ConditionType = "complex"      // Complex query conditions
)

// TransformType defines the type of transformation
type TransformType string

const (
	TransformMetadata  TransformType = "metadata"   // Modify metadata
	TransformContent   TransformType = "content"    // Modify content
	TransformTechnical TransformType = "technical"  // Modify technical details
	TransformPriority  TransformType = "priority"   // Modify priority
)

// NewHandoffRouter creates a new handoff router
func NewHandoffRouter(fallbackAgent string) *HandoffRouter {
	return &HandoffRouter{
		routes:        make(map[string][]RouteRule),
		fallbackAgent: fallbackAgent,
	}
}

// AddRoute adds a routing rule for a specific source agent
func (r *HandoffRouter) AddRoute(fromAgent string, rule RouteRule) {
	r.routesMutex.Lock()
	defer r.routesMutex.Unlock()
	
	if r.routes[fromAgent] == nil {
		r.routes[fromAgent] = make([]RouteRule, 0)
	}
	
	r.routes[fromAgent] = append(r.routes[fromAgent], rule)
	
	// Sort by priority (higher first)
	rules := r.routes[fromAgent]
	for i := 0; i < len(rules); i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority < rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}
}

// RouteHandoff determines the best target agent for a handoff
func (r *HandoffRouter) RouteHandoff(ctx context.Context, handoff *Handoff) (string, error) {
	r.routesMutex.RLock()
	defer r.routesMutex.RUnlock()
	
	fromAgent := handoff.Metadata.FromAgent
	rules, exists := r.routes[fromAgent]
	
	if !exists || len(rules) == 0 {
		// No specific rules, use the target agent from handoff or fallback
		if handoff.Metadata.ToAgent != "" {
			return handoff.Metadata.ToAgent, nil
		}
		if r.fallbackAgent != "" {
			return r.fallbackAgent, nil
		}
		return "", fmt.Errorf("no routing rules found for agent %s and no fallback configured", fromAgent)
	}
	
	// Evaluate rules in priority order
	for _, rule := range rules {
		if r.evaluateRule(handoff, rule) {
			// Apply transforms if specified
			if err := r.applyTransforms(handoff, rule.Transforms); err != nil {
				return "", fmt.Errorf("failed to apply transforms for rule %s: %w", rule.Name, err)
			}
			
			return rule.TargetAgent, nil
		}
	}
	
	// No rules matched, use original target or fallback
	if handoff.Metadata.ToAgent != "" {
		return handoff.Metadata.ToAgent, nil
	}
	
	if r.fallbackAgent != "" {
		return r.fallbackAgent, nil
	}
	
	return "", fmt.Errorf("no routing rules matched and no fallback configured")
}

// evaluateRule checks if all conditions in a rule are met
func (r *HandoffRouter) evaluateRule(handoff *Handoff, rule RouteRule) bool {
	for _, condition := range rule.Conditions {
		if !r.evaluateCondition(handoff, condition) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single condition
func (r *HandoffRouter) evaluateCondition(handoff *Handoff, condition RouteCondition) bool {
	var value interface{}
	
	switch condition.Type {
	case ConditionMetadata:
		value = r.getMetadataValue(handoff, condition.Field)
	case ConditionContent:
		value = r.getContentValue(handoff, condition.Field)
	case ConditionTechnical:
		value = r.getTechnicalValue(handoff, condition.Field)
	case ConditionArtifact:
		value = r.getArtifactValue(handoff, condition.Field)
	case ConditionComplexQuery:
		return r.evaluateComplexQuery(handoff, condition)
	default:
		return false
	}
	
	return r.compareValues(value, condition.Operator, condition.Value, condition.CaseSensitive)
}

// getMetadataValue retrieves a metadata field value
func (r *HandoffRouter) getMetadataValue(handoff *Handoff, field string) interface{} {
	switch field {
	case "from_agent":
		return handoff.Metadata.FromAgent
	case "to_agent":
		return handoff.Metadata.ToAgent
	case "task_context":
		return handoff.Metadata.TaskContext
	case "priority":
		return string(handoff.Metadata.Priority)
	case "handoff_id":
		return handoff.Metadata.HandoffID
	default:
		return nil
	}
}

// getContentValue retrieves a content field value
func (r *HandoffRouter) getContentValue(handoff *Handoff, field string) interface{} {
	switch field {
	case "summary":
		return handoff.Content.Summary
	case "requirements":
		return handoff.Content.Requirements
	case "next_steps":
		return handoff.Content.NextSteps
	case "requirements_count":
		return len(handoff.Content.Requirements)
	case "next_steps_count":
		return len(handoff.Content.NextSteps)
	default:
		return nil
	}
}

// getTechnicalValue retrieves a technical details value
func (r *HandoffRouter) getTechnicalValue(handoff *Handoff, field string) interface{} {
	if handoff.Content.TechnicalDetails == nil {
		return nil
	}
	return handoff.Content.TechnicalDetails[field]
}

// getArtifactValue retrieves artifact-related values
func (r *HandoffRouter) getArtifactValue(handoff *Handoff, field string) interface{} {
	switch field {
	case "created":
		return handoff.Content.Artifacts.Created
	case "modified":
		return handoff.Content.Artifacts.Modified
	case "reviewed":
		return handoff.Content.Artifacts.Reviewed
	case "created_count":
		return len(handoff.Content.Artifacts.Created)
	case "modified_count":
		return len(handoff.Content.Artifacts.Modified)
	case "reviewed_count":
		return len(handoff.Content.Artifacts.Reviewed)
	case "total_artifacts":
		return len(handoff.Content.Artifacts.Created) + 
			   len(handoff.Content.Artifacts.Modified) + 
			   len(handoff.Content.Artifacts.Reviewed)
	default:
		return nil
	}
}

// evaluateComplexQuery handles complex query conditions
func (r *HandoffRouter) evaluateComplexQuery(handoff *Handoff, condition RouteCondition) bool {
	query := condition.Field
	
	switch query {
	case "has_go_files":
		return r.hasFilesWithExtension(handoff, ".go")
	case "has_typescript_files":
		return r.hasFilesWithExtension(handoff, ".ts") || r.hasFilesWithExtension(handoff, ".tsx")
	case "has_test_files":
		return r.hasFilesWithPattern(handoff, "_test.") || 
			   r.hasFilesWithPattern(handoff, ".test.") ||
			   r.hasFilesWithPattern(handoff, "/test/")
	case "has_api_spec":
		return r.hasFilesWithExtension(handoff, ".yaml") || 
			   r.hasFilesWithExtension(handoff, ".yml") ||
			   r.hasFilesWithPattern(handoff, "openapi") ||
			   r.hasFilesWithPattern(handoff, "swagger")
	case "is_implementation_handoff":
		return strings.Contains(strings.ToLower(handoff.Content.Summary), "implement") ||
			   strings.Contains(strings.ToLower(handoff.Content.Summary), "code") ||
			   len(handoff.Content.Artifacts.Created) > 0
	case "is_testing_handoff":
		return strings.Contains(strings.ToLower(handoff.Content.Summary), "test") ||
			   strings.Contains(strings.ToLower(handoff.Content.Summary), "coverage") ||
			   r.hasFilesWithPattern(handoff, "test")
	case "is_deployment_handoff":
		return strings.Contains(strings.ToLower(handoff.Content.Summary), "deploy") ||
			   strings.Contains(strings.ToLower(handoff.Content.Summary), "docker") ||
			   r.hasFilesWithPattern(handoff, "deploy") ||
			   r.hasFilesWithPattern(handoff, "docker")
	default:
		return false
	}
}

// hasFilesWithExtension checks if any artifacts have the specified extension
func (r *HandoffRouter) hasFilesWithExtension(handoff *Handoff, ext string) bool {
	allFiles := append(handoff.Content.Artifacts.Created, handoff.Content.Artifacts.Modified...)
	allFiles = append(allFiles, handoff.Content.Artifacts.Reviewed...)
	
	for _, file := range allFiles {
		if strings.HasSuffix(strings.ToLower(file), ext) {
			return true
		}
	}
	return false
}

// hasFilesWithPattern checks if any artifacts contain the specified pattern
func (r *HandoffRouter) hasFilesWithPattern(handoff *Handoff, pattern string) bool {
	allFiles := append(handoff.Content.Artifacts.Created, handoff.Content.Artifacts.Modified...)
	allFiles = append(allFiles, handoff.Content.Artifacts.Reviewed...)
	
	for _, file := range allFiles {
		if strings.Contains(strings.ToLower(file), strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// compareValues compares two values using the specified operator
func (r *HandoffRouter) compareValues(actual interface{}, operator string, expected interface{}, caseSensitive bool) bool {
	switch operator {
	case "equals", "eq":
		return r.valuesEqual(actual, expected, caseSensitive)
	case "not_equals", "ne":
		return !r.valuesEqual(actual, expected, caseSensitive)
	case "contains":
		return r.valueContains(actual, expected, caseSensitive)
	case "not_contains":
		return !r.valueContains(actual, expected, caseSensitive)
	case "starts_with":
		return r.valueStartsWith(actual, expected, caseSensitive)
	case "ends_with":
		return r.valueEndsWith(actual, expected, caseSensitive)
	case "greater_than", "gt":
		return r.valueGreaterThan(actual, expected)
	case "less_than", "lt":
		return r.valueLessThan(actual, expected)
	case "greater_equal", "ge":
		return r.valueGreaterEqual(actual, expected)
	case "less_equal", "le":
		return r.valueLessEqual(actual, expected)
	case "in":
		return r.valueIn(actual, expected, caseSensitive)
	case "regex":
		return r.valueMatchesRegex(actual, expected)
	default:
		return false
	}
}

// Helper comparison functions
func (r *HandoffRouter) valuesEqual(actual, expected interface{}, caseSensitive bool) bool {
	if actual == nil || expected == nil {
		return actual == expected
	}
	
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	
	if !caseSensitive {
		return strings.EqualFold(actualStr, expectedStr)
	}
	return actualStr == expectedStr
}

func (r *HandoffRouter) valueContains(actual, expected interface{}, caseSensitive bool) bool {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	
	if !caseSensitive {
		return strings.Contains(strings.ToLower(actualStr), strings.ToLower(expectedStr))
	}
	return strings.Contains(actualStr, expectedStr)
}

func (r *HandoffRouter) valueStartsWith(actual, expected interface{}, caseSensitive bool) bool {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	
	if !caseSensitive {
		return strings.HasPrefix(strings.ToLower(actualStr), strings.ToLower(expectedStr))
	}
	return strings.HasPrefix(actualStr, expectedStr)
}

func (r *HandoffRouter) valueEndsWith(actual, expected interface{}, caseSensitive bool) bool {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	
	if !caseSensitive {
		return strings.HasSuffix(strings.ToLower(actualStr), strings.ToLower(expectedStr))
	}
	return strings.HasSuffix(actualStr, expectedStr)
}

func (r *HandoffRouter) valueGreaterThan(actual, expected interface{}) bool {
	actualNum, ok1 := r.toFloat64(actual)
	expectedNum, ok2 := r.toFloat64(expected)
	return ok1 && ok2 && actualNum > expectedNum
}

func (r *HandoffRouter) valueLessThan(actual, expected interface{}) bool {
	actualNum, ok1 := r.toFloat64(actual)
	expectedNum, ok2 := r.toFloat64(expected)
	return ok1 && ok2 && actualNum < expectedNum
}

func (r *HandoffRouter) valueGreaterEqual(actual, expected interface{}) bool {
	actualNum, ok1 := r.toFloat64(actual)
	expectedNum, ok2 := r.toFloat64(expected)
	return ok1 && ok2 && actualNum >= expectedNum
}

func (r *HandoffRouter) valueLessEqual(actual, expected interface{}) bool {
	actualNum, ok1 := r.toFloat64(actual)
	expectedNum, ok2 := r.toFloat64(expected)
	return ok1 && ok2 && actualNum <= expectedNum
}

func (r *HandoffRouter) valueIn(actual, expected interface{}, caseSensitive bool) bool {
	expectedSlice, ok := expected.([]interface{})
	if !ok {
		return false
	}
	
	for _, item := range expectedSlice {
		if r.valuesEqual(actual, item, caseSensitive) {
			return true
		}
	}
	return false
}

func (r *HandoffRouter) valueMatchesRegex(actual, expected interface{}) bool {
	actualStr := fmt.Sprintf("%v", actual)
	pattern := fmt.Sprintf("%v", expected)
	
	matched, err := regexp.MatchString(pattern, actualStr)
	return err == nil && matched
}

func (r *HandoffRouter) toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	default:
		return 0, false
	}
}

// applyTransforms applies route transforms to a handoff
func (r *HandoffRouter) applyTransforms(handoff *Handoff, transforms []RouteTransform) error {
	for _, transform := range transforms {
		if err := r.applyTransform(handoff, transform); err != nil {
			return fmt.Errorf("transform failed: %w", err)
		}
	}
	return nil
}

// applyTransform applies a single transform
func (r *HandoffRouter) applyTransform(handoff *Handoff, transform RouteTransform) error {
	switch transform.Type {
	case TransformMetadata:
		return r.transformMetadata(handoff, transform)
	case TransformContent:
		return r.transformContent(handoff, transform)
	case TransformTechnical:
		return r.transformTechnical(handoff, transform)
	case TransformPriority:
		return r.transformPriority(handoff, transform)
	default:
		return fmt.Errorf("unknown transform type: %s", transform.Type)
	}
}

func (r *HandoffRouter) transformMetadata(handoff *Handoff, transform RouteTransform) error {
	switch transform.Action {
	case "set":
		switch transform.Field {
		case "task_context":
			if val, ok := transform.Value.(string); ok {
				handoff.Metadata.TaskContext = val
			}
		case "priority":
			if val, ok := transform.Value.(string); ok {
				handoff.Metadata.Priority = Priority(val)
			}
		}
	case "append":
		switch transform.Field {
		case "task_context":
			if val, ok := transform.Value.(string); ok {
				handoff.Metadata.TaskContext += " " + val
			}
		}
	}
	return nil
}

func (r *HandoffRouter) transformContent(handoff *Handoff, transform RouteTransform) error {
	switch transform.Action {
	case "set":
		switch transform.Field {
		case "summary":
			if val, ok := transform.Value.(string); ok {
				handoff.Content.Summary = val
			}
		}
	case "append":
		switch transform.Field {
		case "summary":
			if val, ok := transform.Value.(string); ok {
				handoff.Content.Summary += " " + val
			}
		case "requirements":
			if val, ok := transform.Value.(string); ok {
				handoff.Content.Requirements = append(handoff.Content.Requirements, val)
			}
		case "next_steps":
			if val, ok := transform.Value.(string); ok {
				handoff.Content.NextSteps = append(handoff.Content.NextSteps, val)
			}
		}
	}
	return nil
}

func (r *HandoffRouter) transformTechnical(handoff *Handoff, transform RouteTransform) error {
	if handoff.Content.TechnicalDetails == nil {
		handoff.Content.TechnicalDetails = make(map[string]interface{})
	}
	
	switch transform.Action {
	case "set":
		handoff.Content.TechnicalDetails[transform.Field] = transform.Value
	case "append":
		if existing, exists := handoff.Content.TechnicalDetails[transform.Field]; exists {
			if existingSlice, ok := existing.([]interface{}); ok {
				if val, ok := transform.Value.(string); ok {
					handoff.Content.TechnicalDetails[transform.Field] = append(existingSlice, val)
				}
			}
		} else {
			if val, ok := transform.Value.(string); ok {
				handoff.Content.TechnicalDetails[transform.Field] = []interface{}{val}
			}
		}
	}
	return nil
}

func (r *HandoffRouter) transformPriority(handoff *Handoff, transform RouteTransform) error {
	if val, ok := transform.Value.(string); ok {
		handoff.Metadata.Priority = Priority(val)
	}
	return nil
}