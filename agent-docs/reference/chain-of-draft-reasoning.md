# Chain-of-Draft (CoD) Reasoning for Sub-Agents

## Overview

Chain-of-Draft (CoD) is a reasoning technique that constrains each reasoning step to 5 words or less, forcing concise, essential thinking. This document outlines how sub-agents can leverage CoD for more efficient reasoning while maintaining quality.

## Benefits for Sub-Agents

1. **93% token reduction** - Drastically reduces computational overhead
2. **Enforced clarity** - Prevents verbose, unclear explanations
3. **Better scalability** - Complex problems don't explode token usage
4. **Mimics expert thinking** - Aligns with how human experts chunk information

## Implementation Examples by Agent

### Architect Expert

**Traditional reasoning:**
```
"First, we need to consider the scalability requirements. Given that we expect 10,000 concurrent users, we should implement a microservices architecture with load balancing..."
```

**CoD reasoning:**
```
1. Users: 10k concurrent expected
2. Pattern: microservices with balancing
3. Database: shard by tenant
4. Cache: Redis for sessions
5. Queue: RabbitMQ for async
```

### API Expert

**Traditional reasoning:**
```
"For the authentication endpoint, we should use JWT tokens with refresh token rotation to balance security and user experience..."
```

**CoD reasoning:**
```
1. Auth: JWT with refresh
2. Endpoint: POST /auth/login
3. Response: token, refresh, expiry
4. Error: 401 invalid credentials
5. Rate: 5 attempts/minute
```

### Security Expert

**Security audit with CoD:**
```
SCAN: SQL injection vulnerabilities
CHECK: input validation exists
REVIEW: authentication bypass risks
AUDIT: sensitive data exposure
TEST: rate limiting effectiveness
```

### Test Expert

**Test planning with CoD:**
```
UNIT: auth module 80%
INTEGRATION: API endpoints complete
E2E: user journey flows
PERFORMANCE: 1000 RPS target
SECURITY: OWASP top 10
```

### DevOps Expert

**Deployment checklist with CoD:**
```
BUILD: Docker multi-stage optimize
TEST: CI pipeline automated
DEPLOY: Kubernetes rolling update
MONITOR: Prometheus metrics configured
ROLLBACK: Previous version ready
```

## Prompt Templates

### For Analysis Tasks
```
Analyze [system/code/design] using 5-word steps:

Step format: "Action: key insight"
Focus on: [specific aspect]

Example output:
1. Identify: performance bottleneck location
2. Measure: current latency metrics
3. Solution: implement caching layer
```

### For Implementation Tasks
```
Implement [feature] with compressed reasoning:

Each step: max 5 words
Format: "Component: action/decision"

Example:
- Database: PostgreSQL for transactions
- Cache: Redis for speed
- API: REST over GraphQL
```

### For Review Tasks
```
Review [code/design/security] in steps:

[5 words per finding]
Priority: High/Medium/Low

Example:
- HIGH: SQL injection possible
- MEDIUM: Password validation weak
- LOW: Comments need updating
```

## Best Practices

### DO:
- Use domain-specific abbreviations consistently
- Focus on decisions, not descriptions
- Maintain logical flow between steps
- Include quantitative data when relevant
- Preserve critical context

### DON'T:
- Sacrifice accuracy for brevity
- Skip essential security considerations
- Ignore error handling steps
- Compress beyond comprehension
- Lose important nuance

## Integration with Existing Workflows

### 1. Combine with Workflow Patterns
```yaml
workflow_step:
  cod_planning:
    - Analyze: current system state
    - Identify: improvement opportunities  
    - Design: solution architecture
    - Implement: core components
    - Validate: meets requirements
```

### 2. Use for Quick Handoffs
When transferring between agents:
```
FROM: api-expert
TO: typescript-expert
CONTEXT: "Auth: JWT implementation needed"
```

### 3. Debugging with CoD
```
BUG: Login fails intermittently
TRACE: Token validation timeout
CAUSE: Redis connection pool
FIX: Increase pool size
TEST: Concurrent user simulation
```

## Measuring Effectiveness

Track these metrics when using CoD:
- Token usage reduction percentage
- Task completion accuracy
- Time to solution
- Handoff clarity between agents
- Review/revision frequency

## Examples for Common Tasks

### Feature Implementation
```
TASK: Add user notifications
PLAN: Email and push
STORE: PostgreSQL notification queue  
SEND: AWS SES emails
TRACK: Delivery status table
```

### Bug Investigation
```
SYMPTOM: Slow API response
PROFILE: Database query time
FOUND: Missing index column
APPLY: CREATE INDEX users_email
VERIFY: Response under 100ms
```

### Performance Optimization
```
BASELINE: 500ms page load
TARGET: Under 200ms goal
OPTIMIZE: Bundle size reduction
CACHE: Static assets CDN
RESULT: 180ms achieved target
```

## When NOT to Use CoD

Avoid CoD for:
- Legal or compliance documentation
- Complex algorithmic explanations
- User-facing error messages
- Detailed security protocols
- Nuanced architectural decisions requiring context

## Conclusion

CoD reasoning can significantly improve sub-agent efficiency while maintaining quality. Start with simple tasks, measure results, and gradually expand usage as teams become comfortable with the compressed format.