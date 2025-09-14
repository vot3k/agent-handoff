# Claude Code Agent Handoff Integration Architecture

## Overview

This document describes the integration architecture for connecting Claude Code sub-agents with the Redis-based agent handoff system. The design enables seamless agent-to-agent orchestration while maintaining Claude Code's existing UX patterns.

## Architecture Components

### 1. Claude Code SDK Integration Layer

#### 1.1 HandoffSDK for Claude Code
```go
// claude-code-sdk/handoff.go
package claudecode

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"
    
    "github.com/your-org/agent-handoff/handoff"
)

// ClaudeCodeHandoffClient provides handoff capabilities for Claude Code agents
type ClaudeCodeHandoffClient struct {
    agent       *handoff.OptimizedHandoffAgent
    projectName string
    agentName   string
    config      ClaudeCodeConfig
}

// ClaudeCodeConfig contains Claude Code specific configuration
type ClaudeCodeConfig struct {
    ProjectName     string            `json:"project_name"`
    AgentName       string            `json:"agent_name"`
    RedisConfig     handoff.RedisPoolConfig `json:"redis_config"`
    Authentication  AuthConfig        `json:"authentication"`
    Capabilities    AgentCapabilities `json:"capabilities"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {
    TokenPath       string `json:"token_path"`
    APIKey          string `json:"api_key"`
    ProjectToken    string `json:"project_token"`
    AllowedAgents   []string `json:"allowed_agents"`
}

// AgentCapabilities defines what the agent can do
type AgentCapabilities struct {
    InputTypes      []string `json:"input_types"`
    OutputTypes     []string `json:"output_types"`
    MaxConcurrency  int      `json:"max_concurrency"`
    TimeoutSeconds  int      `json:"timeout_seconds"`
}

// NewClaudeCodeHandoffClient creates a new handoff client for Claude Code
func NewClaudeCodeHandoffClient(config ClaudeCodeConfig) (*ClaudeCodeHandoffClient, error) {
    // Initialize optimized handoff agent
    agentConfig := handoff.OptimizedConfig{
        RedisConfig: config.RedisConfig,
        LogLevel:    "info",
    }
    
    agent, err := handoff.NewOptimizedHandoffAgent(agentConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create handoff agent: %w", err)
    }
    
    // Register agent capabilities
    capabilities := handoff.AgentCapabilities{
        Name:          config.AgentName,
        QueueName:     fmt.Sprintf("handoff:project:%s:queue:%s", config.ProjectName, config.AgentName),
        MaxConcurrent: config.Capabilities.MaxConcurrency,
    }
    
    if err := agent.RegisterAgent(capabilities); err != nil {
        return nil, fmt.Errorf("failed to register agent: %w", err)
    }
    
    return &ClaudeCodeHandoffClient{
        agent:       agent,
        projectName: config.ProjectName,
        agentName:   config.AgentName,
        config:      config,
    }, nil
}

// PublishHandoff publishes a handoff to another agent
func (c *ClaudeCodeHandoffClient) PublishHandoff(ctx context.Context, req HandoffRequest) (*HandoffResponse, error) {
    // Validate authentication
    if err := c.validateAuth(req.ToAgent); err != nil {
        return nil, fmt.Errorf("authentication failed: %w", err)
    }
    
    // Create handoff payload
    handoff := &handoff.Handoff{
        Metadata: handoff.HandoffMetadata{
            ProjectName: c.projectName,
            FromAgent:   c.agentName,
            ToAgent:     req.ToAgent,
            TaskContext: req.TaskContext,
            Priority:    handoff.Priority(req.Priority),
        },
        Content: handoff.HandoffContent{
            Summary:          req.Summary,
            Requirements:     req.Requirements,
            Artifacts:        req.Artifacts,
            TechnicalDetails: req.TechnicalDetails,
            NextSteps:        req.NextSteps,
        },
    }
    
    // Publish handoff
    if err := c.agent.PublishHandoff(ctx, handoff); err != nil {
        return nil, fmt.Errorf("failed to publish handoff: %w", err)
    }
    
    return &HandoffResponse{
        HandoffID: handoff.Metadata.HandoffID,
        Status:    "published",
        QueueName: fmt.Sprintf("handoff:project:%s:queue:%s", c.projectName, req.ToAgent),
    }, nil
}

