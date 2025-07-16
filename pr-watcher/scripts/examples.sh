#!/bin/bash

# Example script showing how to use PR Watcher CLI for both GitHub and GitLab
set -e

echo "🚀 PR Watcher CLI - Example Usage"
echo "=================================="
echo ""

# Check if the binary exists
if [ ! -f "./bin/pr-watcher" ]; then
    echo "❌ PR Watcher binary not found. Please run 'make build' first."
    exit 1
fi

echo "📋 Available commands:"
echo "1. GitHub: watch-prs - Monitor pull requests in GitHub organizations"
echo "2. GitLab: watch-mrs - Monitor merge requests in GitLab groups"
echo ""

echo "📚 Command Help Examples:"
echo "========================"
echo ""

echo "🐙 GitHub PR Watcher Help:"
echo "-------------------------"
./bin/pr-watcher watch-prs --help
echo ""

echo "🦊 GitLab MR Watcher Help:"
echo "-------------------------"
./bin/pr-watcher watch-mrs --help
echo ""

echo "💡 Quick Examples:"
echo "=================="
echo ""
echo "GitHub - Find PRs older than 7 days:"
echo "  ./bin/pr-watcher watch-prs --org YOUR_ORG --days 7"
echo ""
echo "GitHub - Save results to file:"
echo "  ./bin/pr-watcher watch-prs --org YOUR_ORG --days 14 --output old-prs.json"
echo ""
echo "GitLab - Find MRs older than 7 days:"
echo "  ./bin/pr-watcher watch-mrs --group YOUR_GROUP --days 7"
echo ""
echo "GitLab - Save results to file:"
echo "  ./bin/pr-watcher watch-mrs --group YOUR_GROUP --days 14 --output old-mrs.json"
echo ""
echo "GitLab - Custom host (self-hosted):"
echo "  ./bin/pr-watcher watch-mrs --group YOUR_GROUP --host gitlab.example.com --days 7"
echo ""

echo "🔧 Prerequisites:"
echo "================="
echo "- For GitHub: Install and authenticate 'gh' (GitHub CLI)"
echo "- For GitLab: Install and authenticate 'glab' (GitLab CLI)"
echo ""
echo "Run './scripts/check-prerequisites.sh' to verify your setup."
