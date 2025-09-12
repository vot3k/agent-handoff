# Getting Started

This guide helps you start using the sub-agent system effectively.

## Initial Setup

### 1. Understanding Agents
Each agent has specific expertise:

- **agent-manager**: Coordinates all agents
- **project-manager**: Tracks tasks and progress
- **project-optimizer**: Handles optimization
- **tech-writer**: Creates documentation
- **security-expert**: Ensures security
- **typescript-expert**: Frontend development
- **golang-expert**: Backend development
- **api-expert**: API design
- **test-expert**: Testing strategy
- **devops-expert**: Deployment and infrastructure

### 2. Basic Workflow
1. Create task with project-manager
2. Let agent-manager coordinate
3. Specialists handle their parts
4. Review and validate results
5. Update documentation

## First Task

### 1. Task Creation

When creating a task, use natural language to describe it to the project-manager. For example:

```
Can you create a task for implementing user authentication? It's a high priority feature that needs:
- JWT-based authentication
- Login endpoint implementation
- Token management system
- Secure storage for credentials
```

The project-manager will structure this information internally using the Backlog.md format.

Alternatively, you can be more direct:

```
Create a new high-priority task: Implement user authentication system with JWT, including login endpoint, token management, and secure storage.
```

Key points to include in your prompt:
- Task type (feature, bug, etc.)
- Priority level
- Clear description
- Key requirements
- Any dependencies or constraints

### 2. Using Sub-agents

To invoke a specific sub-agent, use the Task tool with the appropriate subagent_type. For example:

```
Task(
  description="Create auth task",
  prompt="Create a new task for implementing JWT user authentication",
  subagent_type="project-manager"
)
```

Or for technical implementation:

```
Task(
  description="Implement auth",
  prompt="Implement the JWT authentication system using the standard auth library",
  subagent_type="typescript-expert"
)
```

#### Available Tools

Each agent has access to specific tools. Here are the valid tools in Claude Code:

- **Bash**: Execute shell commands (requires permission)
- **Edit**: Make targeted file edits (requires permission)
- **Glob**: Find files by pattern matching
- **Grep**: Search file contents
- **LS**: List files and directories
- **MultiEdit**: Perform multiple file edits atomically (requires permission)
- **NotebookEdit**: Modify Jupyter notebook cells (requires permission)
- **NotebookRead**: Read Jupyter notebook contents
- **Read**: Read file contents
- **Task**: Run complex multi-step tasks with sub-agents
- **TodoWrite**: Create structured task lists
- **WebFetch**: Fetch URL content (requires permission)
- **WebSearch**: Perform web searches with domain filtering (requires permission)
- **Write**: Create or overwrite files (requires permission)

Note: Some tools require permission to use. Permission rules can be configured using `/allowed-tools` or in permission settings.

Key points for using sub-agents:
- Select the right specialist for the task
- Provide clear, detailed prompts
- Include relevant context
- Specify any constraints or requirements
- Let agent-manager coordinate between specialists

The agent-manager can also coordinate automatically by:
1. Analyzing requirements
2. Selecting needed specialists
3. Creating workflow plan
4. Managing coordination

### 3. Execution
Typical flow:
1. API design (api-expert)
2. Implementation (language experts)
3. Security review (security-expert)
4. Testing (test-expert)
5. Deployment (devops-expert)
6. Documentation (tech-writer)

## Best Practices

### Task Management
- Clear requirements
- Complete context
- Set priorities
- Track progress
- Document changes

### Communication
- Use standard format
- Include context
- Clear handoffs
- Regular updates
- Good documentation

### Quality
- Write tests
- Security review
- Performance check
- Update docs
- Regular validation

## Common Patterns

### Feature Development
1. Requirements gathering
2. Design phase
3. Implementation
4. Testing
5. Deployment
6. Documentation

### Bug Fixes
1. Investigation
2. Root cause
3. Fix implementation
4. Validation
5. Deployment
6. Documentation

### Performance Optimization
1. Analysis
2. Planning
3. Implementation
4. Testing
5. Deployment
6. Documentation

## Tips for Success

### DO:
- Follow workflows
- Document everything
- Complete reviews
- Test thoroughly
- Update docs

### DON'T:
- Skip steps
- Rush handoffs
- Ignore issues
- Skip testing
- Forget docs

## Next Steps

1. Review documentation
2. Start small task
3. Follow workflow
4. Review results
5. Learn patterns

## Getting Help

### Documentation
- README.md: Overview
- workflows.md: Standard processes
- best-practices.md: Guidelines
- troubleshooting.md: Issue resolution

### Support
- Report issues
- Ask questions
- Share feedback
- Suggest improvements
- Contribute updates