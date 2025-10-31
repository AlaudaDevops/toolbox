#!/usr/bin/env bash
# Quick test script to verify the WeChat Work integration works
# This doesn't actually send to WeChat Work, just tests the message generation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🧪 Testing WeChat Work Integration"
echo "=================================="
echo

echo "📝 Testing GitHub PR template..."
GITHUB_MSG=$(go run "$SCRIPT_DIR/wecom-sender.go" "$PROJECT_ROOT/example-output.json" "$PROJECT_ROOT/github-wecom.tmpl")
if [ $? -eq 0 ]; then
    echo "✅ GitHub template works!"
    echo "Preview:"
    echo "$GITHUB_MSG" | head -n 10
    echo "..."
else
    echo "❌ GitHub template failed!"
    exit 1
fi

echo
echo "📝 Testing GitLab MR template..."
GITLAB_MSG=$(go run "$SCRIPT_DIR/wecom-sender.go" "$PROJECT_ROOT/example-gitlab-output.json" "$PROJECT_ROOT/gitlab-wecom.tmpl")
if [ $? -eq 0 ]; then
    echo "✅ GitLab template works!"
    echo "Preview:"
    echo "$GITLAB_MSG" | head -n 10
    echo "..."
else
    echo "❌ GitLab template failed!"
    exit 1
fi

echo
echo "=================================="
echo "✅ All tests passed!"
echo
echo "To send to WeChat Work, run:"
echo "  ./scripts/send-to-wecom.sh github example-output.json \$WECOM_WEBHOOK_URL"
echo "  ./scripts/send-to-wecom.sh gitlab example-gitlab-output.json \$WECOM_WEBHOOK_URL"
