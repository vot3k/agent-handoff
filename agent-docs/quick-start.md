# Quick Start Guide

Get up and running with the Claude Code sub-agent system in 5 minutes.

## What Are Sub-Agents?

The sub-agent system consists of specialized AI agents that work together on different aspects of software development. Instead of doing everything yourself, you coordinate with experts who handle their specific domains.

## Essential Agents (The Big 5)

These are the agents you'll use most often:

### project-manager
**What it does**: Creates and tracks tasks using the Backlog.md system  
**When to use**: "Create a task for...", "Track progress on...", "Plan the sprint..."  
**Key command**: `backlog task create "Title" -d "Description" --ac "AC1,AC2"`

### typescript-expert  
**What it does**: Implements frontend React code and TypeScript applications  
**When to use**: "Build a login component", "Add TypeScript types", "Fix frontend bug"  
**Strength**: Type-safe React development, state management, API integration

### golang-expert
**What it does**: Implements backend services, APIs, and business logic  
**When to use**: "Create REST endpoints", "Add database layer", "Implement auth service"  
**Strength**: High-performance backend code, microservices, API development

### security-expert
**What it does**: Reviews code for vulnerabilities and security best practices  
**When to use**: Runs automatically during development, or "Review security of..."  
**Strength**: Proactive monitoring, vulnerability detection, secure coding patterns

### test-expert
**What it does**: Creates test strategies, automation, and quality assurance  
**When to use**: "Add tests for...", "Set up test automation", "Check test coverage"  
**Strength**: Unit, integration, and E2E testing across the full stack

## 5-Minute Setup

### 1. Understand the Task Tool (30 seconds)

All agents are invoked using the Task tool:

```bash
Task(
  description="Brief description of what you want",
  prompt="Detailed instructions for the agent", 
  subagent_type="agent-name"
)
```

### 2. Create Your First Task (2 minutes)

Let's create a simple authentication feature:

```bash
# Step 1: Create the task
Task(
  description="Create auth task",
  prompt="Create a new high-priority task for implementing JWT user authentication. Include login endpoint, token management, and secure storage.",
  subagent_type="project-manager"
)

# Step 2: Implement the backend
Task(
  description="Build auth backend", 
  prompt="Implement JWT authentication system with login endpoint, token validation middleware, and user service using secure patterns.",
  subagent_type="golang-expert"
)

# Step 3: Build the frontend
Task(
  description="Build auth frontend",
  prompt="Create a login form component with validation, JWT token management, and protected routes using TypeScript.",
  subagent_type="typescript-expert"
)
```

### 3. Let the System Coordinate (1 minute)

For complex tasks, let the agent-manager coordinate multiple agents:

```bash
Task(
  description="Coordinate feature development",
  prompt="I need to build a user dashboard with authentication, data visualization, and real-time updates. Please coordinate the appropriate agents to design and implement this feature.",
  subagent_type="agent-manager"
)
```

### 4. Get Help Documentation (30 seconds)

The tech-writer agent can explain anything:

```bash
Task(
  description="Document auth system",
  prompt="Create API documentation for the authentication endpoints we just built, including code examples and error handling.",
  subagent_type="tech-writer"
)
```

### 5. Review and Test (1 minute)

Always wrap up with testing and security:

```bash
# Add comprehensive tests
Task(
  description="Test auth system",
  prompt="Create unit and integration tests for the authentication system, covering login flow, token validation, and error cases.",
  subagent_type="test-expert"
)

# Security review (often happens automatically)
Task(
  description="Security review",
  prompt="Review the authentication implementation for security vulnerabilities and recommend improvements.",
  subagent_type="security-expert"
)
```

## Common Workflow Patterns

### Pattern 1: Feature Development
1. **project-manager**: Create and break down the task
2. **architect-expert**: Design the system architecture  
3. **api-expert**: Define API contracts (if needed)
4. **golang-expert** + **typescript-expert**: Implement in parallel
5. **test-expert**: Add comprehensive testing
6. **security-expert**: Security review (automatic)
7. **tech-writer**: Update documentation

### Pattern 2: Bug Fix
1. **project-manager**: Create bug task with reproduction steps
2. **Relevant expert** (typescript/golang): Investigate and fix
3. **test-expert**: Add regression tests
4. **tech-writer**: Update docs if needed

### Pattern 3: Architecture Planning  
1. **architect-expert**: High-level system design
2. **security-expert**: Security architecture review
3. **api-expert**: API design and contracts
4. **project-manager**: Break into implementation tasks

## Tips for Success

### DO:
- **Be specific**: "Create a login form with email validation" vs "Add auth"
- **Include context**: Mention existing files, patterns, requirements
- **Follow the flow**: project-manager → design → implement → test → document
- **Let security-expert monitor**: It runs proactively during development

### DON'T:
- **Mix concerns**: Don't ask typescript-expert to do backend work
- **Skip testing**: Always involve test-expert for quality code
- **Forget documentation**: tech-writer keeps everything clear
- **Rush security**: Let security-expert review critical features

## Quick Commands Reference

| Action | Agent | Example Command |
|--------|-------|-----------------|
| Create task | project-manager | `backlog task create "User auth" -d "JWT login system"` |
| View tasks | project-manager | `backlog task list --plain` |
| Build frontend | typescript-expert | `Task(..., subagent_type="typescript-expert")` |
| Build backend | golang-expert | `Task(..., subagent_type="golang-expert")` |
| Add tests | test-expert | `Task(..., subagent_type="test-expert")` |
| Write docs | tech-writer | `Task(..., subagent_type="tech-writer")` |

## Your First Real Task

Try this complete workflow to build a user registration system:

```bash
# 1. Plan the work
Task(
  description="Plan user registration",
  prompt="Create a task for building user registration with email validation, password requirements, and email confirmation. Break it into atomic tasks if needed.",
  subagent_type="project-manager"
)

# 2. Design the system
Task(
  description="Design registration system", 
  prompt="Design the architecture for user registration including database schema, API endpoints, email service integration, and security considerations.",
  subagent_type="architect-expert"
)

# 3. Build it (agents can work in parallel)
Task(
  description="Implement registration API",
  prompt="Implement user registration API with email validation, password hashing, email confirmation workflow, and proper error handling.",
  subagent_type="golang-expert"
)

Task(
  description="Build registration UI",
  prompt="Create a user registration form with real-time validation, password strength indicator, and email confirmation handling.",
  subagent_type="typescript-expert"
)

# 4. Ensure quality
Task(
  description="Test registration flow",
  prompt="Create comprehensive tests for user registration including form validation, API endpoints, email workflows, and error scenarios.",
  subagent_type="test-expert"
)
```

## What's Next?

Once you're comfortable with the basics:

- **Read the detailed guides**: [Workflows](./guides/workflows.md), [Best Practices](./reference/best-practices.md)
- **Explore all agents**: See the full [README](./README.md) for the complete agent list
- **Learn advanced patterns**: Check [Performance Optimization](./reference/performance-optimization.md)
- **Troubleshoot issues**: Visit [Troubleshooting](./reference/troubleshooting.md)

## Get Help

The agents are designed to be helpful and educational. If you're unsure:

1. **Ask tech-writer** to explain any concept
2. **Use project-manager** to break down complex tasks  
3. **Let agent-manager** coordinate when you're overwhelmed
4. **Check the documentation** in this folder for detailed guides

Remember: The sub-agent system is designed to make development faster and more reliable by leveraging specialized expertise. Start simple, then gradually use more advanced patterns as you get comfortable.