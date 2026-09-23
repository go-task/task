---
title: Output and logging
description:
  Choose an output mode, hide command echoes, and display colors and failure
  annotations in CI.
section: Guide
docType: guide
outline: deep
---

# Output and logging

When several tasks run together, their output can be hard to follow. Choose an
output mode to identify the source of each line or keep each command's output
together. Use `silent` separately to hide the command text Task logs.

## Choose an output mode {#output-syntax}

| What do you need?                        | Mode                    |
| ---------------------------------------- | ----------------------- |
| Live output without extra formatting     | `interleaved` (default) |
| Live output labeled by task              | `prefixed`              |
| Each command's output in one block       | `group`                 |
| One progress line per task, with timings | `status`                |

### Identify parallel output

Set `output: prefixed` at the root of the Taskfile:

```yaml
version: '3'

output: prefixed

tasks:
  check:
    deps: [lint, test]

  lint:
    cmds:
      - echo 'Lint passed'

  test:
    cmds:
      - echo 'Tests passed'
```

Run `task --silent check`. The command output identifies each task:

```text
[lint] Lint passed
[test] Tests passed
```

The lines may appear in either order because the tasks run concurrently. You can
override the output mode for one invocation with `task --output group check` (or
`task -o group check`).

Use a task's `prefix` to distinguish repeated calls with different inputs:

```yaml
version: '3'

output: prefixed

tasks:
  default:
    deps:
      - task: check
        vars: { SERVICE: api }
      - task: check
        vars: { SERVICE: worker }

  check:
    prefix: 'check-{{.SERVICE}}'
    silent: true
    cmds:
      - echo 'Passed'
```

The output uses `[check-api]` and `[check-worker]` prefixes.

### Group command output

Set `output: group` to buffer each command's stdout and stderr until that
command finishes. Long-running commands do not provide live feedback in this
mode. Groups are per command, so blocks from different tasks can still alternate
between commands.

To show buffered output only when a command fails:

```yaml
version: '3'

silent: true
output:
  group:
    error_only: true

tasks:
  passes: echo 'Everything passed'
  fails: echo 'Failure details' && exit 1
```

`task passes` prints nothing. `task fails` prints `Failure details` and the
failure message. Hiding successful output does not change exit codes.

### Report task progress

Set `output: status` to print one line for each task instead of its output. Each
line names the task and says whether it ran, was skipped as up to date,
succeeded, or failed. A task that succeeds has its output hidden. A task that
fails shows its output in full, as `group` with `error_only` does.

```yaml
version: '3'

output: status

tasks:
  default:
    deps: [lint, test]

  lint:
    sources: ['**/*.go']
    cmds:
      - echo 'Lint passed'

  test:
    cmds:
      - echo 'Tests passed'
```

```text
Running    default
Running    lint
Running    test
Skipped    lint
Succeeded  test (1.21s)
Succeeded  default (1.22s)
1 Skipped, 2 Succeeded
```

The line is labeled with the task's `prefix` when it sets one, and with the task
name otherwise. Control characters are removed from the label, so a `prefix`
cannot move the cursor and write over a line this mode already printed.

This mode also works with `--dry`, which makes
`task --dry --output status <name>` print the whole plan and mark the steps that
are already up to date. A dry run executes nothing, so every task it lists
reports `Succeeded` with a duration near zero. Read it as the plan, not as a
result.

A task with `interactive: true` keeps the terminal and is not buffered, so a
task that prompts for input still works. Mark any task that prompts, or its
prompt waits behind a buffer that nobody sees.

## Hide command echoes {#silent-mode}

Use `--silent` (or `-s`) to hide the command text Task logs before execution.
The command's own output remains visible:

```yaml
version: '3'

tasks:
  greet:
    cmds:
      - echo 'Hello, World!'
```

`task greet` shows the command and its output. `task --silent greet` prints only
`Hello, World!`.

To make this permanent for selected commands:

```yaml
version: '3'

tasks:
  greet:
    cmds:
      - cmd: echo 'Hello, World!'
        silent: true
```

You can also set `silent: true` on a task for all its commands, or at the root
of the Taskfile for all tasks.

To suppress a command's stdout, use shell redirection instead:

```yaml
version: '3'

tasks:
  quiet:
    cmds:
      - echo 'This output is discarded' > /dev/null
```

Stderr remains visible unless you redirect it too. Silent mode is not a
substitute for [secret masking](./secret-variables.md).

## Make CI logs readable {#ci-integration}

### Fold command logs

Use `output.group.begin` and `end` to add the markers a CI system uses for
collapsible logs. For GitHub Actions:

```yaml
version: '3'

output:
  group:
    begin: '::group::{{.TASK}}'
    end: '::endgroup::'

tasks:
  default:
    silent: true
    cmds:
      - echo 'Hello, World!'
```

Running `task` produces:

```text
::group::default
Hello, World!
::endgroup::
```

These markers follow
[GitHub Actions' grouping format](https://docs.github.com/en/actions/learn-github-actions/workflow-commands-for-github-actions#grouping-log-lines).
Other providers, such as
[Azure Pipelines](https://docs.microsoft.com/en-us/azure/devops/pipelines/scripts/logging-commands?expand=1&view=azure-devops&tabs=bash#formatting-commands),
use different markers.

### Control colors {#colored-output}

Task enables its colored output when `CI=true`, which most CI providers set
automatically. Use `FORCE_COLOR=1` to force colors or `NO_COLOR=1` to disable
them. Programs launched by Task may have their own color settings.

### Surface failures {#error-annotations}

When `GITHUB_ACTIONS=true`, Task automatically emits an annotation when a task
fails, so the workflow can surface the error:

```text
::error title=Task 'build' failed::exit status 1
```

No Taskfile configuration is required for these annotations.

::: tip Related guide

<span id="ignore-errors"></span>

To continue after a selected failure, use `ignore_error`. See
[Errors and cleanup](./errors-and-cleanup.md#ignoring-command-errors) for its
scope and examples.

:::
