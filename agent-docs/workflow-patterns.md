## Agent Workflow Patterns

### Task Handoff Protocol
```yaml
handoff:
  metadata:
    source_agent: string      # Agent passing the task
    target_agent: string      # Agent receiving the task
    task_id: string          # Unique task identifier
    priority: high|medium|low
    deadline: string         # ISO 8601 timestamp
  
  context:
    project: string          # Project name/identifier
    feature: string          # Feature being worked on
    dependencies: string[]   # Related tasks/features
    constraints: string[]    # Time/resource constraints
  
  technical:
    requirements: string[]   # Technical requirements
    analysis: string        # Prior technical analysis
    considerations: string[] # Important technical points
    edge_cases: string[]    # Known edge cases
  
  artifacts:
    code_paths: string[]    # Relevant code files
    docs_paths: string[]    # Related documentation
    test_paths: string[]    # Associated tests
    
  validation:
    success_criteria: string[] # What defines success
    test_requirements: string[] # Required test coverage
    review_points: string[]    # Specific review needs
```

### Common Workflows

#### Feature Implementation
1. **Project Manager** → Language Expert
   - Task specification
   - Requirements
   - Priority
   
2. **Language Expert** → API Expert
   - API requirements
   - Data structures
   - Integration needs
   
3. **API Expert** → Language Expert
   - API contracts
   - Endpoint specifications
   - Error formats
   
4. **Language Expert** → Security Expert
   - Implementation details
   - Data handling
   - Auth flows
   
5. **Security Expert** → Test Expert
   - Security requirements
   - Test scenarios
   - Edge cases
   
6. **Test Expert** → DevOps Expert
   - Test suites
   - Coverage reports
   - Performance metrics
   
7. **DevOps Expert** → Tech Writer
   - Deployment process
   - Configuration
   - Monitoring setup
   
8. **Tech Writer** → Project Manager
   - Documentation
   - Usage guides
   - Process docs

#### Bug Fix
1. **Project Manager** → Test Expert
   - Bug report
   - Reproduction steps
   - Priority
   
2. **Test Expert** → Language Expert
   - Failed tests
   - Edge cases
   - Regression risks
   
3. **Language Expert** → Security Expert
   - Fix implementation
   - Changed flows
   - Risk assessment
   
4. **Security Expert** → DevOps Expert
   - Security validation
   - Deployment needs
   - Monitoring updates
   
5. **DevOps Expert** → Tech Writer
   - Deployment process
   - Config changes
   - Release notes
   
6. **Tech Writer** → Project Manager
   - Updated docs
   - Change notes
   - User impact

#### Performance Optimization
1. **Project Manager** → Test Expert
   - Performance goals
   - Current metrics
   - Problem areas
   
2. **Test Expert** → Language Expert
   - Performance tests
   - Bottlenecks
   - Profiling data
   
3. **Language Expert** → API Expert
   - Code optimizations
   - API impacts
   - Data flow changes
   
4. **API Expert** → Security Expert
   - API changes
   - Security implications
   - Risk assessment
   
5. **Security Expert** → DevOps Expert
   - Security validation
   - Resource needs
   - Scaling configs
   
6. **DevOps Expert** → Tech Writer
   - Infrastructure changes
   - Performance gains
   - Configuration updates
   
7. **Tech Writer** → Project Manager
   - Updated docs
   - Performance notes
   - Best practices

### Validation Rules

#### Task Acceptance
- Verify all required fields
- Check dependencies
- Validate priorities
- Confirm deadlines
- Review artifacts

#### Task Completion
- Meet success criteria
- Pass all tests
- Security approval
- Documentation updates
- Deployment verified

#### Quality Gates
- Code review passed
- Tests passing
- Security cleared
- Docs updated
- Performance validated

### Best Practices

#### DO:
- Follow handoff protocol
- Include all context
- Check dependencies
- Validate handoffs
- Document changes

#### DON'T:
- Skip steps
- Lose context
- Miss dependencies
- Rush handoffs
- Ignore protocols