# Performance Optimization Guide for Agents

This guide provides standardized performance optimization patterns that all agents should follow to maximize efficiency and minimize resource usage.

## Core Performance Principles

### 1. Batch Operations
Always prefer batch operations over sequential individual operations when possible.

#### File Operations
```yaml
# ❌ Inefficient: Multiple sequential edits
- Edit file1.ts with change A
- Edit file1.ts with change B  
- Edit file1.ts with change C

# ✅ Efficient: Single MultiEdit operation
- MultiEdit file1.ts with changes A, B, C
```

#### Tool Invocations
```yaml
# ❌ Inefficient: Sequential reads
- Read file1.ts
- Wait for response
- Read file2.ts
- Wait for response
- Read file3.ts

# ✅ Efficient: Parallel reads in single message
- Read file1.ts, file2.ts, file3.ts (all in one tool invocation batch)
```

### 2. Parallel Execution
Execute independent operations in parallel rather than sequentially.

#### Example Patterns
```yaml
parallel_patterns:
  file_analysis:
    # When analyzing a codebase
    - Glob for source files
    - Grep for patterns
    - Read configuration files
    # All executed in parallel when independent
  
  git_operations:
    # When preparing a commit
    - git status
    - git diff
    - git log
    # All executed in parallel for faster results
  
  test_execution:
    # When running tests
    - Unit tests
    - Integration tests  
    - Lint checks
    # Run in parallel when not dependent
```

### 3. Caching Strategies
Avoid redundant operations by implementing smart caching.

#### Cache Patterns
```yaml
caching_strategies:
  file_reads:
    - Cache file contents during session
    - Track file modifications
    - Invalidate cache on changes
  
  search_results:
    - Cache grep/glob results
    - Reuse for similar patterns
    - Time-based expiration (5 min)
  
  computed_values:
    - Cache analysis results
    - Cache dependency trees
    - Cache type definitions
```

### 4. Resource Usage Optimization

#### Memory Management
```yaml
memory_optimization:
  file_handling:
    - Stream large files instead of loading entirely
    - Process files in chunks when possible
    - Release resources after use
  
  data_structures:
    - Use efficient data structures
    - Avoid deep object cloning
    - Clear unused references
```

#### CPU Optimization
```yaml
cpu_optimization:
  processing:
    - Use appropriate algorithms (O(n) vs O(n²))
    - Avoid nested loops when possible
    - Implement early returns
  
  regex_patterns:
    - Compile regex once, reuse multiple times
    - Use simple patterns when complex ones aren't needed
    - Avoid backtracking regex
```

## Agent-Specific Optimizations

### TypeScript Expert
```yaml
typescript_optimizations:
  type_checking:
    - Use incremental compilation
    - Cache type definitions
    - Batch related type updates
  
  component_generation:
    - Generate multiple components in one operation
    - Reuse common imports and patterns
    - Batch prop type definitions
```

### Test Expert
```yaml
test_optimizations:
  test_execution:
    - Run tests in parallel
    - Use test result caching
    - Skip unchanged test suites
  
  coverage_analysis:
    - Incremental coverage updates
    - Cache baseline metrics
    - Batch coverage reports
```

### API Expert
```yaml
api_optimizations:
  endpoint_design:
    - Batch endpoint definitions
    - Reuse common schemas
    - Generate multiple routes together
  
  documentation:
    - Generate all API docs in one pass
    - Cache OpenAPI schemas
    - Batch validation rules
```

### DevOps Expert
```yaml
devops_optimizations:
  deployment:
    - Parallel container builds
    - Layer caching strategies
    - Concurrent deployments
  
  infrastructure:
    - Batch resource creation
    - Parallel provisioning
    - Cached terraform plans
```

## Performance Monitoring

### Metrics to Track
```yaml
performance_metrics:
  execution_time:
    - Tool invocation duration
    - Total task completion time
    - Time between operations
  
  resource_usage:
    - File operations count
    - API calls made
    - Memory consumption
  
  efficiency_ratios:
    - Batch vs individual operations
    - Cache hit rates
    - Parallel execution usage
```

### Performance Reporting
```yaml
performance_report:
  summary:
    total_duration: "2m 15s"
    operations_count: 45
    batch_operations: 12
    parallel_executions: 8
    cache_hits: 23
  
  optimizations_applied:
    - "Used MultiEdit for 5 file changes"
    - "Executed 3 git commands in parallel"
    - "Cached 15 file reads"
  
  recommendations:
    - "Consider batching remaining edits"
    - "Implement caching for API responses"
    - "Use parallel test execution"
```

## Implementation Examples

### Example 1: Efficient File Updates
```typescript
// ❌ Inefficient: Multiple Edit operations
await edit("config.ts", "old1", "new1");
await edit("config.ts", "old2", "new2");
await edit("config.ts", "old3", "new3");

// ✅ Efficient: Single MultiEdit operation
await multiEdit("config.ts", [
  { old_string: "old1", new_string: "new1" },
  { old_string: "old2", new_string: "new2" },
  { old_string: "old3", new_string: "new3" }
]);
```

### Example 2: Parallel File Analysis
```typescript
// ❌ Inefficient: Sequential operations
const tsFiles = await glob("**/*.ts");
const jsFiles = await glob("**/*.js");
const config = await read("package.json");

// ✅ Efficient: Parallel operations in single message
const [tsFiles, jsFiles, config] = await Promise.all([
  glob("**/*.ts"),
  glob("**/*.js"),
  read("package.json")
]);
```

### Example 3: Smart Caching
```typescript
// ✅ Implement session-based cache
const fileCache = new Map();

async function readWithCache(path: string) {
  if (fileCache.has(path)) {
    return fileCache.get(path);
  }
  const content = await read(path);
  fileCache.set(path, content);
  return content;
}
```

## Best Practices Summary

### DO:
- Use MultiEdit for multiple changes to the same file
- Execute independent operations in parallel
- Cache frequently accessed data
- Batch similar operations together
- Monitor and report performance metrics
- Clean up resources after use

### DON'T:
- Make multiple sequential edits to the same file
- Execute independent operations one by one
- Re-read unchanged files repeatedly
- Process large datasets in memory
- Ignore performance degradation
- Leave caches unbounded

## Integration with Handoffs

When creating handoffs, include performance metrics:

```yaml
handoff_performance:
  metrics:
    execution_time: "1m 45s"
    operations_batched: 8
    cache_utilization: "85%"
  
  optimizations_used:
    - "MultiEdit for component updates"
    - "Parallel test execution"
    - "Cached dependency analysis"
  
  recommendations_for_next_agent:
    - "Use cached type definitions in src/types/"
    - "Run integration tests in parallel"
    - "Batch API endpoint creation"
```

This ensures performance optimization is maintained across agent handoffs.