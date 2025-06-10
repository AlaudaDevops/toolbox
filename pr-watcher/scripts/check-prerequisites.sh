#!/bin/bash

# Script to check GitHub CLI prerequisites
set -e

echo "🔍 Checking prerequisites for PR Watcher CLI..."

# Check if GitHub CLI is installed
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) is not installed"
    echo "   Please install it from: https://cli.github.com/"
    exit 1
fi

echo "✅ GitHub CLI is installed: $(gh --version | head -n1)"

# Check if GitHub CLI is authenticated
if ! gh auth status &> /dev/null; then
    echo "❌ GitHub CLI is not authenticated"
    echo "   Please run: gh auth login"
    exit 1
fi

echo "✅ GitHub CLI is authenticated"

# Get current user info
USER_INFO=$(gh api user --jq '.login')
echo "✅ Authenticated as: $USER_INFO"

echo ""
# Check if GitLab CLI is installed (optional)
if command -v glab &> /dev/null; then
    echo "✅ GitLab CLI is installed: $(glab --version | head -n1)"

    # Check if GitLab CLI is authenticated
    if glab auth status &> /dev/null; then
        echo "✅ GitLab CLI is authenticated"
        GITLAB_USER=$(glab api user --jq '.username' 2>/dev/null || echo "Unknown")
        echo "✅ GitLab authenticated as: $GITLAB_USER"
    else
        echo "⚠️  GitLab CLI is not authenticated (run: glab auth login)"
    fi
else
    echo "⚠️  GitLab CLI (glab) is not installed (optional for GitLab features)"
    echo "   Install from: https://gitlab.com/gitlab-org/cli"
fi

echo ""
echo "🎉 GitHub prerequisites are met!"
if command -v glab &> /dev/null && glab auth status &> /dev/null; then
    echo "🎉 GitLab prerequisites are also met!"
fi

echo ""
echo "Example usage:"
echo "GitHub:"
echo "  ./bin/pr-watcher watch-prs --org YOUR_ORG --days 7"
echo "  ./bin/pr-watcher watch-prs --org YOUR_ORG --days 14 --output old-prs.json"
echo ""
echo "GitLab:"
echo "  ./bin/pr-watcher watch-mrs --group YOUR_GROUP --days 7"
echo "  ./bin/pr-watcher watch-mrs --group YOUR_GROUP --days 14 --output old-mrs.json"
