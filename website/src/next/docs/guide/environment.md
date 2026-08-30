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

Environment variables are set with `env`, which works at the root of the
Taskfile and on individual tasks.

## Task

You can use `env` to set custom environment variables for a specific task:

```yaml
version: '3'

tasks:
  greet:
    cmds:
      - echo $GREETING
    env:
      GREETING: Hey, there!
```

Additionally, you can set global environment variables that will be available to
all tasks:

```yaml
version: '3'

env:
  GREETING: Hey, there!

tasks:
  greet:
    cmds:
      - echo $GREETING
```

::: info

`env` supports expansion and retrieving output from a shell command just like
variables, as you can see in the [Variables](./variables.md) section.

:::

## .env files

You can also ask Task to include `.env` like files by using the `dotenv:`
setting:

::: code-group

```shell [.env]
KEYNAME=VALUE
```

```shell [testing/.env]
ENDPOINT=testing.com
```

:::

```yaml
version: '3'

env:
  ENV: testing

dotenv: ['.env', '{{.ENV}}/.env', '{{.HOME}}/.env']

tasks:
  greet:
    cmds:
      - echo "Using $KEYNAME and endpoint $ENDPOINT"
```

When the same variable is defined in multiple dotenv files, the **first file in
the list takes precedence**. This allows you to set up override patterns by
placing higher-priority files first:

```yaml
version: '3'

dotenv:
  - .env.local # Highest priority - local developer overrides
  - .env.{{.ENV}} # Environment-specific settings
  - .env # Base defaults (lowest priority)
```

Dotenv files can also be specified at the task level:

```yaml
version: '3'

env:
  ENV: testing

tasks:
  greet:
    dotenv: ['.env', '{{.ENV}}/.env', '{{.HOME}}/.env']
    cmds:
      - echo "Using $KEYNAME and endpoint $ENDPOINT"
```

Environment variables specified explicitly at the task-level will override
variables defined in dotfiles:

```yaml
version: '3'

env:
  ENV: testing

tasks:
  greet:
    dotenv: ['.env', '{{.ENV}}/.env', '{{.HOME}}/.env']
    env:
      KEYNAME: DIFFERENT_VALUE
    cmds:
      - echo "Using $KEYNAME and endpoint $ENDPOINT"
```

::: info

Please note that you are not currently able to use the `dotenv` key inside
included Taskfiles.

:::
