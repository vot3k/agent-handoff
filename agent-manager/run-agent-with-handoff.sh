#!/bin/bash
# run-agent-with-handoff.sh - Enhanced agent execution with handoff capability

set -e  # Exit immediately if a command exits with a non-zero status

AGENT_NAME="$1"
PAYLOAD="$2"

if [ -z "$AGENT_NAME" ] || [ -z "$PAYLOAD" ]; then
  echo "Usage: $0 <agent_name> <json_payload>"
  echo "ERROR: Missing required arguments"
  exit 1
fi

echo "=== Executor: Running agent '$AGENT_NAME' with handoff capability ==="
echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "Project Context: $AGENT_PROJECT_NAME"
echo "Agent: $AGENT_NAME"
echo "Payload size: $(echo -n "$PAYLOAD" | wc -c) bytes"

# Parse the handoff payload to extract key information
HANDOFF_ID=$(echo "$PAYLOAD" | jq -r '.handoff_id // .payload.metadata.handoff_id // "unknown"')
FROM_AGENT=$(echo "$PAYLOAD" | jq -r '.payload.metadata.from_agent // .metadata.from_agent // "unknown"')
SUMMARY=$(echo "$PAYLOAD" | jq -r '.payload.content.summary // .content.summary // "No summary provided"')

echo "Handoff ID: $HANDOFF_ID"
echo "From Agent: $FROM_AGENT"
echo "Summary: $SUMMARY"
echo ""

# Store original directory before changing
ORIGINAL_DIR="$(pwd)"

# Create a temporary file for the agent's working directory
WORK_DIR="/tmp/agent-work-$AGENT_NAME-$$"
mkdir -p "$WORK_DIR"
cd "$WORK_DIR"

echo "Working directory: $WORK_DIR"

# Save the payload to a file for the agent to process
echo "$PAYLOAD" > handoff.json

# Log the start of processing
echo "--- Agent Processing Started ---"

# Enhanced Claude Code Integration
if command -v claude >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
    echo "🤖 Using Claude Code with Task tool integration..."
    
    # Create a prompt that instructs Claude to handle the handoff
    CLAUDE_PROMPT="You are the $AGENT_NAME agent. Process this handoff: $(cat handoff.json)

