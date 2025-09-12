# Claude Code Sub-Agent System Documentation

Welcome to the Claude Code sub-agent system documentation hub. This README serves as your primary navigation guide to all documentation resources.

## Quick Navigation

### Get Started Immediately
- **[Quick Start Guide](./quick-start.md)** - Get up and running in 5 minutes (START HERE)
- **[Getting Started](./getting-started.md)** - Detailed setup instructions

### Essential References
- **[Agent Overview](./reference/agent-overview.md)** - Complete agent list and selection guide
- **[Best Practices](./reference/best-practices.md)** - Guidelines for optimal usage
- **[Troubleshooting](./reference/troubleshooting.md)** - Common issues and solutions

### Implementation Guides
- **[Workflows](./guides/workflows.md)** - Step-by-step workflow patterns
- **[Handoff Protocol](./guides/handoff-protocol.md)** - Inter-agent communication standards

## Documentation Structure

```
agent-docs/
├── README.md                    # This navigation hub
├── quick-start.md              # 5-minute getting started guide
├── getting-started.md          # Detailed setup instructions
├── guides/                     # Implementation guides
│   ├── workflows.md           # Common workflow patterns
│   └── handoff-protocol.md    # Agent communication protocols
├── reference/                  # Detailed reference material
│   ├── agent-overview.md      # Complete agent list and selection
│   ├── best-practices.md      # Usage guidelines and patterns
│   ├── chain-of-draft-reasoning.md # CoD reasoning patterns
│   ├── performance-optimization.md # Performance tuning
│   └── troubleshooting.md     # Problem resolution
├── templates/                  # Reusable templates
│   └── compressed-templates.md # Agent template library
└── archive/                    # Historical documentation
    ├── OPTIMIZATION-REPORT.md
    ├── PERFORMANCE-OPTIMIZATION-SUMMARY.md
    └── ... [other archived docs]
```

## Common Tasks - Quick Links

### Getting Started
| Task | Resource | Description |
|------|----------|-------------|
| New user setup | [Quick Start](./quick-start.md) | 5-minute introduction to the system |
| Learn the basics | [Getting Started](./getting-started.md) | Comprehensive setup guide |
| Understand agents | [Agent Overview](./reference/agent-overview.md) | Complete agent reference |

