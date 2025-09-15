# Handoff Agent System

A Redis-based handoff coordination system that modernizes agent-to-agent communication from file-based to real-time queue management.

## Overview

The Handoff Agent serves as the central coordinator for all inter-agent communication in the Claude Code ecosystem. It provides:

- **Real-time Queue Management**: Redis-based message queuing with priority handling
- **Intelligent Routing**: Content-based agent selection with configurable rules
- **Schema Validation**: Unified handoff schema enforcement and validation
- **System Monitoring**: Real-time metrics, alerting, and health tracking
- **Retry Logic**: Robust error handling with exponential backoff

## Quick Start

### Prerequisites
- Go 1.21 or later
- Redis server (docker-compose provided)

### Installation

1. Start Redis:
```bash
make docker-up
```

2. Build and run:
```bash
make run
```

Or run manually:
```bash
go build -o bin/handoff-agent ./cmd
./bin/handoff-agent -config config.json
```

### Configuration

Configuration is provided via JSON file:

```json
{
  "redis": {
    "addr": "localhost:6379",
    "db": 0
  },
  "logging": {
    "level": "info"
  },
  "agents": [
    {
      "name": "golang-expert",
      "queue_name": "handoff:project:my-project:queue:golang-expert",
      "max_concurrent": 3
    }
  ],
  "monitoring": {
    "enabled": true,
    "interval": "30s"
  }
}
```

## Architecture

### Core Components

- **HandoffAgent**: Main coordination service managing Redis queues
- **HandoffRouter**: Intelligent routing based on content analysis
- **HandoffValidator**: Schema validation and sanitization
- **HandoffMonitor**: System monitoring and alerting

### Data Flow

```
Publisher → Validation → Routing → Queue → Consumer → Processing → Completion
     ↓           ↓          ↓        ↓        ↓          ↓           ↓
  Schema    Agent Rules  Priority  Redis   Worker   Business    Metrics
 Checking   Evaluation  Ordering  Queue    Pool     Logic      Updates
```

## Usage Examples

### Publishing a Handoff

```go
handoff := &Handoff{
    Metadata: Metadata{
        ProjectName: "my-project", // Added for multi-project support
        FromAgent:   "api-expert",
        ToAgent:     "golang-expert", 
        TaskContext: "User authentication",
        Priority:    PriorityHigh,
    },
    Content: Content{
        Summary:      "Implement JWT authentication",
        Requirements: []string{"Create login endpoint", "Add JWT middleware"},
        TechnicalDetails: map[string]interface{}{
            "endpoints": []string{"/auth/login"},
        },
        NextSteps: []string{"Write tests", "Update docs"},
    },
}

err := agent.PublishHandoff(ctx, handoff)
```

### Consuming Handoffs

```go
handler := func(ctx context.Context, handoff *Handoff) error {
    // Process the handoff
    return processImplementation(ctx, handoff)
}

// Agents listen on their project-specific queue
err := agent.ConsumeHandoffs(ctx, "my-project", "golang-expert", handler)
```

### Intelligent Routing

```go
router := NewHandoffRouter("fallback-agent")

rule := RouteRule{
    Name:        "route-go-impl",
    TargetAgent: "golang-expert",
    Priority:    100,
    Conditions: []RouteCondition{
        {
            Type:     ConditionComplexQuery,
            Field:    "has_go_files", 
            Operator: "equals",
            Value:    true,
        },
    },
}

router.AddRoute("api-expert", rule)
```

## Monitoring & Alerts

The system provides comprehensive monitoring:

### Metrics
- Queue depth and processing times
- Success/failure rates 
- Active agent counts
- System health scores

### Alerts
- High queue depth (>50 items)
- Slow processing (>30s)
- High failure rates (>10%)
- Low system health (<50)

### Accessing Metrics

```go
metrics := agent.GetMetrics()
fmt.Printf("Queue depth: %d\n", metrics.QueueDepth)
fmt.Printf("Avg processing time: %v\n", metrics.AvgProcessingTime)
```

## Development

### Build Commands

```bash
make build      # Build binary
make test       # Run tests  
make lint       # Run linting
make run        # Start service
make clean      # Clean artifacts
make docker-up  # Start Redis
```

### Testing

