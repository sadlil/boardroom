#!/bin/bash
set -e

FEATURE_NAME=$1
COMMIT_MSG=$2

if [ -z "$FEATURE_NAME" ] || [ -z "$COMMIT_MSG" ]; then
    echo "Usage: ./finish-feature.sh <feature-name> <\"commit message\">"
    exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel)

WORKTREE_DIR="$REPO_ROOT/.worktree/boardroom-$FEATURE_NAME"

if [ ! -d "$WORKTREE_DIR" ]; then
    echo "Worktree does not exist at $WORKTREE_DIR"
    exit 1
fi

# Switch context to worktree
cd "$WORKTREE_DIR"

echo "Running tests..."
if command -v go >/dev/null 2>&1; then
    go test ./... || {
        echo "❌ Tests failed! Fix the code before finalizing."
        exit 1
    }
else
    echo "⚠️ 'go' binary not found. Skipping tests."
fi

echo "Staging changes..."
git add .
git commit -m "$COMMIT_MSG"

echo "================================================="
echo "✅ Feature $FEATURE_NAME committed locally."
echo "The code has not been pushed to the remote repository."
echo "Waiting for user to review and push."
echo "Cleaning up worktree..."
cd "$REPO_ROOT"
git worktree remove "$WORKTREE_DIR"
echo "================================================="