// validateAuth validates if the current agent can handoff to the target agent
func (c *ClaudeCodeHandoffClient) validateAuth(toAgent string) error {
    // Check if target agent is in allowed list
    if len(c.config.Authentication.AllowedAgents) > 0 {
        allowed := false
        for _, agent := range c.config.Authentication.AllowedAgents {
            if agent == toAgent || agent == "*" {
                allowed = true
                break
            }
        }
        if !allowed {
            return fmt.Errorf("agent %s not authorized to handoff to %s", c.agentName, toAgent)
        }
    }
    
    return nil
}

// HandoffRequest represents a handoff request from Claude Code
type HandoffRequest struct {
    ToAgent          string                 `json:"to_agent"`
    TaskContext      string                 `json:"task_context"`
    Priority         string                 `json:"priority"`
    Summary          string                 `json:"summary"`
    Requirements     []string               `json:"requirements"`
    Artifacts        map[string][]string    `json:"artifacts"`
    TechnicalDetails map[string]interface{} `json:"technical_details"`
    NextSteps        []string               `json:"next_steps"`
}

// HandoffResponse represents the response from a handoff publication
type HandoffResponse struct {
    HandoffID string `json:"handoff_id"`
    Status    string `json:"status"`
    QueueName string `json:"queue_name"`
}
```

#### 1.2 Task Tool Integration
```go
// claude-code-sdk/task.go
package claudecode

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
)

// TaskHandoffBridge bridges Claude Code's Task tool with the handoff system
type TaskHandoffBridge struct {
    client *ClaudeCodeHandoffClient
}

// NewTaskHandoffBridge creates a new task handoff bridge
func NewTaskHandoffBridge(configPath string) (*TaskHandoffBridge, error) {
    // Load configuration from file or environment
    config, err := loadConfig(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }
    
    client, err := NewClaudeCodeHandoffClient(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create handoff client: %w", err)
    }
    
    return &TaskHandoffBridge{
        client: client,
    }, nil
}

// ExecuteTaskWithHandoff executes a task and optionally hands off to another agent
func (t *TaskHandoffBridge) ExecuteTaskWithHandoff(ctx context.Context, task TaskRequest) (*TaskResponse, error) {
    // Execute the current agent's task
    response, err := t.executeCurrentTask(ctx, task)
    if err != nil {
        return nil, fmt.Errorf("task execution failed: %w", err)
    }
    
    // Check if handoff is requested
    if task.HandoffTo != "" {
        handoffReq := HandoffRequest{
            ToAgent:          task.HandoffTo,
            TaskContext:      task.Context,
            Priority:         task.Priority,
            Summary:          response.Summary,
            Requirements:     task.NextStepRequirements,
            Artifacts:        response.Artifacts,
            TechnicalDetails: response.TechnicalDetails,
            NextSteps:        task.NextSteps,
        }
        
        handoffResp, err := t.client.PublishHandoff(ctx, handoffReq)
        if err != nil {
            return nil, fmt.Errorf("handoff failed: %w", err)
        }
        
        response.HandoffID = handoffResp.HandoffID
        response.HandoffStatus = handoffResp.Status
    }
    
    return response, nil
}

// executeCurrentTask handles the current agent's specific task logic
func (t *TaskHandoffBridge) executeCurrentTask(ctx context.Context, task TaskRequest) (*TaskResponse, error) {
    // This would integrate with Claude Code's existing task execution logic
    // For now, we'll simulate the execution
    
    switch t.client.agentName {
    case "api-expert":
        return t.executeAPITask(ctx, task)
    case "golang-expert":
        return t.executeGolangTask(ctx, task)
    case "typescript-expert":
        return t.executeTypescriptTask(ctx, task)
    default:
        return t.executeGenericTask(ctx, task)
    }
}

// TaskRequest represents a task request from Claude Code
type TaskRequest struct {
    Context              string                 `json:"context"`
    Requirements         []string               `json:"requirements"`
    Priority             string                 `json:"priority"`
    HandoffTo            string                 `json:"handoff_to,omitempty"`
    NextSteps            []string               `json:"next_steps,omitempty"`
    NextStepRequirements []string               `json:"next_step_requirements,omitempty"`
    TechnicalDetails     map[string]interface{} `json:"technical_details,omitempty"`
}

