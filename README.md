# Agent Handoff System

A Redis-based agent orchestration system that enables sophisticated inter-agent communication and task management for Claude Code workflows.

## Overview

The Agent Handoff System consists of two complementary components:

1. **Handoff Library** (`handoff/`) - A sophisticated Redis-based coordination system with intelligent routing, monitoring, and validation
2. **Agent Manager** (`agent-manager/`) - A lightweight orchestrator that executes agents based on handoff messages

Together, they provide real-time queue management, intelligent routing, schema validation, and comprehensive monitoring for agent-to-agent workflows.

## Quick Start

### Prerequisites

- Go 1.21 or later
- Redis server
- Docker and Docker Compose (recommended)

### 1. Start Redis

```bash
docker-compose up -d redis
```

This starts a Redis container on `localhost:6379` with persistent storage.

### 2. Run the Agent Manager

```bash
cd agent-manager
go run main.go
```

The Agent Manager will start listening to all configured agent queues and automatically execute agents when handoffs are received.

### 3. Test with a Handoff

In another terminal, publish a test handoff:

```bash
cd agent-manager
go run test-publisher.go architect-expert api-expert "Design authentication system"
```

You should see the Agent Manager pick up the handoff and execute the `api-expert` agent.

## System Architecture

### Components Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Publisher     │    │  Handoff Lib    │    │ Agent Manager   │
│                 │───▶│                 │───▶│                 │
│ - Agents        │    │ - Validation    │    │ - Queue Monitor │
│ - External API  │    │ - Routing       │    │ - Agent Exec    │
│ - CLI Tools     │    │ - Monitoring    │    │ - Archival      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                       │
                                ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │     Redis       │    │   run-agent.sh  │
                       │                 │    │                 │
                       │ - Message Queue │    │ - Bridge Script │
                       │ - Metadata      │    │ - Agent Wrapper │
                       │ - Monitoring    │    │ - Error Handling│
                       └─────────────────┘    └─────────────────┘
```

### Data Flow

1. **Handoff Creation**: Agents or external systems create handoff messages
2. **Validation**: Handoff library validates schema and content
3. **Routing**: Messages are routed to appropriate agent queues
4. **Queuing**: Redis stores messages with priority ordering
5. **Processing**: Agent Manager monitors queues and dispatches work
6. **Execution**: Agents are executed via the bridge script
7. **Archival**: Completed handoffs are archived to filesystem

## Installation

### Using Docker Compose

```bash
# Clone the repository
git clone <repository-url>
cd agent-handoff-system

# Start Redis
docker-compose up -d

# Build and run Agent Manager
cd agent-manager
go build -o agent-manager main.go
./agent-manager
```

### Manual Installation

```bash
# Install Redis
# macOS: brew install redis
# Ubuntu: apt-get install redis-server

# Start Redis
redis-server

# Build components
cd handoff && go build ./cmd/main.go
cd ../agent-manager && go build main.go

