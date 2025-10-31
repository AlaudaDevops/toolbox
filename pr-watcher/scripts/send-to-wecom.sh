#!/usr/bin/env bash
# Script to send PR/MR reports to WeChat Work webhook
# Usage: ./send-to-wecom.sh <github|gitlab> <json-file> <webhook-url>

set -e

PLATFORM=$1
JSON_FILE=$2
WEBHOOK_URL=$3

if [ -z "$PLATFORM" ] || [ -z "$JSON_FILE" ] || [ -z "$WEBHOOK_URL" ]; then
    echo "Usage: $0 <github|gitlab> <json-file> <webhook-url>"
    echo "Example: $0 github example-output.json https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY"
    exit 1
fi

if [ ! -f "$JSON_FILE" ]; then
    echo "Error: JSON file '$JSON_FILE' not found"
    exit 1
fi

# Determine template file
if [ "$PLATFORM" = "github" ]; then
    TEMPLATE_FILE="github-wecom.tmpl"
elif [ "$PLATFORM" = "gitlab" ]; then
    TEMPLATE_FILE="gitlab-wecom.tmpl"
else
    echo "Error: Platform must be 'github' or 'gitlab'"
    exit 1
fi

# Find project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMPLATE_PATH="$PROJECT_ROOT/$TEMPLATE_FILE"

if [ ! -f "$TEMPLATE_PATH" ]; then
    echo "Error: Template file '$TEMPLATE_PATH' not found"
    exit 1
fi

# Generate markdown message using Go template
echo "Generating message from template..."
MESSAGE=$(go run "$SCRIPT_DIR/wecom-sender.go" "$JSON_FILE" "$TEMPLATE_PATH")

if [ -z "$MESSAGE" ]; then
    echo "Error: Failed to generate message"
    exit 1
fi

# Prepare WeChat Work webhook payload
PAYLOAD=$(jq -n \
    --arg content "$MESSAGE" \
    '{
        "msgtype": "markdown",
        "markdown": {
            "content": $content
        }
    }')

# Send to WeChat Work webhook
echo "Sending message to WeChat Work..."
RESPONSE=$(curl -s -X POST "$WEBHOOK_URL" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD")

# Check response
ERROR_CODE=$(echo "$RESPONSE" | jq -r '.errcode // 0')
ERROR_MSG=$(echo "$RESPONSE" | jq -r '.errmsg // "ok"')

if [ "$ERROR_CODE" = "0" ]; then
    echo "✅ Message sent successfully!"
else
    echo "❌ Failed to send message: [$ERROR_CODE] $ERROR_MSG"
    exit 1
fi