// TaskResponse represents the response from task execution
type TaskResponse struct {
    Summary          string                 `json:"summary"`
    Artifacts        map[string][]string    `json:"artifacts"`
    TechnicalDetails map[string]interface{} `json:"technical_details"`
    HandoffID        string                 `json:"handoff_id,omitempty"`
    HandoffStatus    string                 `json:"handoff_status,omitempty"`
    Status           string                 `json:"status"`
}

// loadConfig loads configuration from file or environment
func loadConfig(configPath string) (ClaudeCodeConfig, error) {
    var config ClaudeCodeConfig
    
    // Try to load from file first
    if configPath != "" {
        data, err := os.ReadFile(configPath)
        if err == nil {
            if err := json.Unmarshal(data, &config); err == nil {
                return config, nil
            }
        }
    }
    
    // Fall back to environment variables
    config = ClaudeCodeConfig{
        ProjectName: getEnvOrDefault("CLAUDE_CODE_PROJECT_NAME", "default"),
        AgentName:   getEnvOrDefault("CLAUDE_CODE_AGENT_NAME", "claude-code"),
        RedisConfig: handoff.RedisPoolConfig{
            Addr:         getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
            Password:     os.Getenv("REDIS_PASSWORD"),
            DB:           0,
            PoolSize:     10,
            MinIdleConns: 5,
        },
        Authentication: AuthConfig{
            TokenPath:     getEnvOrDefault("CLAUDE_CODE_TOKEN_PATH", ""),
            APIKey:        os.Getenv("CLAUDE_CODE_API_KEY"),
            ProjectToken:  os.Getenv("CLAUDE_CODE_PROJECT_TOKEN"),
            AllowedAgents: []string{"*"}, // Allow all by default
        },
        Capabilities: AgentCapabilities{
            InputTypes:      []string{"text", "json", "yaml"},
            OutputTypes:     []string{"text", "json", "yaml", "code"},
            MaxConcurrency:  5,
            TimeoutSeconds:  300,
        },
    }
    
    return config, nil
}

func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

### 2. Communication Patterns

#### 2.1 Synchronous Handoff Pattern
```yaml
# Synchronous handoff for immediate response requirements
pattern: synchronous_handoff
use_case: "API design → immediate Go implementation"
flow:
  1. api-expert completes API specification
  2. Publishes handoff to golang-expert 
  3. Waits for completion confirmation
  4. Returns combined results to user
timeout: 5 minutes
error_handling: "rollback to previous state"
```

#### 2.2 Asynchronous Pipeline Pattern
```yaml
# Asynchronous pipeline for complex workflows
pattern: async_pipeline
use_case: "Architecture → API → Implementation → Testing"
flow:
  1. architect-expert creates system design
  2. Publishes to api-expert (async)
  3. api-expert publishes to golang-expert (async)
  4. golang-expert publishes to test-expert (async)
  5. Each agent reports completion independently
monitoring: "progress tracking via Redis keys"
error_handling: "circuit breaker with retry"
```

#### 2.3 Parallel Processing Pattern
```yaml
# Parallel processing for independent tasks
pattern: parallel_processing
use_case: "Frontend and backend development simultaneously"
flow:
  1. architect-expert creates design
  2. Publishes to both golang-expert and typescript-expert
  3. Both agents work in parallel
  4. Results aggregated when both complete
coordination: "barrier synchronization"
error_handling: "partial failure tolerance"
```

### 3. Configuration and Deployment

