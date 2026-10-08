---
title: Environment variables
description:
  Set environment variables on a single task or on every task, and load them
  from `.env` files.
section: Guide
docType: guide
outline: deep
---

# Environment variables

Use `env` for values a program reads from its environment. In this example, the
shell reads `GREETING` with `$GREETING`:

```yaml
version: '3'

tasks:
  greet:
    env:
      GREETING: Hello, World!
    cmds:
      - echo "$GREETING"
```

Run `task greet` to print the greeting. Task uses the same shell syntax on
supported platforms, including Windows. For values used in Task templates, use
[Variables](./variables.md#env-and-vars-are-not-the-same-thing).

## Set values for commands {#task}

Put `env` on a task to configure its commands, or at the root to provide values
for every task:

```yaml
version: '3'

env:
  GREETING: Hello, World!

tasks:
  greet:
    cmds:
      - echo "$GREETING"

  greet:local:
    env:
      GREETING: Hello from this task!
    cmds:
      - echo "$GREETING"
```

With no `GREETING` in the process environment, `task greet` uses the root value
and `task greet:local` uses the task's value.

By default, a variable already present in the process environment takes
precedence over Taskfile `env` values. If the result differs from the example,
check the environment used to launch Task. The
[Env Precedence experiment](../experiments/env-precedence.md) changes this rule.

`env` values support templates and dynamic `sh` commands. For example, an entry
<span v-pre>`BUILD_MODE: '{{.MODE}}'`</span> exports a Task variable to a
command's environment. A `vars` entry alone is not exported to commands in
`cmds`.

## Load values from files {#env-files}

Use `dotenv` to load values from files instead of repeating them in the
Taskfile. Create these two files in your project:

::: code-group

```dotenv [.env]
GREETING=Hello from .env!
```

```yaml [Taskfile.yml]
version: '3'

dotenv: ['.env']

tasks:
  greet:
    cmds:
      - echo "$GREETING"
```

:::

Run `task greet` to use the value from `.env`, assuming the process environment
does not already set `GREETING`.

### Choose file precedence

When several dotenv files define a name, the **first file wins**. Put local
overrides before shared defaults:

```yaml
version: '3'

vars:
  ENV: development

tasks:
  greet:
    dotenv:
      - .env.local
      - .env.{{.ENV}}
      - .env
    cmds:
      - echo "$GREETING"
```

`task greet ENV=testing` selects `.env.testing` for the middle entry. This
example puts `dotenv` on the task so it can use call variables; root-level
dotenv files are loaded before those variables are available. Missing dotenv
files are skipped. Keep files containing local credentials out of version
control.

### Load values for one task

Put `dotenv` on a task when only that task needs the file. Explicit task-level
`env` values take precedence over values loaded from dotenv files:

```yaml
version: '3'

tasks:
  greet:
    dotenv: ['.env']
    env:
      GREETING: Hello from the task!
    cmds:
      - echo "$GREETING"
```

The process environment still takes precedence by default. Dotenv paths can
contain templates, as in <span v-pre>`'{{.HOME}}/.env'`</span>.

### Load values for an included Taskfile

Included Taskfiles can also declare `dotenv`. These variables are available only
to tasks defined in that Taskfile. Nested includes use their own dotenv values.
For example, a common Taskfile included by an application does not inherit the
application's dotenv values. The root Taskfile's dotenv remains available to all
tasks for compatibility.

Relative paths are resolved from the Taskfile declaring `dotenv`, independently
of the include's `dir`. Path templates can use that Taskfile's variables and the
variables supplied by its include. Remote Taskfiles read dotenv files locally
relative to the root execution directory.

An included Taskfile's dotenv values take precedence over the root's values for
its own tasks, and the first file in each list takes precedence. Explicit
Taskfile `env` values override dotenv values; task-level `dotenv` and `env`
retain their existing precedence.
The existing merge behavior of `vars` and `env` declarations is unchanged.

See [Including Taskfiles](./includes.md).
