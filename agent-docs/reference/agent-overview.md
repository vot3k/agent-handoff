# Agent System Overview

A comprehensive reference for all available agents in the system and guidance on agent selection and interaction patterns.

## Complete Agent List

The system includes 11 specialized agents, each with distinct responsibilities and trigger conditions:

### Core System Agents

#### 1. **agent-manager**
- **Purpose**: Agent system coordination, workflow orchestration, and inter-agent communication
- **Key Responsibilities**: Agent registry management, workflow design, handoff protocols
- **When to Use**: Agent conflicts, workflow optimization, system-wide coordination needs
- **Tools**: Read, Write, LS

#### 2. **project-manager** 
- **Purpose**: Task and sprint management using Backlog.md CLI system
- **Key Responsibilities**: Task tracking, sprint planning, progress monitoring, state management
- **When to Use**: Task creation/management, sprint planning, progress tracking, backlog organization
- **Tools**: Read, Write, LS, Bash
- **Important**: Uses standard CLI commands (`backlog task create`), NOT slash commands

#### 3. **tech-writer**
- **Purpose**: Expert technical documentation and knowledge management
- **Key Responsibilities**: README files, API documentation, user guides, architecture docs
- **When to Use**: Documentation creation/updates, user guides, knowledge base management
- **Tools**: Read, Write, LS, Bash

### Architecture & Design Agents

#### 4. **architect-expert**
- **Purpose**: System architecture, design patterns, and technical planning
- **Key Responsibilities**: System design, architectural decisions, technical standards, cross-cutting concerns
- **When to Use**: System design, technology selection, architectural reviews, performance bottlenecks
- **Tools**: Read, Write, LS, Bash

#### 5. **api-expert**
- **Purpose**: API design, REST principles, and protocol implementation
- **Key Responsibilities**: REST/GraphQL/gRPC design, data modeling, protocol standards
- **When to Use**: API design, interface contracts, versioning strategies, rate limiting
- **Tools**: Read, Write, LS, Bash

#### 6. **project-optimizer**
- **Purpose**: Project configuration, build optimization, and development workflow automation
- **Key Responsibilities**: Project structure, build systems, performance optimization, dependency management
- **When to Use**: Build optimization, project structure, bundle size issues, development workflow improvements
- **Tools**: Read, Write, LS, Bash

### Implementation Agents

#### 7. **typescript-expert**
- **Purpose**: TypeScript/React frontend implementation specialist
- **Key Responsibilities**: Type-safe React code, component development, state management, frontend patterns
- **When to Use**: Frontend implementation, React components, TypeScript conversion, UI development
- **Tools**: Read, Write, LS, Bash
- **Important**: MUST write actual code implementation, not just plans

#### 8. **golang-expert**
- **Purpose**: Go backend implementation specialist  
- **Key Responsibilities**: Idiomatic Go code, backend services, API endpoints, database integration
- **When to Use**: Backend implementation, Go services, API handlers, server-side logic
- **Tools**: Read, Write, LS, Bash
- **Important**: MUST write actual code implementation, not just plans

### Quality & Operations Agents

#### 9. **test-expert**
- **Purpose**: Test strategy, automation, and quality assurance
- **Key Responsibilities**: Unit/integration/E2E tests, coverage analysis, test automation, quality metrics
- **When to Use**: Test implementation, coverage analysis, QA strategy, test automation setup
- **Tools**: Read, Write, LS, Bash

#### 10. **security-expert**
- **Purpose**: Web application security and vulnerability assessment (proactive monitoring)
- **Key Responsibilities**: Security reviews, vulnerability scanning, secure coding practices, compliance
- **When to Use**: Security audits, vulnerability assessment, secure architecture review
- **Tools**: Read, Write, LS, Bash
- **Unique**: Proactively monitors development for security issues

