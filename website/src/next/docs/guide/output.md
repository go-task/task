---
title: Output and logging
description:
  Choose how Task prints command output, silence it, ignore errors, and annotate
  failures in CI.
section: Guide
docType: guide
outline: deep
---

# Output and logging

By default Task streams each command's output straight through. These settings
change what reaches the terminal and how it is grouped.

## Output syntax

By default, Task just redirects the STDOUT and STDERR of the running commands to
the shell in real-time. This is good for having live feedback for logging
printed by commands, but the output can become messy if you have multiple
commands running simultaneously and printing lots of stuff.

To make this more customizable, there are currently three different output
options you can choose:

- `interleaved` (default)
- `group`
- `prefixed`

To choose another one, just set it to root in the Taskfile:

```yaml
version: '3'

output: 'group'

tasks:
  # ...
```

The `group` output will print the entire output of a command once after it
finishes, so you will not have live feedback for commands that take a long time
to run.

When using the `group` output, you can optionally provide a templated message to
print at the start and end of the group. This can be useful for instructing CI
systems to group all of the output for a given task, such as with
[GitHub Actions' `::group::` command](https://docs.github.com/en/actions/learn-github-actions/workflow-commands-for-github-actions#grouping-log-lines)
or
[Azure Pipelines](https://docs.microsoft.com/en-us/azure/devops/pipelines/scripts/logging-commands?expand=1&view=azure-devops&tabs=bash#formatting-commands).

```yaml
version: '3'

output:
  group:
    begin: '::group::{{.TASK}}'
    end: '::endgroup::'

tasks:
  default:
    cmds:
      - echo 'Hello, World!'
    silent: true
```

```shell
$ task default
::group::default
Hello, World!
::endgroup::
```

When using the `group` output, you may swallow the output of the executed
command on standard output and standard error if it does not fail (zero exit
code).

```yaml
version: '3'

silent: true

output:
  group:
    error_only: true

tasks:
  passes: echo 'output-of-passes'
  errors: echo 'output-of-errors' && exit 1
```

```shell
$ task passes
$ task errors
output-of-errors
task: Failed to run task "errors": exit status 1
```

The `prefix` output will prefix every line printed by a command with
`[task-name] ` as the prefix, but you can customize the prefix for a command
with the `prefix:` attribute:

```yaml
version: '3'

output: prefixed

tasks:
  default:
    deps:
      - task: print
        vars: { TEXT: foo }
      - task: print
        vars: { TEXT: bar }
      - task: print
        vars: { TEXT: baz }

  print:
    cmds:
      - echo "{{.TEXT}}"
    prefix: 'print-{{.TEXT}}'
    silent: true
```

```shell
$ task default
[print-foo] foo
[print-bar] bar
[print-baz] baz
```

::: tip

The `output` option can also be specified by the `--output` or `-o` flags.

:::

## Silent mode

Silent mode disables the echoing of commands before Task runs it. For the
following Taskfile:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "Print something"
```

Normally this will be printed:

```shell
echo "Print something"
Print something
```

With silent mode on, the below will be printed instead:

```shell
Print something
```

There are four ways to enable silent mode:

- At command level:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - cmd: echo "Print something"
        silent: true
```

- At task level:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "Print something"
    silent: true
```

- Globally at Taskfile level:

```yaml
version: '3'

silent: true

tasks:
  echo:
    cmds:
      - echo "Print something"
```

- Or globally with `--silent` or `-s` flag

If you want to suppress STDOUT instead, just redirect a command to `/dev/null`:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "This will print nothing" > /dev/null
```

## Ignore errors

You have the option to ignore errors during command execution. Given the
following Taskfile:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - exit 1
      - echo "Hello World"
```

Task will abort the execution after running `exit 1` because the status code `1`
stands for `EXIT_FAILURE`. However, it is possible to continue with execution
using `ignore_error`:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - cmd: exit 1
        ignore_error: true
      - echo "Hello World"
```

`ignore_error` can also be set for a task, which means errors will be suppressed
for all commands. Nevertheless, keep in mind that this option will not propagate
to other tasks called either by `deps` or `cmds`!

## CI Integration

### Colored output

Task automatically enables colored output when running in CI environments
(`CI=true`). Most CI providers set this variable automatically.

You can also force colored output with `FORCE_COLOR=1` or disable it with
`NO_COLOR=1`.

### Error annotations

When running in GitHub Actions (`GITHUB_ACTIONS=true`), Task automatically emits
error annotations when a task fails. These annotations appear in the workflow
summary, making it easier to spot failures without scrolling through logs.

```shell
::error title=Task 'build' failed::exit status 1
```

This feature requires no configuration and works automatically.
