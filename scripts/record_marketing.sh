#!/bin/bash

# Simon Marketing Demo Recorder
# This script prepares a demo and records it using asciinema.

set -e

# 1. Setup Clean Environment
TEMP_HOME=$(mktemp -d)
export HOME=$TEMP_HOME
SIMON_BIN="./simon"

# Build Simon with the new cinematic stub
echo "Building Simon with cinematic stub..."
go build -o simon cmd/simon/main.go

# 2. Prepare Demo Spec
cat <<EOF > demo_task.yaml
goal: "Demonstrate Simon's wizardly governance."
definition_of_done: "Done."
evidence: ["demo_task.yaml"]
EOF

echo "--- READY FOR RECORDING ---"
echo "Recording will capture TUI rendering..."

# 3. Record Execution
# We wrap the command to add a 5 second pause at the end so viewers can see the 'Completed' status
RECORD_CMD="$SIMON_BIN run demo_task.yaml -i --provider stub; sleep 5"

asciinema rec --overwrite -c "$RECORD_CMD" simon_demo.cast

echo "--- RECORDING COMPLETE ---"
echo "Saved to: simon_demo.cast"
