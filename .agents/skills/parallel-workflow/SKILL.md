---
name: Parallel Feature Workflow
description: Automates isolating tasks using git worktrees and syncing knowledge.
---

# Parallel Agent Workflow

When you are asked to implement a feature or perform a task using "the parallel workflow", you MUST follow these exact steps to ensure you do not interfere with other agents or block the main repository.

## 1. Start the Feature
You must create an isolated workspace via git worktrees.
Run the startup script from the repository root:
`./.agents/skills/parallel-workflow/scripts/start-feature.sh <feature-branch-name>`

## 2. Switch Context
The startup script will output an absolute path pointing to `.worktree/boardroom-<feature-branch-name>`.
**CRITICAL**: You must execute ALL your subsequent file edits, file creations, and commands inside that specific `$WORKTREE_DIR`! Do not edit files in the main repository folder, otherwise you defeat the purpose of the worktree.

## 3. Implement the Feature
Complete the user's task within your worktree. If you make any core architectural changes, DB schema updates, or API route definitions, you MUST document them.

## 4. Document Changes (Share Knowledge)
Before finishing, append a summary of your structural changes to:
`.agents/shared_context/README.md`
*(Note: update the file in the main repository if it's not being tracked inside your worktree, or update the worktree copy before committing so it gets synced).*

## 5. Finish and Commit
When development AND testing are complete, run the finish script from the repository root:
`./.agents/skills/parallel-workflow/scripts/finish-feature.sh <feature-branch-name> "<detailed commit message>"`

This will commit the code locally to your branch and clean up the worktree folder. The user is responsible for reviewing and pushing the changes.
