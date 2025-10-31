# WeChat Work Integration

## Overview

This directory contains scripts to send PR/MR reports to WeChat Work (企业微信) webhooks.

## Files

- `send-to-wecom.sh` - Main script to send messages to WeChat Work
- `wecom-sender.go` - Helper Go program to render templates

## Prerequisites

- `bash`
- `go` (for template rendering)
- `jq` (for JSON processing)
- `curl` (for sending HTTP requests)

Install prerequisites on macOS:
```bash
brew install jq
```

## Usage

### 1. Get WeChat Work Webhook URL

1. Open WeChat Work admin console
2. Go to Apps & Mini Programs → Group Bots
3. Create a new bot or use existing one
4. Copy the webhook URL (format: `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY`)

### 2. Send GitHub PR Report

```bash
./scripts/send-to-wecom.sh github example-output.json "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY"
```

### 3. Send GitLab MR Report

```bash
./scripts/send-to-wecom.sh gitlab example-gitlab-output.json "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY"
```

## Environment Variable

You can also set the webhook URL as an environment variable:

```bash
export WECOM_WEBHOOK_URL="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY"

# Then use it
./scripts/send-to-wecom.sh github example-output.json "$WECOM_WEBHOOK_URL"
```

## Testing

To test the message generation without sending:

```bash
# Generate message only
go run scripts/wecom-sender.go example-output.json github-wecom.tmpl

# Or for GitLab
go run scripts/wecom-sender.go example-gitlab-output.json gitlab-wecom.tmpl
```

## Examples

### Example 1: GitHub PR Report
```bash
cd /Users/danielfbm/code/github.com/alaudadevops/toolbox/pr-watcher
./scripts/send-to-wecom.sh github example-output.json "$WECOM_WEBHOOK_URL"
```

### Example 2: GitLab MR Report
```bash
cd /Users/danielfbm/code/github.com/alaudadevops/toolbox/pr-watcher
./scripts/send-to-wecom.sh gitlab example-gitlab-output.json "$WECOM_WEBHOOK_URL"
```

## Troubleshooting

### jq not found
```bash
brew install jq
```

### Permission denied
```bash
chmod +x scripts/send-to-wecom.sh
```

### Template not found
Make sure you're running the script from the project root directory or the templates are in the correct location.

## WeChat Work Markdown Support

WeChat Work supports the following markdown syntax:
- **Bold**: `**text**`
- *Italic*: `*text*` (may not work in all versions)
- Links: `[text](url)`
- Code: `` `code` ``
- Mentions: `@username` or `<@userid>`

## Customization

Edit the template files to customize the message format:
- `github-wecom.tmpl` - GitHub PR report template
- `gitlab-wecom.tmpl` - GitLab MR report template