#### 11. **devops-expert**
- **Purpose**: CI/CD, deployment automation, and infrastructure management
- **Key Responsibilities**: Pipeline design, containerization, Kubernetes, monitoring, deployment flows
- **When to Use**: Deployment setup, CI/CD configuration, infrastructure automation, monitoring
- **Tools**: Read, Write, LS, Bash

## Agent Selection Decision Tree

### Is this a system design or architecture question?
- **Yes** → Use **architect-expert**
  - Follow up with **api-expert** for interface design
  - Then **security-expert** for security architecture

### Is this about API design or protocols?
- **Yes** → Use **api-expert**
  - For REST/GraphQL/gRPC specifications
  - Interface contracts and data modeling

### Is this about project management or task tracking?
- **Yes** → Use **project-manager**
  - Task creation, sprint planning, progress tracking
  - Remember: Use CLI commands, not slash commands

### Is this about implementation?

#### Frontend Implementation Needed?
- **Yes** → Use **typescript-expert**
  - React components, TypeScript code, UI logic
  - State management, frontend patterns

#### Backend Implementation Needed?
- **Yes** → Use **golang-expert**
  - Go services, API handlers, database code
  - Backend business logic, server implementation

### Is this about quality assurance?

#### Testing Related?
- **Yes** → Use **test-expert**
  - Test strategy, automation, coverage
  - Unit, integration, E2E tests

#### Security Concerns?
- **Yes** → Use **security-expert**
  - Security reviews, vulnerability assessment
  - Proactive security monitoring

### Is this about deployment or infrastructure?
- **Yes** → Use **devops-expert**
  - CI/CD pipelines, containerization
  - Kubernetes, monitoring, deployment

### Is this about documentation?
- **Yes** → Use **tech-writer**
  - README files, API docs, user guides
  - Technical documentation, knowledge management

### Is this about build optimization or project structure?
- **Yes** → Use **project-optimizer**
  - Build performance, bundle optimization
  - Project structure, development workflows

### Is this about agent coordination or workflow issues?
- **Yes** → Use **agent-manager**
  - Agent conflicts, workflow optimization
  - Inter-agent communication issues

## Agent Interaction Patterns

### Sequential Workflows

#### Feature Development Flow
```
project-manager → architect-expert → api-expert → security-expert →
[typescript-expert + golang-expert (parallel)] → test-expert → devops-expert → tech-writer
```

#### Security Review Flow
```
security-expert (proactive) → [implementation-agent] → security-expert (validation) → devops-expert
```

### Parallel Patterns

#### Implementation Phase
- **typescript-expert** and **golang-expert** can work simultaneously
- Both receive handoffs from **api-expert** and **security-expert**
- Both hand off to **test-expert** when complete

#### Architecture Phase
- **architect-expert** and **security-expert** can review requirements in parallel
- **api-expert** waits for architecture decisions
- **project-optimizer** can work on build setup in parallel

### Proactive Monitoring

#### security-expert
- Continuously monitors code changes for security issues
- Automatically triggers reviews for auth-related changes
- Provides early security guidance

#### agent-manager
- Monitors agent performance and handoff success
- Detects workflow inefficiencies
- Resolves agent conflicts

## Agent Boundaries and Scope

### Clear Separations

#### Implementation Agents
- **typescript-expert**: Frontend only, no backend code
- **golang-expert**: Backend only, no frontend code
- **test-expert**: Testing only, no feature implementation

#### Design Agents  
- **architect-expert**: High-level design, not implementation details
- **api-expert**: Interface contracts, not implementation
- **project-optimizer**: Build systems, not application logic

#### Support Agents
- **tech-writer**: Documentation only, not code implementation
- **devops-expert**: Deployment/infrastructure, not application code
- **security-expert**: Security review, not feature development

### Common Overlaps to Avoid

#### Documentation Boundaries
- **tech-writer**: User-facing documentation, API docs, guides
- **Implementation agents**: Code comments and inline documentation only

#### Architecture Boundaries
- **architect-expert**: System design, patterns, technology choices
- **api-expert**: Interface design, protocols, data contracts
- **Implementation agents**: Tactical implementation decisions only

