#!/usr/bin/env bash
# mine hook: todo.done at notify stage (notify mode)
#
# Logs completed todos to a file and optionally sends a webhook.
# This is a notify hook — it runs in the background and doesn't
# block the command. Output is ignored.
#
# Install:
#   cp todo.done.notify.sh ~/.config/mine/hooks/
#   chmod +x ~/.config/mine/hooks/todo.done.notify.sh
#
# Test:
#   mine hook test ~/.config/mine/hooks/todo.done.notify.sh

# Read JSON context from stdin
CONTEXT=$(cat)

# Extract command info
COMMAND=$(echo "$CONTEXT" | jq -r '.command')
TIMESTAMP=$(echo "$CONTEXT" | jq -r '.timestamp')

# Log to file
LOG_FILE="${XDG_DATA_HOME:-$HOME/.local/share}/mine/completed.log"
mkdir -p "$(dirname "$LOG_FILE")"
echo "[$TIMESTAMP] $COMMAND completed" >> "$LOG_FILE"

# Optional: send a webhook notification
# Uncomment and set your webhook URL to enable:
# WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
# curl -s -X POST "$WEBHOOK_URL" \
#   -H "Content-Type: application/json" \
#   -d "{\"text\": \"mine: todo completed at $TIMESTAMP\"}" \
#   > /dev/null 2>&1