### Daily Development
| Task | Resource | Description |
|------|----------|-------------|
| Create tasks | [Quick Start - Project Manager](./quick-start.md#project-manager) | Task creation with backlog commands |
| Build frontend | [Quick Start - TypeScript Expert](./quick-start.md#typescript-expert) | React/TypeScript implementation |
| Build backend | [Quick Start - Golang Expert](./quick-start.md#golang-expert) | Go service implementation |
| Add tests | [Quick Start - Test Expert](./quick-start.md#test-expert) | Testing strategy and automation |
| Security review | [Quick Start - Security Expert](./quick-start.md#security-expert) | Vulnerability assessment |

### Workflow Patterns
| Task | Resource | Description |
|------|----------|-------------|
| Feature development | [Workflows - Feature Implementation](./guides/workflows.md) | Complete feature workflow |
| Bug fixes | [Workflows - Bug Fix](./guides/workflows.md) | Bug resolution process |
| Architecture planning | [Workflows - Architecture](./guides/workflows.md) | System design workflow |
| Agent coordination | [Handoff Protocol](./guides/handoff-protocol.md) | Inter-agent communication |

### Troubleshooting
| Issue | Resource | Description |
|-------|----------|-------------|
| Agent selection | [Agent Overview - Decision Tree](./reference/agent-overview.md#agent-selection-decision-tree) | Which agent to use when |
| Workflow problems | [Troubleshooting](./reference/troubleshooting.md) | Common workflow issues |
| Performance issues | [Performance Optimization](./reference/performance-optimization.md) | System performance tuning |
| Best practices | [Best Practices](./reference/best-practices.md) | Guidelines and patterns |

## System Overview

The sub-agent system consists of 11 specialized agents organized into functional groups:

### Core System Agents
- **agent-manager** - System coordination and workflow orchestration
- **project-manager** - Task tracking and sprint management using Backlog.md
- **tech-writer** - Documentation creation and knowledge management

### Architecture & Design
- **architect-expert** - System design and technical planning
- **api-expert** - API design and interface contracts
- **project-optimizer** - Build optimization and project structure

### Implementation
- **typescript-expert** - Frontend React and TypeScript development
- **golang-expert** - Backend Go services and APIs

### Quality & Operations
- **test-expert** - Testing strategy and quality assurance
- **security-expert** - Security monitoring and vulnerability assessment
- **devops-expert** - CI/CD and infrastructure management

## Essential Concepts

### The Big 5 Agents
For 80% of tasks, you'll use these five agents:
1. **project-manager** - Creates and tracks tasks
2. **typescript-expert** - Frontend implementation
3. **golang-expert** - Backend implementation
4. **security-expert** - Security reviews (automatic monitoring)
5. **test-expert** - Testing and quality assurance

### Task Tool Usage
All agents are invoked using the Task tool:
```bash
Task(
  description="Brief description",
  prompt="Detailed instructions", 
  subagent_type="agent-name"
)
```

### Common Workflow Pattern
1. **project-manager** → Create and break down tasks
2. **architect-expert** → Design system architecture
3. **api-expert** → Define API contracts (if needed)
4. **Implementation agents** → Build features (parallel)
5. **test-expert** → Add comprehensive testing
6. **security-expert** → Security review (automatic)
7. **tech-writer** → Update documentation

## Performance Features

### Parallel Execution
- Frontend and backend development can run simultaneously
- Architecture and security reviews can happen in parallel
- Testing and documentation can be done concurrently

### Proactive Monitoring
- **security-expert** automatically monitors for security issues
- **agent-manager** coordinates workflows and resolves conflicts
- Performance optimization through batch operations and caching

### Chain-of-Draft (CoD) Reasoning
All agents use compressed 5-word reasoning steps for faster, more focused decision-making. See [Chain-of-Draft Reasoning](./reference/chain-of-draft-reasoning.md) for details.

## Advanced Topics

### For Experienced Users
- **[Performance Optimization](./reference/performance-optimization.md)** - Advanced performance tuning
- **[Chain-of-Draft Reasoning](./reference/chain-of-draft-reasoning.md)** - CoD implementation patterns
- **[Compressed Templates](./templates/compressed-templates.md)** - Reusable agent templates

### For System Administrators
- **[Handoff Protocol](./guides/handoff-protocol.md)** - Inter-agent communication standards
- **[Agent Overview - Boundaries](./reference/agent-overview.md#agent-boundaries-and-scope)** - Agent responsibility boundaries
- **[Performance Considerations](./reference/agent-overview.md#performance-considerations)** - System optimization

## Need Help?

### Step-by-step guidance
1. **New to the system?** Start with [Quick Start Guide](./quick-start.md)
2. **Need to choose an agent?** Use [Agent Overview - Decision Tree](./reference/agent-overview.md#agent-selection-decision-tree)
3. **Having workflow issues?** Check [Troubleshooting](./reference/troubleshooting.md)
4. **Want documentation?** Ask the **tech-writer** agent
5. **Complex coordination needed?** Use the **agent-manager** agent

### Ask the Agents
The agents themselves can provide help:

```bash
# Get documentation for any topic
Task(
  description="Explain authentication patterns",
  prompt="Explain how to implement JWT authentication using the agent system, including which agents to use and in what order.",
  subagent_type="tech-writer"
)

# Break down complex tasks
Task(
  description="Plan complex feature",
  prompt="I need to build a real-time dashboard with user authentication, data visualization, and live updates. Help me break this into tasks and coordinate the right agents.",
  subagent_type="project-manager"
)

# Coordinate multiple agents
Task(
  description="Coordinate feature development",
  prompt="I need to implement user profiles with photo upload, privacy settings, and audit logging. Please coordinate the appropriate agents for this feature.",
  subagent_type="agent-manager"
)
```

## Contributing

When contributing to this documentation:

1. **Review existing content** - Check current docs before adding new content
2. **Follow the structure** - Use the established directory organization
3. **Update navigation** - Add new content to this README's quick links
4. **Test thoroughly** - Verify all examples and workflows work
5. **Keep it current** - Update docs when agent behavior changes

## Archive

Historical documentation and implementation summaries are preserved in the [archive/](./archive/) directory for reference and learning from past optimizations.

---

**Remember**: The sub-agent system is designed to make development faster and more reliable through specialized expertise. Start with the [Quick Start Guide](./quick-start.md), then explore the detailed guides as you become more comfortable with the system.