If you need to handoff to another agent, output JSON in this format:
{\"handoff_required\": true, \"target_agent\": \"agent-name\", \"summary\": \"description\", \"details\": \"additional context\"}

If no handoff is needed, output:
{\"handoff_required\": false, \"result\": \"your work summary\"}"

    # Execute Claude and capture output
    CLAUDE_OUTPUT=$(echo "$CLAUDE_PROMPT" | claude)
    echo "$CLAUDE_OUTPUT" > claude_response.txt
    
    # Extract JSON from Claude output (it may contain non-JSON text)
    # Look for JSON blocks in ```json``` blocks or standalone { } blocks
    JSON_BLOCK=$(echo "$CLAUDE_OUTPUT" | sed -n '/```json/,/```/p' | sed '1d;$d' | head -1)
    
    # If no code block found, look for standalone JSON
    if [ -z "$JSON_BLOCK" ]; then
        JSON_BLOCK=$(echo "$CLAUDE_OUTPUT" | sed -n '/{/,/}/p' | tr '\n' ' ')
    fi
    
    echo "Debug: Extracted JSON: $JSON_BLOCK"
    
    # Check if handoff is required (with error handling)
    if echo "$JSON_BLOCK" | jq . >/dev/null 2>&1; then
        HANDOFF_REQUIRED=$(echo "$JSON_BLOCK" | jq -r '.handoff_required // false')
        
        if [ "$HANDOFF_REQUIRED" = "true" ]; then
            TARGET_AGENT=$(echo "$JSON_BLOCK" | jq -r '.target_agent')
            HANDOFF_SUMMARY=$(echo "$JSON_BLOCK" | jq -r '.summary')
            HANDOFF_DETAILS=$(echo "$JSON_BLOCK" | jq -r '.details // ""')
            
            echo "🔄 Handoff requested to: $TARGET_AGENT"
            echo "📝 Summary: $HANDOFF_SUMMARY"
            
            # Publish handoff using the CLI publisher
            PUBLISHER_PATH="$ORIGINAL_DIR/../bin/publisher"
            if [ ! -x "$PUBLISHER_PATH" ]; then
                PUBLISHER_PATH="$ORIGINAL_DIR/../../bin/publisher"  # Try parent directory structure
            fi
            if [ ! -x "$PUBLISHER_PATH" ]; then
                # Try to find it in PATH
                PUBLISHER_PATH=$(which publisher 2>/dev/null)
            fi
            
            if [ -x "$PUBLISHER_PATH" ]; then
                echo "Publishing handoff via CLI publisher at $PUBLISHER_PATH..."
                "$PUBLISHER_PATH" "$AGENT_NAME" "$TARGET_AGENT" "$HANDOFF_SUMMARY" "$HANDOFF_DETAILS"
                echo "✅ Handoff published successfully"
            else
                echo "⚠️  Publisher binary not found at expected locations, handoff not published"
                echo "Checked: ../bin/publisher, ../../bin/publisher, $PWD/../bin/publisher"
            fi
        else
            RESULT=$(echo "$JSON_BLOCK" | jq -r '.result // "Task completed"')
            echo "✅ $AGENT_NAME completed: $RESULT"
        fi
    else
        echo "⚠️  Claude output was not valid JSON, proceeding with simulation"
        echo "Claude output: $CLAUDE_OUTPUT"
    fi

elif [ -f ".claude/handoff-config.yaml" ]; then
    echo "🔧 Using project-specific handoff configuration..."
    
    # Extract configuration
    REDIS_ADDR=$(grep -E "^\s*address:" .claude/handoff-config.yaml | cut -d'"' -f2)
    PROJECT_NAME=$(grep -E "^\s*project_namespace:" .claude/handoff-config.yaml | cut -d'"' -f2)
    
    if [ -n "$REDIS_ADDR" ] && [ -n "$PROJECT_NAME" ]; then
        export HANDOFF_REDIS_ADDR="$REDIS_ADDR"
        export HANDOFF_PROJECT_NAME="$PROJECT_NAME"
        export HANDOFF_ENABLED="true"
        echo "✅ Handoff environment configured"
    fi
    
    # Continue with simulation but with handoff capability
    echo "🤖 Running $AGENT_NAME with handoff capability..."
    SHOULD_HANDOFF=false
    
    # Agent-specific logic with handoff decisions
    case "$AGENT_NAME" in
        "architect-expert")
            echo "🏗️  Architecture Expert: Analyzing system requirements..."
            sleep 1
            echo "🏗️  Architecture Expert: Designing system architecture..."
            sleep 1
            echo "🏗️  Architecture Expert: Creating component diagrams..."
            sleep 1
            echo "🏗️  Architecture Expert: Documenting design decisions..."
            echo "✅ Architecture Expert: Architecture design completed"
            
            # Check if implementation is needed
            if echo "$SUMMARY" | grep -qi "implement\|code\|develop"; then
                SHOULD_HANDOFF=true
                TARGET_AGENT="golang-expert"
                HANDOFF_SUMMARY="Implement the designed architecture using Go"
            fi
            ;;
        
        "golang-expert")
            echo "🐹 Go Expert: Analyzing Go implementation requirements..."
            sleep 1
            echo "🐹 Go Expert: Generating Go code structure..."
            sleep 1
            echo "🐹 Go Expert: Implementing handlers and middleware..."
            sleep 1
            echo "🐹 Go Expert: Adding error handling and validation..."
            echo "✅ Go Expert: Go implementation completed"
            
            # Check if testing is needed
            if echo "$SUMMARY" | grep -qi "test\|validation\|quality"; then
                SHOULD_HANDOFF=true
                TARGET_AGENT="test-expert"
                HANDOFF_SUMMARY="Create comprehensive tests for the Go implementation"
            fi
            ;;
        
        "api-expert")
            echo "🔧 API Expert: Analyzing API requirements..."
            sleep 1
            echo "🔧 API Expert: Generating OpenAPI specification..."
            sleep 1
            echo "🔧 API Expert: Validating endpoint designs..."
            echo "✅ API Expert: API specification completed"
            
            # Check if implementation is needed
            if echo "$SUMMARY" | grep -qi "implement\|backend\|server"; then
                SHOULD_HANDOFF=true
                TARGET_AGENT="golang-expert"
                HANDOFF_SUMMARY="Implement the API specification using Go"
            fi
            ;;
        
        *)
            echo "🤖 Generic Agent: Processing handoff for '$AGENT_NAME'..."
            sleep 2
            echo "🤖 Generic Agent: Task completed successfully"
            echo "✅ Generic Agent: Handoff processed"
            ;;
    esac
    
    # Publish handoff if needed
    if [ "$SHOULD_HANDOFF" = "true" ] && [ -n "$TARGET_AGENT" ]; then
        echo ""
        echo "🔄 Publishing handoff to $TARGET_AGENT..."
        
        if [ -x "../bin/publisher" ]; then
            ../bin/publisher "$AGENT_NAME" "$TARGET_AGENT" "$HANDOFF_SUMMARY"
            echo "✅ Handoff published successfully"
        else
            echo "⚠️  Publisher binary not found at ../bin/publisher"
            echo "Manual handoff: $AGENT_NAME -> $TARGET_AGENT: $HANDOFF_SUMMARY"
        fi
    fi

else
    echo "🔄 Running in basic mode without handoff capability..."
    
    # Original simulation logic for backward compatibility
    case "$AGENT_NAME" in
        "api-expert")
            echo "🔧 API Expert: Analyzing API requirements..."
            sleep 1
            echo "🔧 API Expert: Generating OpenAPI specification..."
            sleep 1
            echo "🔧 API Expert: Validating endpoint designs..."
            echo "✅ API Expert: API specification completed"
            ;;
        
        "golang-expert")
            echo "🐹 Go Expert: Analyzing Go implementation requirements..."
            sleep 1
            echo "🐹 Go Expert: Generating Go code structure..."
            sleep 1
            echo "🐹 Go Expert: Implementing handlers and middleware..."
            sleep 1
            echo "🐹 Go Expert: Adding error handling and validation..."
            echo "✅ Go Expert: Go implementation completed"
            ;;
        
        *)
            echo "🤖 Generic Agent: Processing handoff for '$AGENT_NAME'..."
            sleep 2
            echo "🤖 Generic Agent: Task completed successfully"
            echo "✅ Generic Agent: Handoff processed"
            ;;
    esac
fi

# Simulate creating some output artifacts
echo "Creating output artifacts..."
echo "{\"agent\": \"$AGENT_NAME\", \"status\": \"completed\", \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" > output.json

echo ""
echo "--- Agent Processing Completed ---"
echo "Agent '$AGENT_NAME' finished successfully"
echo "Working directory: $WORK_DIR (will be cleaned up)"

# Clean up working directory (optional - you might want to keep it for debugging)
# rm -rf "$WORK_DIR"

# Return success
exit 0