#### 3.1 Project-Level Configuration
```yaml
# .claude-code/handoff-config.yaml
project:
  name: "my-project"
  version: "1.0.0"
  
redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 10
  
authentication:
  mode: "token"  # "token", "api_key", "none"
  token_path: ".claude-code/project-token"
  api_key_env: "CLAUDE_CODE_API_KEY"
  allowed_agents:
    - "api-expert"
    - "golang-expert"
    - "typescript-expert"
    - "test-expert"

agents:
  api-expert:
    capabilities:
      input_types: ["requirements", "architecture"]
      output_types: ["openapi", "specification"]
      max_concurrency: 3
      timeout_seconds: 300
    
  golang-expert:
    capabilities:
      input_types: ["openapi", "requirements"]
      output_types: ["go_code", "tests"]
      max_concurrency: 5
      timeout_seconds: 600
    
  typescript-expert:
    capabilities:
      input_types: ["api_spec", "requirements"]
      output_types: ["react_components", "types"]
      max_concurrency: 3
      timeout_seconds: 450

monitoring:
  metrics_enabled: true
  health_check_interval: "30s"
  log_level: "info"
  
deployment:
  auto_start_manager: true
  manager_binary_path: "./bin/agent-manager"
  run_agent_script: "./scripts/run-agent.sh"
```

#### 3.2 Agent-Specific Configuration
```json
{
  "agent_name": "golang-expert",
  "project_name": "my-project",
  "redis_config": {
    "addr": "localhost:6379",
    "pool_size": 10
  },
  "authentication": {
    "token_path": ".claude-code/tokens/golang-expert.token",
    "allowed_agents": ["api-expert", "architect-expert", "test-expert"]
  },
  "capabilities": {
    "input_types": ["openapi_spec", "requirements", "architecture"],
    "output_types": ["go_source", "go_tests", "go_modules"],
    "max_concurrency": 5,
    "timeout_seconds": 600
  },
  "handoff_rules": {
    "auto_handoff_patterns": [
      {
        "when": "implementation_complete",
        "to_agent": "test-expert",
        "condition": "if tests_required"
      }
    ],
    "retry_policy": {
      "max_retries": 3,
      "initial_delay": "5s",
      "backoff_factor": 2.0
    }
  }
}
```

### 4. Authentication and Security

#### 4.1 Token-Based Authentication
```go
// security/auth.go
package security

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "time"
)

// ProjectToken represents a project-specific authentication token
type ProjectToken struct {
    ProjectName string    `json:"project_name"`
    AgentName   string    `json:"agent_name"`
    TokenHash   string    `json:"token_hash"`
    CreatedAt   time.Time `json:"created_at"`
    ExpiresAt   time.Time `json:"expires_at"`
    Permissions []string  `json:"permissions"`
}

// TokenManager manages authentication tokens for agents
type TokenManager struct {
    tokens map[string]*ProjectToken
}

// GenerateToken generates a new authentication token for an agent
func (tm *TokenManager) GenerateToken(projectName, agentName string, permissions []string) (string, error) {
    // Generate random token
    tokenBytes := make([]byte, 32)
    if _, err := rand.Read(tokenBytes); err != nil {
        return "", fmt.Errorf("failed to generate token: %w", err)
    }
    
    token := hex.EncodeToString(tokenBytes)
    tokenHash := sha256.Sum256([]byte(token))
    
    // Create token record
    projectToken := &ProjectToken{
        ProjectName: projectName,
        AgentName:   agentName,
        TokenHash:   hex.EncodeToString(tokenHash[:]),
        CreatedAt:   time.Now(),
        ExpiresAt:   time.Now().Add(24 * time.Hour), // 24 hour expiry
        Permissions: permissions,
    }
    
    // Store token
    tokenKey := fmt.Sprintf("%s:%s", projectName, agentName)
    tm.tokens[tokenKey] = projectToken
    
    return token, nil
}

// ValidateToken validates an authentication token
func (tm *TokenManager) ValidateToken(projectName, agentName, token string) (*ProjectToken, error) {
    tokenKey := fmt.Sprintf("%s:%s", projectName, agentName)
    storedToken, exists := tm.tokens[tokenKey]
    if !exists {
        return nil, fmt.Errorf("token not found for %s:%s", projectName, agentName)
    }
    
    // Check expiry
    if time.Now().After(storedToken.ExpiresAt) {
        delete(tm.tokens, tokenKey)
        return nil, fmt.Errorf("token expired")
    }
    
    // Validate token hash
    tokenHash := sha256.Sum256([]byte(token))
    if hex.EncodeToString(tokenHash[:]) != storedToken.TokenHash {
        return nil, fmt.Errorf("invalid token")
    }
    
    return storedToken, nil
}
```

