---
title: Conditional execution
description:
  Skip optional work with if, enforce requirements with preconditions, and ask
  for confirmation before execution with prompt.
section: Guide
docType: guide
outline: deep
---

# Conditional execution

Decide what a failed check should mean before choosing a setting: skip optional
work, stop with an error, or ask the user whether to continue.

| What do you need?                | Use                                                                                                  |
| -------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Skip optional work               | [`if`](#conditional-execution-with-if)                                                               |
| Stop when a requirement is unmet | [`preconditions`](#using-programmatic-checks-to-cancel-the-execution-of-a-task-and-its-dependencies) |
| Ask before executing commands    | [`prompt`](#confirmation-prompts)                                                                    |
| Skip work already completed      | [Up-to-date checks](./up-to-date.md)                                                                 |

## Skip optional work {#conditional-execution-with-if}

An `if` condition is a shell command. Exit code `0` allows execution; a non-zero
exit code skips the guarded task or command without failing the run.

### Skip a whole task {#task-level-if}

Use a task-level condition for optional work. This task only runs when a `docs/`
directory exists:

```yaml
version: '3'

tasks:
  docs:
    if: test -d docs
    cmds:
      - echo 'Building documentation'
```

Run `task docs` with and without the directory. If it is absent, Task skips the
task and its dependencies. Other tasks in the invocation can continue.

### Skip one command {#command-level-if}

Put `if` on a command when the rest of the task should still run:

```yaml
version: '3'

tasks:
  build:
    cmds:
      - cmd: echo 'Building documentation'
        if: test -d docs
      - echo 'Building the application'
```

`task build` always prints the application message. It prints the documentation
message first only if `docs/` exists.

### Check a Task variable {#using-templates-in-if-conditions}

Template comparisons produce `true` or `false`, which are also shell commands
with the corresponding exit codes. Put a configurable default at the root:

```yaml
version: '3'

vars:
  ENABLE_FEATURE: 'false'

tasks:
  feature:
    if: '{{eq .ENABLE_FEATURE "true"}}'
    cmds:
      - echo 'Feature is enabled'
```

`task feature ENABLE_FEATURE=true` runs the command. Without that override, Task
skips it.

### Filter loop iterations {#using-if-with-for-loops}

A command's condition is evaluated for each iteration, with that iteration's
values:

```yaml
version: '3'

tasks:
  process-items:
    cmds:
      - for: ['a', 'b', 'c']
        cmd: echo "processing {{.ITEM}}"
        if: '[ "{{.ITEM}}" != "b" ]'
```

Run `task process-items` to print `processing a` and `processing c`. See
[Loops](./loops.md) for the other iteration forms.

## Enforce a requirement {#using-programmatic-checks-to-cancel-the-execution-of-a-task-and-its-dependencies}

Use `preconditions` when missing a requirement should fail the task. Every check
must exit with code `0`. Add `msg` to explain how to resolve a failure:

```yaml
version: '3'

tasks:
  generate-files:
    preconditions:
      - sh: test -f .env
        msg: 'Create a .env file before generating files.'
    cmds:
      - mkdir -p directory
      - touch directory/file.txt
```

`task generate-files` fails with the message if `.env` is missing. Create the
file and run it again to generate `directory/file.txt`.

A plain shell string also works when no custom message is needed. A failed
precondition fails tasks that call or depend on this task. Dependencies of the
checked task have already run before its preconditions are evaluated; put a
check on the task whose commands need protecting.

`--force` bypasses preconditions as well as up-to-date checks.

### Choose skip or failure {#if-vs-preconditions}

Use `if` when skipping is an acceptable result. Use `preconditions` when a
caller should receive an error instead. To require a variable or restrict its
allowed values, use [Validation and prompts](./required-variables.md).

## Ask for confirmation {#confirmation-prompts}

Use `prompt` to ask for approval before a task's commands run. For example, this
task asks before deleting a local build directory:

```yaml
version: '3'

tasks:
  clean:
    prompt: Remove the local build directory?
    cmds:
      - rm -rf build/
```

Run `task clean` in a terminal and answer the confirmation. Declining stops the
task with [exit code](../reference/cli.md#exit-codes) `205`.

Unlike
[prompts for missing input](./required-variables.md#prompting-for-missing-variables-interactively),
confirmation prompts do not require `--interactive`. Supplying every required
variable does not bypass them.

For several confirmations, use a list:

```yaml
prompt:
  - Remove the local build directory?
  - Have you saved everything you need from it?
```

Task asks after dependencies have completed and before running `cmds`. If
approval must precede other tasks, call them from `cmds` in the prompted task
instead of listing them as dependencies.

Use `task --yes clean` (or `task -y clean`) to accept automatically. In CI or
another environment without a terminal, confirmation fails unless automatic
acceptance is enabled.

::: tip Related guide

<span id="limiting-when-tasks-run"></span>

To control how often a task runs when several tasks call it, use
[`run: once` or `run: when_changed`](./dependencies.md#repeated-calls). These
settings apply within one invocation. Use [up-to-date checks](./up-to-date.md)
to skip completed work across invocations.

:::
