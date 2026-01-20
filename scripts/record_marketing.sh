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

if [[ "$RECORD_PROVIDER" == "openai" && -z "${OPENAI_API_KEY}" ]]; then
  echo "OPENAI_API_KEY must be set for provider=openai"
  exit 1
fi

# Build Simon with the new cinematic stub
echo "Building Simon with cinematic stub..."
go build -o simon cmd/simon/main.go

# 2. Prepare Demo Spec
cat <<EOF > demo_task.yaml
goal: "Record Simon enforcing budgets, evidence, and verification with the stub provider."
definition_of_done: "Session completes after evidence verification and memory archiving."
evidence: ["demo_task.yaml"]
EOF

echo "--- READY FOR RECORDING ---"
echo "Recording will capture TUI rendering..."

# 3. Record Execution
# We wrap the command to add a 5 second pause at the end so viewers can see the 'Completed' status
RECORD_CMD="$SIMON_BIN run demo_task.yaml -i --provider $RECORD_PROVIDER --model $RECORD_MODEL; sleep 5"

asciinema rec --overwrite -c "$RECORD_CMD" simon_demo.cast

echo "--- RECORDING COMPLETE ---"
echo "Saved to: simon_demo.cast"
