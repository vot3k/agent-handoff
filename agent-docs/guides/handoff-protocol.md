# Agent Handoff Protocol

## Overview

Since Claude Code agents are stateless, all communication between agents happens through files and is orchestrated by the main Claude instance. This document defines the standard protocols for agent handoffs.

## Core Principles

1. **Agents are stateless** - Each invocation starts fresh
2. **Files are the communication medium** - All data exchange via files
3. **Claude orchestrates** - The main Claude instance manages agent sequencing
4. **Explicit is better** - Clear file naming and locations

## Standard Handoff Structure

### File Naming Convention
```
/project/
  .agent-artifacts/
    {timestamp}-{source-agent}-{artifact-type}.{ext}
    
Examples:
  2024-01-15-api-expert-auth-spec.yaml
  2024-01-15-golang-expert-implementation.md
  2024-01-15-test-expert-coverage-report.json
```

### Metadata Header
Every handoff file should include:
```yaml
---
source_agent: api-expert
target_agent: golang-expert
task_id: AUTH-001
created: 2024-01-15T10:30:00Z
workflow_stage: api_design
artifacts:
  - auth-endpoints.yaml
  - auth-flow.md
---
```

## Agent-Specific Handoffs

### API Expert → Implementation Experts
```yaml
# File: .agent-artifacts/api-design-{feature}.yaml
---
source_agent: api-expert
target_agents: [golang-expert, typescript-expert]
---

endpoints:
  - path: /auth/login
    method: POST
    request:
      content-type: application/json
      schema: 
        $ref: '#/schemas/LoginRequest'
    response:
      200:
        schema:
          $ref: '#/schemas/AuthToken'
      401:
        schema:
          $ref: '#/schemas/Error'

schemas:
  LoginRequest:
    type: object
    required: [email, password]
    properties:
      email: 
        type: string
        format: email
      password:
        type: string
        minLength: 8
```

### Architect Expert → All Agents
```markdown
# File: .agent-artifacts/architecture-decision-{feature}.md
---
source_agent: architect-expert
target_agents: all
adr_number: 001
---

# Authentication Architecture

## Decision
Use JWT tokens with refresh token rotation

## Implementation Requirements
- Token expiry: 15 minutes
- Refresh token expiry: 7 days
- Store refresh tokens in Redis
- Use RS256 for signing

## Component Boundaries
- Auth Service: Handles all authentication
- API Gateway: Validates tokens only
- User Service: Manages user data
```

### Implementation → Test Expert
```markdown
# File: .agent-artifacts/implementation-summary-{feature}.md
---
source_agent: golang-expert
target_agent: test-expert
---

# Implementation Summary

## Completed Files
- `/src/auth/handler.go` - Main auth handlers
- `/src/auth/jwt.go` - JWT token management
- `/src/auth/middleware.go` - Auth middleware

## Key Functions to Test
1. `CreateToken(user User) (string, error)`
2. `ValidateToken(token string) (*Claims, error)`
3. `RefreshToken(refreshToken string) (string, error)`

## Edge Cases Identified
- Expired tokens
- Malformed tokens
- Concurrent refresh attempts
- Rate limiting on login attempts
```

### Test Expert → DevOps Expert
```yaml
# File: .agent-artifacts/test-requirements-{feature}.yaml
---
source_agent: test-expert
target_agent: devops-expert
---

test_suites:
  - name: auth-unit-tests
    command: go test ./src/auth/...
    requirements:
      - Redis instance
      - Mock time support
    
  - name: auth-integration-tests  
    command: go test ./tests/integration/auth/...
    requirements:
      - Full API running
      - Test database
      - Redis instance

coverage_requirements:
  minimum: 80
  exclude_paths:
    - "*/mock_*.go"
    - "*/test_*.go"
```

## Orchestration Pattern

When Claude orchestrates multiple agents:

```python
# Conceptual flow (handled by Claude)
1. User Request → Claude
2. Claude → API Expert
   - Input: Requirements
   - Output: API spec file
3. Claude → Implementation Expert  
   - Input: Requirements + API spec file location
   - Output: Implementation files
4. Claude → Test Expert
   - Input: Implementation file locations + API spec
   - Output: Test files
5. Claude → User with summary
```

## Best Practices

### DO:
- Use descriptive file names
- Include metadata headers
- Reference specific file paths
- Document key decisions
- List all created/modified files

### DON'T:
- Assume state between calls
- Reference memory/variables
- Use relative paths without context
- Skip documentation
- Hide important details

## State Tracking via Backlog.md

The project-manager agent maintains workflow state:

```bash
# After each handoff
backlog task meta set TASK-001 last_handoff ".agent-artifacts/2024-01-15-api-expert-auth-spec.yaml"
backlog task meta set TASK-001 current_stage "implementation"
backlog task meta set TASK-001 next_agent "golang-expert"

# Query state
backlog task show TASK-001 --json | jq '.metadata'
```

## Error Recovery

If a handoff fails:

1. Check for artifact files in `.agent-artifacts/`
2. Review the last successful handoff
3. Use `backlog task show` to see current state
4. Re-run the failed agent with additional context

## Example Complete Workflow

```bash
# 1. API Design
User: "Design auth API"
Claude → api-expert
Output: .agent-artifacts/2024-01-15-api-expert-auth-spec.yaml

# 2. Implementation  
User: "Now implement it"
Claude → golang-expert (with API spec reference)
Output: Implementation files + .agent-artifacts/2024-01-15-golang-expert-implementation.md

# 3. Testing
User: "Add tests"  
Claude → test-expert (with implementation summary)
Output: Test files + .agent-artifacts/2024-01-15-test-expert-coverage.json

# 4. State visible via
backlog task show AUTH-001
ls -la .agent-artifacts/
```

## Summary

Successful handoffs require:
1. Clear file artifacts
2. Explicit file references
3. Metadata for traceability
4. Claude orchestration
5. Backlog.md for state tracking

This protocol ensures reliable communication between stateless agents while maintaining a clear audit trail of the development process.