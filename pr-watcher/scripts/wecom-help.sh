#!/usr/bin/env bash
# Quick reference for WeChat Work integration

cat << 'EOF'
╔════════════════════════════════════════════════════════════════╗
║           WeChat Work Integration - Quick Reference            ║
╚════════════════════════════════════════════════════════════════╝

📋 TEMPLATES
  • github-wecom.tmpl  - GitHub PR report template
  • gitlab-wecom.tmpl  - GitLab MR report template

🚀 COMMANDS
  Test templates:
    ./scripts/test-wecom.sh

  Preview message:
    go run scripts/wecom-sender.go <json-file> <template-file>

  Send to WeChat Work:
    ./scripts/send-to-wecom.sh github <json-file> <webhook-url>
    ./scripts/send-to-wecom.sh gitlab <json-file> <webhook-url>

💡 EXAMPLES
  # GitHub PR report
  ./scripts/send-to-wecom.sh github example-output.json "$WECOM_WEBHOOK_URL"

  # GitLab MR report
  ./scripts/send-to-wecom.sh gitlab example-gitlab-output.json "$WECOM_WEBHOOK_URL"

  # Integrated workflow
  pr-watcher watch-prs --org myorg --days 7 --output /tmp/prs.json
  ./scripts/send-to-wecom.sh github /tmp/prs.json "$WECOM_WEBHOOK_URL"

🔧 SETUP
  1. Install prerequisites: brew install jq
  2. Get WeChat Work webhook URL from admin console
  3. Export: export WECOM_WEBHOOK_URL="https://qyapi.weixin.qq.com/..."
  4. Test: ./scripts/test-wecom.sh
  5. Send: ./scripts/send-to-wecom.sh github example-output.json "$WECOM_WEBHOOK_URL"

📚 DOCUMENTATION
  • WECOM_INTEGRATION.md - Full integration guide
  • WECOM_SUMMARY.md     - Feature summary
  • README.md            - Updated with WeChat Work section

✨ FEATURES
  ✓ Compact markdown format
  ✓ Limits to 10 PRs/MRs
  ✓ Draft indicators (🚧)
  ✓ Pipeline status (✅❌🔄⚠️)
  ✓ Clickable links
  ✓ Assignee mentions

EOF