```bash
# Unit tests
make test

# Integration tests (requires Redis)
make integration-test

# Benchmarks
make bench

# Coverage report
make coverage
```

## Schema Specification

### Unified Handoff Schema

```yaml
metadata:
  project_name: string       # Name of the project context
  from_agent: string       # Source agent name
  to_agent: string         # Target agent name  
  timestamp: datetime      # Creation timestamp
  task_context: string     # Task description
  priority: enum           # low|normal|high|critical
  handoff_id: string       # Unique identifier

content:
  summary: string          # Brief description
  requirements: string[]   # List of requirements
  artifacts:
    created: string[]      # Files created
    modified: string[]     # Files modified  
    reviewed: string[]     # Files reviewed
  technical_details: object # Agent-specific data
  next_steps: string[]     # Follow-up actions

validation:
  schema_version: string   # Schema version
  checksum: string         # Content checksum
```

### Agent-Specific Fields

Different agents use specific technical_details:

**golang-expert**:
```yaml
technical_details:
  handlers: string[]       # HTTP handlers
  services: string[]       # Service methods
  models: string[]         # Data models  
  test_coverage: number    # Coverage percentage
```

**typescript-expert**:
```yaml
technical_details:
  components: string[]     # React components
  hooks: string[]          # Custom hooks
  types: string[]          # TypeScript types
```

## Integration with Existing Agents

The handoff system integrates seamlessly with existing agents:

1. **Agent Registration**: Each agent registers its capabilities
2. **Queue Consumption**: Agents consume from their dedicated, project-specific queues
3. **Handoff Creation**: Agents create handoffs for other agents, including the project name
4. **Status Tracking**: All handoff status is tracked in Redis

### Agent Integration Pattern

```go
type Agent struct {
    handoffAgent *handoff.HandoffAgent
}

func (a *Agent) Start(ctx context.Context) error {
    // Register capabilities
    cap := handoff.AgentCapabilities{
        Name: "my-agent",
        QueueName: "handoff:project:my-project:queue:my-agent",
        MaxConcurrent: 3,
    }
    a.handoffAgent.RegisterAgent(cap)
    
    // Start consuming
    return a.handoffAgent.ConsumeHandoffs(ctx, "my-project", "my-agent", a.processHandoff)
}
```

## Configuration Reference

### Redis Configuration
- `addr`: Redis server address 
- `password`: Redis password (optional)
- `db`: Redis database number

### Agent Configuration
- `name`: Agent identifier
- `description`: Agent description
- `queue_name`: Redis queue name (e.g., `handoff:project:my-project:queue:my-agent`)
- `max_concurrent`: Max concurrent processors

### Routing Configuration
- `name`: Rule name
- `target_agent`: Target agent for routing
- `priority`: Rule priority (higher = more important)
- `conditions`: List of routing conditions

### Alert Configuration
- `name`: Alert rule name
- `type`: Alert type (queue_depth, failure_rate, etc.)
- `condition`: Comparison operator
- `threshold`: Alert threshold value
- `enabled`: Whether alert is active
- `cooldown`: Minimum time between alerts

## Troubleshooting

### Common Issues

**Redis Connection Failed**
```bash
# Check Redis is running
make docker-up
# Or manually
docker-compose up redis
```

**High Queue Depth**

```bash
# Check consumer health for a specific project
redis-cli ZCARD handoff:project:my-project:queue:agent-name
# Restart consumers or scale up
```

**Processing Failures**

```bash
# Check error logs
grep ERROR /var/log/handoff-agent.log
# Review retry configuration
```

### Debugging

Enable debug logging:
```bash
./bin/handoff-agent -log-level debug
```

Check Redis queues:

```bash
redis-cli KEYS "handoff:project:*"
redis-cli ZRANGE handoff:project:my-project:queue:golang-expert 0 -1
```


## Performance Tuning

### Redis Optimization
- Use connection pooling
- Set appropriate timeouts
- Configure memory limits

### Queue Management
- Adjust max_concurrent per agent
- Tune priority scoring
- Monitor queue depths

### System Resources
- Monitor CPU and memory usage
- Scale horizontally with multiple instances
- Use Redis clustering for high availability

## Contributing

1. Fork the repository
2. Create feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `make ci`
5. Submit pull request

## License

MIT License - see LICENSE file for details.