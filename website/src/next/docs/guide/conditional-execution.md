---
title: Conditional execution
description:
  Decide whether a task should run at all, using `preconditions`, `if`, and the
  flags that limit when a task runs.
outline: deep
---

# Conditional execution

Where up-to-date checks ask whether the work is already done, these controls ask
whether the work should happen in the first place.

## Using programmatic checks to cancel the execution of a task and its dependencies

In addition to `status` checks, `preconditions` checks are the logical inverse
of `status` checks. That is, if you need a certain set of conditions to be
_true_ you can use the `preconditions` stanza. `preconditions` are similar to
`status` lines, except they support `sh` expansion, and they SHOULD all
return 0.

```yaml
version: '3'

tasks:
  generate-files:
    cmds:
      - mkdir directory
      - touch directory/file1.txt
      - touch directory/file2.txt
    # test existence of files
    preconditions:
      - test -f .env
      - sh: '[ 1 = 0 ]'
        msg: "One doesn't equal Zero, Halting"
```

Preconditions can set specific failure messages that can tell a user what steps
to take using the `msg` field.

If a task has a dependency on a sub-task with a precondition, and that
precondition is not met - the calling task will fail. Note that a task executed
with a failing precondition will not run unless `--force` is given.

Unlike `status`, which will skip a task if it is up to date and continue
executing tasks that depend on it, a `precondition` will fail a task, along with
any other tasks that depend on it.

```yaml
version: '3'

tasks:
  task-will-fail:
    preconditions:
      - sh: 'exit 1'

  task-will-also-fail:
    deps:
      - task-will-fail

  task-will-still-fail:
    cmds:
      - task: task-will-fail
      - echo "I will not run"
```

## Conditional execution with `if`

The `if` attribute allows you to conditionally skip tasks or commands based on a
shell command's exit code. Unlike `preconditions` which fail and stop execution,
`if` simply skips the task or command when the condition is not met and
continues with the rest of the Taskfile.

### Task-level `if`

When `if` is set on a task, the entire task is skipped if the condition fails:

```yaml
version: '3'

tasks:
  deploy:
    if: '[ "$CI" = "true" ]'
    cmds:
      - echo "Deploying..."
      - ./deploy.sh
```

### Command-level `if`

When `if` is set on a command, only that specific command is skipped:

```yaml
version: '3'

tasks:
  build:
    cmds:
      - cmd: echo "Building for production"
        if: '[ "$ENV" = "production" ]'
      - cmd: echo "Building for development"
        if: '[ "$ENV" = "development" ]'
      - go build ./...
```

### Using templates in `if` conditions

You can use Go template expressions in `if` conditions. Template expressions
like <span v-pre>`{{eq .VAR "value"}}`</span> evaluate to `true` or `false`,
which are valid shell commands (`true` exits with 0, `false` exits with 1):

```yaml
version: '3'

tasks:
  conditional:
    vars:
      ENABLE_FEATURE: 'true'
    cmds:
      - cmd: echo "Feature is enabled"
        if: '{{eq .ENABLE_FEATURE "true"}}'
      - cmd: echo "Feature is disabled"
        if: '{{ne .ENABLE_FEATURE "true"}}'
```

### Using `if` with `for` loops

When used inside a `for` loop, the `if` condition is evaluated for each
iteration:

```yaml
version: '3'

tasks:
  process-items:
    cmds:
      - for: ['a', 'b', 'c']
        cmd: echo "processing {{.ITEM}}"
        if: '[ "{{.ITEM}}" != "b" ]'
```

This will output:

```
processing a
processing c
```

### `if` vs `preconditions`

| Aspect     | `if`                 | `preconditions` |
| ---------- | -------------------- | --------------- |
| On failure | Skips (continues)    | Fails (stops)   |
| Message    | Only in verbose mode | Always shown    |
| Use case   | "Run if possible"    | "Must be true"  |

Use `if` when you want optional conditional execution that shouldn't stop the
workflow. Use `preconditions` when the condition must be met for the task to
make sense.

## Limiting when tasks run

If a task executed by multiple `cmds` or multiple `deps` you can control when it
is executed using `run`. `run` can also be set at the root of the Taskfile to
change the behavior of all the tasks unless explicitly overridden.

Supported values for `run`:

- `always` (default) always attempt to invoke the task regardless of the number
  of previous executions
- `once` only invoke this task once regardless of the number of references
- `when_changed` only invokes the task once for each unique set of variables
  passed into the task

```yaml
version: '3'

tasks:
  default:
    cmds:
      - task: generate-file
        vars: { CONTENT: '1' }
      - task: generate-file
        vars: { CONTENT: '2' }
      - task: generate-file
        vars: { CONTENT: '2' }

  generate-file:
    run: when_changed
    deps:
      - install-deps
    cmds:
      - echo {{.CONTENT}}

  install-deps:
    run: once
    cmds:
      - sleep 5 # long operation like installing packages
```