# Run Agent Manager
./agent-manager
```

## Configuration

### Agent Manager

The Agent Manager uses environment variables for configuration:

```bash
export REDIS_ADDR="localhost:6379"  # Redis connection string
```

As of the latest update, the Agent Manager in `main.go` no longer uses a static list of queues. It dynamically discovers and listens on all project-specific queues that match the pattern `handoff:project:*:queue:*`. This allows it to handle multiple projects concurrently without manual configuration for each new project.

### Handoff Library

Configuration via JSON file:

```json
{
  "redis": {
    "addr": "localhost:6379",
    "db": 0
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

## Usage Examples

### Publishing Handoffs

Using the test publisher:

```bash
cd agent-manager
go run test-publisher.go <from_agent> <to_agent> [message]

# Examples:
go run test-publisher.go architect-expert golang-expert "Implement user service"
go run test-publisher.go api-expert test-expert "Create authentication tests"
go run test-publisher.go devops-expert security-expert "Security audit needed"
```

### Programmatic Handoff Creation

```go
package main

import (
    "context"
    "encoding/json"
    "time"
    "github.com/go-redis/redis/v8"
)

type HandoffPayload struct {
    Metadata struct {
        ProjectName string    `json:"project_name"`
        FromAgent   string    `json:"from_agent"`
        ToAgent     string    `json:"to_agent"`
        Timestamp   time.Time `json:"timestamp"`
        TaskContext string    `json:"task_context"`
        Priority    string    `json:"priority"`
        HandoffID   string    `json:"handoff_id"`
    } `json:"metadata"`
    Content struct {
        Summary          string                 `json:"summary"`
        Requirements     []string               `json:"requirements"`
        Artifacts        map[string][]string    `json:"artifacts"`
        TechnicalDetails map[string]interface{} `json:"technical_details"`
        NextSteps        []string               `json:"next_steps"`
    } `json:"content"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

func publishHandoff(rdb *redis.Client, handoff HandoffPayload) error {
    ctx := context.Background()
    
    // Serialize handoff
    payload, err := json.Marshal(handoff)
    if err != nil {
        return err
    }
    
    // Store in Redis with TTL
    handoffKey := fmt.Sprintf("handoff:%s", handoff.Metadata.HandoffID)
    err = rdb.Set(ctx, handoffKey, payload, 24*time.Hour).Err()
    if err != nil {
        return err
    }
    
    // Add to priority queue for a specific project
    queueName := fmt.Sprintf("handoff:project:%s:queue:%s", handoff.Metadata.ProjectName, handoff.Metadata.ToAgent)
    score := 3.0 + float64(time.Now().UnixNano())/1e18 // Priority + timestamp
    
    return rdb.ZAdd(ctx, queueName, &redis.Z{
        Score:  score,
        Member: handoff.Metadata.HandoffID,
    }).Err()
}
```

### Agent Integration Pattern

To integrate with the handoff system, agents should:

1. **Listen to their queue**: Monitor `handoff:queue:agent-name`
2. **Process handoffs**: Implement business logic
3. **Update status**: Mark handoffs as completed/failed
4. **Create new handoffs**: Chain to other agents as needed

Example agent integration:

```bash
#!/bin/bash
# Custom agent script

HANDOFF_JSON="$1"
HANDOFF_ID=$(echo "$HANDOFF_JSON" | jq -r '.metadata.handoff_id')

# Process the handoff
echo "Processing handoff $HANDOFF_ID..."

# Your agent logic here
# - Parse requirements
# - Execute tasks  
# - Generate artifacts
# - Create output

# Success
echo "✅ Agent completed successfully"
exit 0
```

## Handoff Schema

### Standard Schema

```yaml
metadata:
  project_name: string       # Name of the project context
  from_agent: string          # Source agent identifier
  to_agent: string            # Target agent identifier  
  timestamp: datetime         # ISO8601 timestamp
  task_context: string        # Brief task description
  priority: enum              # low|normal|high|critical
  handoff_id: string          # Unique handoff identifier

content:
  summary: string             # Task summary
  requirements: string[]      # List of requirements
  artifacts:
    created: string[]         # Files created
    modified: string[]        # Files modified
    reviewed: string[]        # Files reviewed
  technical_details: object   # Agent-specific data
  next_steps: string[]        # Follow-up actions

status: string                # pending|processing|completed|failed
created_at: datetime          # Creation timestamp
updated_at: datetime          # Last update timestamp
```

### Agent-Specific Technical Details

**golang-expert**:
```yaml
technical_details:
  packages: string[]          # Go packages to create
  handlers: string[]          # HTTP handlers needed
  models: string[]            # Data structures
  test_coverage: number       # Target coverage %
```

**typescript-expert**:
```yaml
technical_details:
  components: string[]        # React components
  hooks: string[]             # Custom hooks
  types: string[]             # TypeScript definitions
  api_integration: boolean    # Needs API calls
```

**devops-expert**:
```yaml
technical_details:
  containers: string[]        # Docker containers
  services: string[]          # Kubernetes services
  environments: string[]      # Target environments
  monitoring: boolean         # Add monitoring
```

## Monitoring and Observability

### Queue Monitoring

Check queue depths for a specific project:

```bash
redis-cli ZCARD handoff:project:my-project:queue:golang-expert
redis-cli ZCARD handoff:project:my-project:queue:api-expert
```

View queued handoffs:

```bash
redis-cli ZRANGE handoff:project:my-project:queue:golang-expert 0 -1 WITHSCORES
```

### Archive Analysis

Completed handoffs are archived to `agent-manager/archive/` in project-specific directories:

```
archive/
└── my-project/
    ├── 2024-01-15/
    │   ├── 20240115T143022Z-api-expert-abc12345.json
    │   ├── 20240115T143545Z-golang-expert-def67890.json
    │   └── 20240115T144012Z-test-expert-ghi13579.json
    └── 2024-01-16/
        └── ...
```

Each file contains the complete handoff payload for audit and debugging.

### System Health

Monitor Agent Manager logs:

```bash
# Watch real-time processing
tail -f agent-manager.log

# Check for errors
grep ERROR agent-manager.log

# Monitor throughput
grep "SUCCESS" agent-manager.log | wc -l
```

## Development and Testing

### Running Tests

```bash
# Handoff library tests
cd handoff
go test ./...

# Agent Manager tests  
cd agent-manager
go test ./...

# Integration tests (requires Redis)
docker-compose up -d redis
go test -tags=integration ./...
```

### Development Workflow

1. **Start Redis**: `docker-compose up -d redis`
2. **Run Agent Manager**: `cd agent-manager && go run main.go`
3. **Test with handoffs**: `go run test-publisher.go agent1 agent2 "test message"`
4. **Check archives**: `ls agent-manager/archive/$(date +%Y-%m-%d)/`

### Adding New Agents

1. Add agent queue to `main.go`:
   ```go
   queues := []string{
       // existing agents...
       "handoff:queue:my-new-agent",
   }
   ```

2. Add case to `run-agent.sh`:
   ```bash
   "my-new-agent")
       echo "🤖 My New Agent: Processing..."
       # Your agent logic here
       echo "✅ My New Agent: Completed"
       ;;
   ```

3. Test the integration:
   ```bash
   go run test-publisher.go test-agent my-new-agent "Test message"
   ```

## Troubleshooting

### Common Issues

**Redis Connection Failed**
```bash
# Check Redis status
docker-compose ps redis

# Check connectivity
redis-cli ping
# Expected: PONG

# Restart if needed
docker-compose restart redis
```

**Agent Manager Not Processing**
```bash
# Check if queues have messages for a specific project
redis-cli ZCARD handoff:project:my-project:queue:api-expert

# Check Agent Manager logs
grep ERROR agent-manager.log

# Verify queue naming matches by scanning for all queues
redis-cli KEYS "handoff:project:*:queue:*"
```

**Handoffs Stuck in Processing**
```bash
# Check for failed executions
grep "FAILURE" agent-manager.log

# Manually inspect handoff data
redis-cli GET handoff:12345abc-def

# Clear stuck queues if needed
redis-cli DEL handoff:project:my-project:queue:problematic-agent
```

**run-agent.sh Permissions**
```bash
# Make script executable
chmod +x agent-manager/run-agent.sh

# Check script syntax
bash -n agent-manager/run-agent.sh
```

### Debugging Tips

1. **Enable verbose logging** in `main.go`:
   ```go
   log.SetLevel(log.DebugLevel)
   ```

2. **Monitor Redis operations**:
   ```bash
   redis-cli MONITOR
   ```

3. **Validate handoff JSON**:
   ```bash
   echo "$PAYLOAD" | jq .
   ```

4. **Check agent execution manually**:
   ```bash
   cd agent-manager
   ./run-agent.sh test-expert '{"metadata":{"handoff_id":"test"}}'
   ```

### Performance Tuning

**Redis Optimization**:
- Use connection pooling
- Set appropriate memory limits
- Configure persistence settings

**Agent Manager Scaling**:
- Run multiple Agent Manager instances
- Distribute agents across instances
- Use Redis Cluster for high availability

**Queue Management**:
- Monitor queue depths regularly
- Adjust processing delays
- Implement priority-based processing

## Advanced Features

### Priority Handling

Handoffs support four priority levels:

- **Critical** (score: 1.x): Immediate processing
- **High** (score: 2.x): Expedited processing  
- **Normal** (score: 3.x): Standard processing
- **Low** (score: 4.x): Background processing

Set priority in handoffs:

```go
handoff.Metadata.Priority = "high"
```

### Retry Logic

Failed handoffs can be automatically retried with exponential backoff:

```go
retryPolicy := RetryPolicy{
    MaxRetries:    3,
    InitialDelay:  time.Second,
    MaxDelay:      time.Minute,
    BackoffFactor: 2.0,
}
```

### Agent Routing

The handoff library supports intelligent routing based on content:

```go
router.AddRoute("api-expert", RouteRule{
    Name:        "route-go-implementation",
    TargetAgent: "golang-expert",
    Priority:    100,
    Conditions: []RouteCondition{{
        Field:    "technical_details.language",
        Operator: "equals",
        Value:    "go",
    }},
})
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Write tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Submit a pull request

### Code Style

- Follow Go conventions and `gofmt`
- Add comprehensive tests
- Document public APIs
- Use meaningful commit messages

## License

MIT License - see LICENSE file for details.