#### 4.2 Project Isolation
```yaml
# Project isolation through Redis namespacing
isolation_strategy:
  redis_namespacing:
    pattern: "handoff:project:{project_name}:*"
    benefits:
      - "Projects can't access each other's queues"
      - "Independent scaling per project"
      - "Separate authentication contexts"
  
  file_system_isolation:
    working_dirs: "/tmp/claude-code/{project_name}/{agent_name}"
    artifact_storage: ".claude-code/artifacts/{project_name}"
    log_separation: ".claude-code/logs/{project_name}/{agent_name}"
  
  network_isolation:
    redis_db_per_project: true
    connection_pools: "separate per project"
    rate_limiting: "per project basis"
```

### 5. Error Handling and Monitoring

#### 5.1 Error Handling Strategy
```go
// monitoring/errors.go
package monitoring

import (
    "context"
    "fmt"
    "time"
)

// ErrorHandler handles errors across the handoff system
type ErrorHandler struct {
    circuitBreaker *CircuitBreaker
    retryPolicy    *RetryPolicy
    alertManager   *AlertManager
}

// CircuitBreaker prevents cascading failures
type CircuitBreaker struct {
    maxFailures    int
    resetTimeout   time.Duration
    state          CircuitState
    failures       int
    lastFailureTime time.Time
}

type CircuitState int

const (
    CircuitClosed CircuitState = iota
    CircuitOpen
    CircuitHalfOpen
)

// HandleError processes errors and determines appropriate action
func (eh *ErrorHandler) HandleError(ctx context.Context, err error, handoffID string) error {
    // Log error
    eh.logError(err, handoffID)
    
    // Check circuit breaker
    if eh.circuitBreaker.ShouldReject() {
        return fmt.Errorf("circuit breaker open: %w", err)
    }
    
    // Determine if error is retriable
    if eh.isRetriable(err) {
        return eh.scheduleRetry(ctx, handoffID, err)
    }
    
    // Send alert for non-retriable errors
    eh.alertManager.SendAlert(AlertCritical, fmt.Sprintf("Non-retriable error in handoff %s: %v", handoffID, err))
    
    return err
}

// isRetriable determines if an error should trigger a retry
func (eh *ErrorHandler) isRetriable(err error) bool {
    retriableErrors := []string{
        "connection refused",
        "timeout",
        "temporary",
        "redis connection",
    }
    
    errStr := err.Error()
    for _, retriable := range retriableErrors {
        if contains(errStr, retriable) {
            return true
        }
    }
    
    return false
}
```

#### 5.2 Monitoring and Metrics
```go
// monitoring/metrics.go
package monitoring

import (
    "context"
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector collects and exports handoff metrics
type MetricsCollector struct {
    handoffTotal       prometheus.Counter
    handoffDuration    prometheus.Histogram
    handoffErrors      prometheus.Counter
    queueDepth         prometheus.Gauge
    activeAgents       prometheus.Gauge
    redisConnections   prometheus.Gauge
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
    return &MetricsCollector{
        handoffTotal: promauto.NewCounter(prometheus.CounterOpts{
            Name: "claude_code_handoffs_total",
            Help: "Total number of handoffs processed",
        }),
        handoffDuration: promauto.NewHistogram(prometheus.HistogramOpts{
            Name:    "claude_code_handoff_duration_seconds",
            Help:    "Duration of handoff processing",
            Buckets: prometheus.DefBuckets,
        }),
        handoffErrors: promauto.NewCounter(prometheus.CounterOpts{
            Name: "claude_code_handoff_errors_total",
            Help: "Total number of handoff errors",
        }),
        queueDepth: promauto.NewGauge(prometheus.GaugeOpts{
            Name: "claude_code_queue_depth",
            Help: "Current depth of handoff queues",
        }),
        activeAgents: promauto.NewGauge(prometheus.GaugeOpts{
            Name: "claude_code_active_agents",
            Help: "Number of active agents",
        }),
        redisConnections: promauto.NewGauge(prometheus.GaugeOpts{
            Name: "claude_code_redis_connections",
            Help: "Number of active Redis connections",
        }),
    }
}

// RecordHandoff records a handoff completion
func (mc *MetricsCollector) RecordHandoff(duration time.Duration, success bool) {
    mc.handoffTotal.Inc()
    mc.handoffDuration.Observe(duration.Seconds())
    
    if !success {
        mc.handoffErrors.Inc()
    }
}

// UpdateQueueDepth updates queue depth metrics
func (mc *MetricsCollector) UpdateQueueDepth(depth int64) {
    mc.queueDepth.Set(float64(depth))
}
```

