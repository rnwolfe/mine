#!/usr/bin/env bash
# mine hook: todo.add at preexec stage (transform mode)
#
# Auto-tags todos with "inbox" if no tags are specified.
# This is a transform hook — it modifies the command context
# and passes it along the pipeline.
#
# Install:
#   cp todo.add.preexec.sh ~/.config/mine/hooks/
#   chmod +x ~/.config/mine/hooks/todo.add.preexec.sh
#
# Test:
#   mine hook test ~/.config/mine/hooks/todo.add.preexec.sh

# Read JSON context from stdin
CONTEXT=$(cat)

# Check if --tags flag is already set (using jq for robust JSON parsing)
HAS_TAGS=$(echo "$CONTEXT" | jq -r 'if .flags | has("tags") then "yes" else empty end')

if [ -z "$HAS_TAGS" ]; then
  # No tags flag — inject "inbox" as default tag
  CONTEXT=$(echo "$CONTEXT" | jq '.flags.tags = "inbox"')
fi

# Output modified context for the next hook in the chain
echo "$CONTEXT"
