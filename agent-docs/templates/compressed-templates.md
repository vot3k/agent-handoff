# Compressed Templates for Agent Token Optimization

## Compressed Handoff Schema

Replace verbose handoff sections with this minimal template:

```markdown
## Handoff Protocol

Uses unified schema with agent-specific `technical_details`:
```yaml
metadata: {from_agent, to_agent, timestamp, task_context, priority}
content: {summary, requirements[], artifacts{created[], modified[], reviewed[]}, technical_details, next_steps[]}
validation: {schema_version: "1.0", checksum}
```

### [Agent-Name] Technical Details
```yaml
technical_details:
  key1: value1  # Agent-specific field
  key2: value2  # Agent-specific field
```
```

## Compressed Performance Optimization

Replace verbose performance sections with:

```markdown
## Performance Optimization

### Patterns
- **Batch**: [Agent-specific batch operations]
- **Parallel**: [Agent-specific parallel opportunities]
- **Cache**: [Agent-specific caching strategies]

### Metrics
Track: execution_time, resource_usage, cache_hits
```

## Compressed Example Format

Instead of verbose scenarios:

```markdown
## Example Scenarios

**Scenario**: [Name]
- Trigger: [One line]
- Process: [2-3 key steps]
- Output: [Expected result]
```

## Compressed CoD Format

Use 3-4 word steps:

```markdown
### [Pattern] CoD
```
ANALYZE: Input requirements
DESIGN: Solution approach
IMPLEMENT: Core logic
VALIDATE: Quality checks
```
```

## Compressed Common Mistakes

```markdown
## Common Mistakes

1. **[Mistake]**: [Why wrong] → [Correct approach]
2. **[Mistake]**: [Why wrong] → [Correct approach]
```