### 6. Integration Examples

#### 6.1 API Expert to Golang Expert Handoff
```bash
# Claude Code command that triggers handoff
claude-code task --agent-type api-expert --handoff-to golang-expert \
  --context "Design and implement user authentication API" \
  --requirements "REST API, JWT tokens, password hashing, rate limiting" \
  --priority high
```

#### 6.2 Architecture Expert Orchestrating Multiple Agents
```yaml
# Handoff configuration for complex workflow
workflow:
  name: "full_stack_development"
  stages:
    - agent: "architect-expert"
      task: "Design system architecture"
      outputs: ["architecture_docs", "api_boundaries"]
      handoff_to: ["api-expert", "database-expert"]
      
    - agent: "api-expert"
      depends_on: ["architect-expert"]
      task: "Design REST API specification"
      outputs: ["openapi_spec", "api_docs"]
      handoff_to: ["golang-expert", "typescript-expert"]
      
    - agent: "golang-expert"
      depends_on: ["api-expert"]
      task: "Implement backend services"
      outputs: ["go_services", "unit_tests"]
      handoff_to: ["test-expert"]
      
    - agent: "typescript-expert"
      depends_on: ["api-expert"]
      task: "Implement frontend components"
      outputs: ["react_components", "types"]
      handoff_to: ["test-expert"]
      
    - agent: "test-expert"
      depends_on: ["golang-expert", "typescript-expert"]
      task: "Create integration tests"
      outputs: ["test_suite", "coverage_report"]
```

## Implementation Roadmap

### Phase 1: Core Integration (Weeks 1-2)
- [ ] Implement ClaudeCodeHandoffClient SDK
- [ ] Create TaskHandoffBridge for Task tool integration
- [ ] Basic Redis queue integration
- [ ] Simple authentication mechanism

### Phase 2: Security and Configuration (Weeks 3-4)
- [ ] Token-based authentication system
- [ ] Project-level configuration management
- [ ] Error handling and retry mechanisms
- [ ] Basic monitoring and logging

### Phase 3: Advanced Features (Weeks 5-6)
- [ ] Circuit breaker implementation
- [ ] Prometheus metrics integration
- [ ] Complex workflow orchestration
- [ ] Performance optimization

### Phase 4: Production Readiness (Weeks 7-8)
- [ ] Comprehensive testing
- [ ] Documentation and examples
- [ ] Performance benchmarking
- [ ] Security audit

## Deployment Guide

### For Existing Projects

1. **Install agent-manager binary**:
   ```bash
   curl -L https://github.com/your-org/agent-handoff/releases/latest/download/agent-manager-$(uname -s)-$(uname -m) -o /usr/local/bin/agent-manager
   chmod +x /usr/local/bin/agent-manager
   ```

2. **Initialize project configuration**:
   ```bash
   claude-code init-handoff --project-name my-project
   ```

3. **Start agent manager**:
   ```bash
   agent-manager --config .claude-code/handoff-config.yaml
   ```

4. **Use handoffs in Claude Code**:
   ```bash
   claude-code task --agent-type api-expert --handoff-to golang-expert \
     --context "Create user API" --priority high
   ```

### For New Projects

1. **Create project with handoff support**:
   ```bash
   claude-code new-project my-project --enable-handoffs
   ```

2. **Configure agents**:
   ```bash
   claude-code configure-agents --agents api-expert,golang-expert,test-expert
   ```

3. **Start development with orchestration**:
   ```bash
   claude-code orchestrate --workflow full-stack-development
   ```

This architecture provides a robust, scalable foundation for integrating Claude Code sub-agents with the Redis handoff system while maintaining security, monitoring, and ease of use.