#### Testing Boundaries
- **test-expert**: Test strategy, automation, quality assurance
- **Implementation agents**: Unit tests for their own code only

## Handoff Protocol

All agents use a unified handoff schema:

```yaml
handoff_schema:
  metadata:
    from_agent: string
    to_agent: string  
    timestamp: ISO8601
    task_context: string
    priority: high|medium|low
  
  content:
    summary: string
    requirements: string[]
    artifacts:
      created: string[]
      modified: string[]
      reviewed: string[]
    technical_details: object  # Agent-specific details
    next_steps: string[]
  
  validation:
    schema_version: "1.0"
    checksum: string
```

### File-Based Communication

Agents communicate through files in `.claude/handoffs/` using the format:
```
[timestamp]-[from-agent]-to-[to-agent].md
```

### Context Preservation

- Each handoff includes full context from previous work
- Artifacts section tracks all created/modified files
- Technical details provide agent-specific information
- Next steps guide the receiving agent

## Performance Considerations

### Batch Operations
- Process multiple similar tasks together
- Use parallel execution for independent operations
- Cache frequently accessed data

### Parallel Execution
- Identify independent work streams
- Execute non-conflicting agents simultaneously
- Coordinate through proper handoff protocols

### Caching Strategies
- Cache parsed configurations and analysis results
- Store frequently accessed metadata
- Implement TTL-based invalidation

## Common Anti-Patterns

### Agent Misuse

#### Using Wrong Agent for Task
- **Don't**: Ask **tech-writer** to implement code
- **Don't**: Ask **typescript-expert** to write Go code
- **Don't**: Ask **golang-expert** to design APIs

#### Bypassing Proper Workflow
- **Don't**: Skip architecture phase for complex features
- **Don't**: Implement without security review
- **Don't**: Deploy without testing validation

#### Poor Handoff Practices
- **Don't**: Incomplete context in handoffs
- **Don't**: Missing technical requirements
- **Don't**: Unclear next steps for receiving agent

### Workflow Anti-Patterns

#### Sequential Over-Processing
- **Don't**: Force sequential execution when parallel is possible
- **Do**: Identify independent work streams
- **Do**: Maximize parallel agent utilization

#### Inefficient Agent Selection
- **Don't**: Use generalist approach when specialist exists
- **Do**: Match agent expertise to task requirements
- **Do**: Follow the decision tree for agent selection

## Best Practices

### Agent Selection
1. Use the decision tree to identify the most appropriate agent
2. Consider whether work can be parallelized across multiple agents
3. Ensure proper sequencing for dependent work
4. Verify agent capabilities match task requirements

### Workflow Design
1. Start with **project-manager** for task breakdown
2. Use **architect-expert** for system design decisions
3. Implement security reviews early with **security-expert**
4. Parallelize implementation with **typescript-expert** and **golang-expert**
5. Complete with **test-expert**, **devops-expert**, and **tech-writer**

### Quality Assurance
1. Always include **test-expert** in implementation workflows
2. Use **security-expert** for proactive security monitoring
3. Ensure **devops-expert** reviews deployment requirements
4. Have **tech-writer** document all major features

### Performance Optimization
1. Identify parallelizable work streams
2. Use batch operations for similar tasks
3. Implement proper caching strategies
4. Monitor agent performance and handoff success rates

## Monitoring and Optimization

### Key Metrics
- Agent activation success rate
- Handoff completion rate
- Workflow execution time
- Parallel efficiency ratio

### Common Issues
- Agent boundary violations
- Incomplete handoffs
- Missing context preservation
- Sequential processing of parallelizable work

### Resolution Patterns
- Use **agent-manager** for system-wide issues
- Implement proper validation in handoff protocols
- Establish clear agent boundaries and responsibilities
- Design workflows for maximum parallel execution

This overview serves as the definitive reference for understanding and effectively utilizing the agent system. Use this guide to select the right agents, design efficient workflows, and ensure proper collaboration between specialized agents.