#!/bin/bash
set -e

FEATURE_NAME=$1

if [ -z "$FEATURE_NAME" ]; then
    echo "Usage: ./start-feature.sh <feature-name>"
    exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

WORKTREE_DIR="$REPO_ROOT/.worktree/boardroom-$FEATURE_NAME"

if [ -d "$WORKTREE_DIR" ]; then
    echo "Worktree already exists at $WORKTREE_DIR"
    exit 1
fi

# Ensure .worktree directory exists
mkdir -p .worktree

# Create worktree and branch
git worktree add "$WORKTREE_DIR" -b "$FEATURE_NAME"

echo "================================================="
echo "✅ Git Worktree created for feature: $FEATURE_NAME"
echo "📂 Worktree Path: $WORKTREE_DIR"
echo "================================================="
echo "IMPORTANT: As an AI agent, you MUST chdir or specify paths into $WORKTREE_DIR"
echo "before executing any further code changes."
