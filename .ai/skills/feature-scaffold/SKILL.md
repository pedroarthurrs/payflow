---
name: feature-scaffold
description: Scaffold a new feature spec folder from project templates
---

Scaffold a new task/feature folder.

## Task

Create specs folder for the task described in $ARGUMENTS.

Format: `{task-slug}`
Example: `audit-logging`

## Process

1. Create folder `.ai/specs/{task-slug}/`
2. Copy `.ai/templates/FEATURE_SPEC.md` to `.ai/specs/{task-slug}/FEATURE_SPEC.md`
3. Copy `.ai/templates/IMPLEMENTATION_TRACKER.md` to `.ai/specs/{task-slug}/IMPLEMENTATION_TRACKER.md`
4. Replace `[FEATURE_NAME]` placeholders with the task name
5. Confirm creation by listing files
