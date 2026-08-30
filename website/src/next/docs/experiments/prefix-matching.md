---
title: 'Prefix Matching (#2947)'
description: Experiment to enable shortest unique prefix and segment matching for task names
outline: deep
---

# Prefix Matching (#2947)

::: warning

All experimental features are subject to breaking changes and/or removal _at any
time_. We strongly recommend that you do not use these features in a production
environment. They are intended for testing and feedback only.

:::

::: info

To enable this experiment, set the environment variable:
`TASK_X_PREFIX_MATCHING=1` or configure it in `.taskrc.yml`:

```yaml
experiments:
  PREFIX_MATCHING: 1
```

Check out [our guide to enabling experiments](./index.md#enabling-experiments) for more information.

:::

This experiment adds support for **Shortest Unique Prefix Matching** (segment-wise abbreviation matching) when running tasks.

In large Taskfiles with multi-level namespaces (for example, `api:openapi:export`), typing the full task name or manually defining short aliases for each task can be tedious.

### How It Works

1. **Unique Match:** When an input prefix uniquely matches a task (or one of its aliases), Task will execute it immediately.
   - `task a:o:e` matches `api:openapi:export`
   - `task api:o` matches `api:openapi:export` (if no other `api:o*` tasks exist)
   - `task b` matches `build` (if no other `b*` tasks exist)

2. **Ambiguous Match:** If the prefix matches multiple tasks, Task will halt execution and return a conflict error listing all candidate tasks:
   ```text
   task: Found multiple tasks (docker:build:production, docker:build:staging) that match "doc"
   ```

3. **Precedence:** Direct task name matches, alias matches, and wildcard matches always take precedence over prefix matching.
