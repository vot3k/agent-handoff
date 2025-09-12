# Workflow Registry System

This document defines the workflow registry system used by Claude Code sub-agents.

## Overview

The workflow registry system:
- Defines standard workflows
- Validates workflow definitions
- Tracks workflow state
- Enforces workflow rules
- Manages agent handoffs

## Registry Structure

```yaml
workflow_registry:
  # Required fields for workflow definition
  required_fields:
    name: string           # Unique workflow identifier
    description: string    # Workflow purpose and triggers
    stages: Stage[]        # Workflow stages
    validation: Rule[]     # Validation rules

  # Stage definition structure
  stage_definition:
    name: string          # Stage name
    agent: string         # Required agent
    requires: string[]    # Required previous stages
    provides: string[]    # Required outputs
    optional: boolean     # Can be skipped?
    parallel: boolean     # Can run in parallel?

  # Rules for workflow validation
  validation_rules:
    - valid_sequence: true      # Stages form valid path
    - no_cycles: true          # No circular dependencies
    - all_agents_valid: true   # All agents exist
    - inputs_provided: true    # All required inputs available

  # Runtime validation
  runtime_checks:
    - agent_available: true    # Required agent is ready
    - inputs_ready: true       # Required inputs are ready
    - no_conflicts: true       # No conflicting parallel work
    - handoff_valid: true      # Proper handoff protocol used
```

## File-Based Workflow Coordination

### Workflow Artifacts
```yaml
workflow_coordination:
  # Files used for workflow coordination
  coordination_files:
    - .claude/workflows/active.json     # Currently active workflows
    - .claude/handoffs/                # Agent handoff documents
    - .claude/context/                 # Shared workflow context
  
  # How workflows progress without state
  progression_method:
    - User invokes agent for next stage
    - Agent reads handoff files from previous stage
    - Agent performs work and creates outputs
    - Agent writes handoff for next stage
```

### Handoff Files
```yaml
handoff_structure:
  # Standard handoff file format
  location: ".claude/handoffs/[timestamp]-[from]-to-[to].md"
  
  content:
    header:
      - workflow: "Feature Implementation"
      - stage: "API Design"
      - from: "architect-expert"
      - to: "api-expert"
      - timestamp: "2024-01-15T10:30:00Z"
    
    context:
      - previous_work: "Architecture defined"
      - requirements: "RESTful API needed"
      - constraints: "Must support versioning"
    
    artifacts:
      - created: ["architecture/system-design.md"]
      - modified: ["requirements/api.md"]
      - referenced: ["specs/data-models.md"]
    
    next_steps:
      - "Design API endpoints"
      - "Define request/response formats"
      - "Document authentication flow"
```

## Agent Integration

### Agent Requirements
```yaml
agent_workflow_support:
  # Required agent capabilities
  capabilities:
    - workflow_awareness: true    # Understands position
    - file_coordination: true    # Uses files for context
    - handoff_protocol: true     # Follows protocols
    - output_validation: true    # Validates outputs

  # Workflow participation via files
  workflow_artifacts:
    reads_handoffs: ".claude/handoffs/*-to-[agent].md"
    creates_artifacts: "[output files]"
    writes_handoffs: ".claude/handoffs/[agent]-to-*.md"
    updates_docs: "docs/"
```

### Handoff Protocol
```yaml
handoff_protocol:
  # Standard handoff message
  message:
    metadata:
      workflow_id: string
      from_stage: string
      to_stage: string
      timestamp: string
    
    content:
      outputs: {}          # Stage outputs
      context: {}          # Workflow context
      requirements: {}     # Next stage needs
      validation: {}       # Output validation

  # Handoff validation
  validation:
    - all_outputs_provided: true
    - outputs_validated: true
    - context_preserved: true
    - next_stage_valid: true
```

## Standard Workflows

### Feature Implementation
```yaml
feature_implementation:
  name: "Feature Implementation"
  description: "Complete feature implementation workflow"
  stages:
    - name: "Planning"
      agent: project-manager
      requires: []
      provides: ["requirements", "priorities"]
    
    - name: "Architecture"
      agent: architect-expert
      requires: ["Planning"]
      provides: ["architecture", "patterns"]
    
    - name: "API Design"
      agent: api-expert
      requires: ["Architecture"]
      provides: ["api_contracts"]
      optional: true
    
    - name: "Security Planning"
      agent: security-expert
      requires: ["Architecture"]
      provides: ["security_requirements"]
    
    - name: "Frontend Implementation"
      agent: typescript-expert
      requires: ["API Design", "Security Planning"]
      provides: ["frontend_implementation"]
      parallel: true
    
    - name: "Backend Implementation"
      agent: golang-expert
      requires: ["API Design", "Security Planning"]
      provides: ["backend_implementation"]
      parallel: true
    
    - name: "Testing"
      agent: test-expert
      requires: ["Frontend Implementation", "Backend Implementation"]
      provides: ["test_results"]
    
    - name: "Deployment"
      agent: devops-expert
      requires: ["Testing"]
      provides: ["deployment_status"]
    
    - name: "Documentation"
      agent: tech-writer
      requires: ["Deployment"]
      provides: ["documentation"]
```

### Bug Fix
```yaml
bug_fix:
  name: "Bug Fix"
  description: "Bug investigation and fix workflow"
  stages:
    - name: "Report"
      agent: project-manager
      requires: []
      provides: ["bug_report"]
    
    - name: "Investigation"
      agent: test-expert
      requires: ["Report"]
      provides: ["root_cause"]
    
    - name: "Architecture Review"
      agent: architect-expert
      requires: ["Investigation"]
      provides: ["fix_approach"]
    
    - name: "Implementation"
      agent: ["typescript-expert", "golang-expert"]
      requires: ["Architecture Review"]
      provides: ["fix_implementation"]
    
    - name: "Security Review"
      agent: security-expert
      requires: ["Implementation"]
      provides: ["security_validation"]
    
    - name: "Testing"
      agent: test-expert
      requires: ["Security Review"]
      provides: ["test_results"]
    
    - name: "Documentation"
      agent: tech-writer
      requires: ["Testing"]
      provides: ["documentation"]
```

## Usage

1. Workflow Registration
```yaml
# Register a new workflow
register_workflow:
  workflow: WorkflowDefinition
  validation: ValidationRules
  result: Success|Error
```

2. Workflow Execution
```yaml
# Start workflow execution
start_workflow:
  workflow_id: string
  initial_context: {}
  result: WorkflowState

# Transition to next stage
transition_stage:
  workflow_id: string
  from_stage: string
  to_stage: string
  outputs: {}
  result: WorkflowState
```

3. State Management
```yaml
# Get workflow state
get_workflow_state:
  workflow_id: string
  result: WorkflowState

# Validate state transition
validate_transition:
  workflow_id: string
  from_stage: string
  to_stage: string
  result: Valid|Error
```

## Error Handling

```yaml
error_handling:
  # Validation errors
  validation_error:
    type: string          # Error type
    stage: string        # Failed stage
    reason: string       # Error reason
    recovery: string[]   # Recovery steps

  # Runtime errors
  runtime_error:
    type: string         # Error type
    agent: string        # Failed agent
    stage: string       # Current stage
    context: {}         # Error context
    recovery: string[]  # Recovery steps
```

Remember: This workflow registry system is critical for maintaining proper agent coordination and ensuring reliable workflow execution.