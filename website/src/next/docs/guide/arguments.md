---
title: Command-line arguments
description:
  Supply variables with NAME=value, forward arguments after --, and capture
  values with wildcard task names.
section: Guide
docType: guide
outline: deep
---

# Command-line arguments {#passing-arguments}

Choose how to pass input based on what needs to receive it:

| Input                   | Example                | Read it with                            |
| ----------------------- | ---------------------- | --------------------------------------- |
| A Task variable         | `task greet NAME=Bob`  | <span v-pre>`{{.NAME}}`</span>          |
| Arguments for a command | `task yarn -- install` | <span v-pre>`{{.CLI_ARGS}}`</span>      |
| Part of a task name     | `task start:api`       | <span v-pre>`{{index .MATCH 0}}`</span> |

## Pass named values {#variable-assignments}

Pass `NAME=value` to supply a Task variable. A default in the root `vars` block
can be overridden from the command line:

```yaml
version: '3'

vars:
  NAME: World

tasks:
  greet:
    cmds:
      - echo "Hello, {{.NAME}}!"
```

```shell
task greet NAME=Bob
task greet "NAME=Jane Doe"
```

Quote assignments that contain spaces. These assignments work across shells,
including Windows, and are shared by all tasks in the invocation:

```shell
task build test ENV=production
```

A value declared in a task's own `vars` can override the command-line value. See
[variable resolution](./variables.md#resolution-order) for precedence and
configurable defaults. To set values in a command's shell environment, see
[Environment variables](./environment.md).

## Forward command arguments {#forwarding-cli-arguments-to-commands}

Everything after `--` is available in `.CLI_ARGS`, including values containing
`=`. Insert it into a command to forward those arguments. This example runs
`yarn install`:

```shell
$ task yarn -- install
```

```yaml
version: '3'

tasks:
  yarn:
    cmds:
      - yarn {{.CLI_ARGS}}
```

## Capture task-name inputs {#wildcard-arguments}

Use `*` in a task name to turn a call such as `task start:api` into an input.
Task captures each wildcard in the `.MATCH` array. Give the captured values
names with `vars` so their meaning stays clear:

```yaml
version: '3'

tasks:
  start:*:*:
    vars:
      SERVICE: '{{index .MATCH 0}}'
      REPLICAS: '{{index .MATCH 1}}'
    cmds:
      - echo "Starting {{.SERVICE}} with {{.REPLICAS}} replicas"

  start:*:
    vars:
      SERVICE: '{{index .MATCH 0}}'
    cmds:
      - echo "Starting {{.SERVICE}}"
```

The first captured value becomes `SERVICE`:

```shell
$ task start:foo
Starting foo
```

You can use whitespace in your arguments as long as you quote the task name:

```shell
$ task "start:foo bar"
Starting foo bar
```

If multiple matching tasks are found, the first one listed in the Taskfile will
be used. If you are using included Taskfiles, tasks in parent files will be
considered first.

```shell
$ task start:foo:3
Starting foo with 3 replicas
```

Wildcard task names also work with aliases:

```yaml
version: '3'

tasks:
  start:*:
    aliases: [run:*]
    vars:
      SERVICE: '{{index .MATCH 0}}'
    cmds:
      - echo "Running {{.SERVICE}}"
```

Call the task through the `run:*` alias:

```shell
$ task run:foo
Running foo
```
