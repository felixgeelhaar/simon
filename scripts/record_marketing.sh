#!/bin/bash

# Simon Marketing Demo Recorder
# This script prepares a demo and records it using asciinema.

set -e

# 1. Setup Clean Environment
TEMP_HOME=$(mktemp -d)
export HOME=$TEMP_HOME
SIMON_BIN="./simon"

# Build if missing
if [ ! -f "$SIMON_BIN" ]; then
    echo "Building Simon..."
    go build -o simon cmd/simon/main.go
fi

# Configure API Key (using env var or prompt)
if [ -z "$OPENAI_API_KEY" ]; then
    echo "Error: OPENAI_API_KEY environment variable is not set."
    exit 1
fi

echo "Configuring Simon..."
$SIMON_BIN config set openai.api_key "$OPENAI_API_KEY"

# 2. Prepare Demo Spec
mkdir -p demo_project
cat <<EOF > demo_task.yaml
goal: "Create a Go script that lists all files and saves the count to 'output.txt'."
definition_of_done: "'output.txt' exists and contains the count."
evidence: ["output.txt"]
EOF

echo "--- READY FOR RECORDING ---"
echo "Starting asciinema in 3 seconds..."
sleep 3

# 3. Record Execution
# We use -i for interactive TUI mode which is the most visual
asciinema rec -c "$SIMON_BIN run demo_task.yaml -i --provider openai --model gpt-4o" simon_demo.cast

echo "--- RECORDING COMPLETE ---"
echo "Saved to: simon_demo.cast"
echo "You can play it back with: asciinema play simon_demo.cast"
echo "Or upload it to asciinema.org for a shareable link."
