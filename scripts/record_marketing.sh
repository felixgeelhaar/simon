#!/bin/bash

# Simon Marketing Demo Recorder
# This script prepares a demo and records it using asciinema.

set -e

# 1. Setup Clean Environment
TEMP_HOME=$(mktemp -d)
export HOME=$TEMP_HOME
SIMON_BIN="./simon"

# 1.1 Provider config
RECORD_PROVIDER=${RECORD_PROVIDER:-openai}
RECORD_MODEL=${RECORD_MODEL:-gpt-4o}
RECORD_COLS=${RECORD_COLS:-120}
RECORD_ROWS=${RECORD_ROWS:-34}
RECORD_OUTPUT=${RECORD_OUTPUT:-website/public/simon_demo.cast}

# Validate API keys for providers that need them
case "$RECORD_PROVIDER" in
  openai)
    if [[ -z "${OPENAI_API_KEY}" ]]; then
      echo "OPENAI_API_KEY must be set for provider=openai"
      exit 1
    fi
    ;;
  anthropic)
    if [[ -z "${ANTHROPIC_API_KEY}" ]]; then
      echo "ANTHROPIC_API_KEY must be set for provider=anthropic"
      exit 1
    fi
    ;;
  gemini)
    if [[ -z "${GEMINI_API_KEY}" ]]; then
      echo "GEMINI_API_KEY must be set for provider=gemini"
      exit 1
    fi
    ;;
  ollama)
    # Ollama runs locally, no API key needed
    ;;
  *)
    echo "Unknown provider: $RECORD_PROVIDER"
    echo "Supported providers: openai, anthropic, gemini, ollama"
    exit 1
    ;;
esac

# Build Simon
echo "Building Simon..."
go build -o simon cmd/simon/main.go

# 2. Prepare Demo Spec
cat <<EOF > demo_task.yaml
goal: "Create a hello world Go CLI application with proper project structure."
definition_of_done: "A working Go module with main.go that prints 'Hello, Simon!' when executed."
evidence: ["demo_task.yaml"]
EOF

echo "--- READY FOR RECORDING ---"
echo "Recording will capture TUI rendering..."

# 3. Record Execution
# We wrap the command to add a 5 second pause at the end so viewers can see the 'Completed' status
RECORD_CMD="clear; $SIMON_BIN run demo_task.yaml -i --provider $RECORD_PROVIDER --model $RECORD_MODEL; sleep 5"

asciinema rec --overwrite --cols "$RECORD_COLS" --rows "$RECORD_ROWS" -c "$RECORD_CMD" "$RECORD_OUTPUT"

echo "--- RECORDING COMPLETE ---"
echo "Saved to: $RECORD_OUTPUT"
