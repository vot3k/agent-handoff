---
metadata:
  from_agent: agent-manager
  to_agent: golang-expert
  timestamp: 2024-09-13T10:30:00Z
  task_context: "Implement built-in agent executor approach - Phase 1: Core Binary Extension"
  priority: high

content:
  summary: "Extend the existing manager binary with new execution modes to eliminate manual run-agent.sh copying and enable built-in agent execution with tool-agnostic strategies"
  
  requirements:
    - "Add new CLI flags: --mode, --agent, --payload-file, --payload-stdin"
    - "Implement execution mode logic alongside existing dispatcher mode"
    - "Maintain full backward compatibility with current dispatcher functionality"
    - "Create extensible execution framework for future tool integrations"
    - "Add comprehensive error handling and logging for new execution modes"
    - "Preserve existing Redis queue monitoring and handoff processing"

  artifacts:
    created: []
    modified:
      - "/Users/jimmy/Dev/ai-platforms/agent-handoff/agent-manager/cmd/manager/main.go"
    reviewed:
      - "/Users/jimmy/Dev/ai-platforms/agent-handoff/agent-manager/go.mod"
      - "/Users/jimmy/Dev/ai-platforms/agent-handoff/agent-manager/run-agent.sh"

  technical_details:
    current_architecture:
      mode: "dispatcher_only"
      execution_method: "external_script"
      script_dependency: "run-agent.sh"
      
    target_architecture:
      modes:
        dispatcher: "Current behavior - calls external run-agent.sh"
        executor: "New built-in execution with --agent and --payload flags"
        hybrid: "Built-in execution with script fallback"
      
      new_cli_flags:
        - "--mode [dispatcher|executor|hybrid]"
        - "--agent [agent-name] (required for executor mode)"
        - "--payload-file [path] (payload from file)"
        - "--payload-stdin (payload from stdin)"
        - "--project-path [path] (working directory context)"
        
      execution_framework:
        tool_detection: "Auto-detect available tools (claude, cursor, go, npm)"
        strategy_selection: "Priority-based tool selection"
        fallback_mechanism: "Generic execution when no specific tools"
        context_awareness: "Project type detection and environment setup"

    implementation_phases:
      phase_1_scope:
        - "CLI flag parsing and mode switching"
        - "Basic executor mode implementation"
        - "Tool detection foundation"
        - "Maintain existing dispatcher functionality"
        - "Add execution result handling"

  next_steps:
    - "Implement flag parsing for new execution modes"
    - "Add mode switching logic in main() function"
    - "Create basic executor framework"
    - "Implement tool detection strategies"
    - "Add comprehensive error handling and logging"
    - "Ensure backward compatibility testing"
    - "Prepare for Phase 2 tool integration strategies"

validation:
  schema_version: "1.0"
  checksum: "sha256:mgr-phase1-executor-golang"
---