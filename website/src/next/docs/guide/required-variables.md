---
title: Validation and prompts
description:
  Require variables to be set, restrict them to allowed values, and prompt for
  missing input.
section: Guide
docType: guide
outline: deep
---

# Validation and prompts {#required-variables-and-prompts}

Use `requires` to catch missing inputs before a task runs. Add `enum` when only
certain values are allowed, and enable interactive prompts when users should be
able to fill in missing values at the terminal.

## Require an input {#ensuring-required-variables-are-set}

Declare inputs in `requires.vars`, then pass them on the command line:

```yaml
version: '3'

tasks:
  release:
    requires:
      vars: [VERSION]
    cmds:
      - echo "Preparing release {{.VERSION}}"
```

Run `task release VERSION=1.2.3` to print the message. Without `VERSION`, Task
fails and identifies the missing input. Task variables and values inherited from
the process environment can satisfy the requirement.

A variable set to an empty string counts as present. To reject an empty value,
use an enum of nonempty strings or a
[precondition](./conditional-execution.md#using-programmatic-checks-to-cancel-the-execution-of-a-task-and-its-dependencies).

## Restrict allowed values {#ensuring-required-variables-have-allowed-values}

Use `name` and `enum` to require a string from a fixed list:

```yaml
version: '3'

tasks:
  deploy:
    requires:
      vars:
        - name: ENV
          enum: [dev, staging, prod]
    cmds:
      - echo "Deploying to {{.ENV}}"
```

`task deploy ENV=staging` succeeds. `task deploy ENV=preview` fails before
running the command. Enum validation applies to string variables, including
strings inherited from the environment.

## Ask for missing input {#prompting-for-missing-variables-interactively}

Run `task --interactive deploy` with the example above to choose the missing
`ENV` from a menu. Required variables without an enum use a text input instead.

To enable this behavior by default, set it in your
[Task configuration](../reference/config.md#interactive):

```yaml [~/.taskrc.yml]
interactive: true
```

For a task with both kinds of input:

```yaml
version: '3'

tasks:
  deploy:
    requires:
      vars:
        - name: ENVIRONMENT
          enum: [dev, staging, prod]
        - VERSION
    cmds:
      - echo "Deploying {{.VERSION}} to {{.ENVIRONMENT}}"
```

`task --interactive deploy` asks for an environment and a version. Values
already supplied through the CLI, environment, or Taskfile do not prompt:

```shell
task deploy ENVIRONMENT=prod VERSION=1.0.0
```

Prompts require a terminal. In CI or another non-interactive environment,
missing variables cause an error; supply all required values explicitly.

## Reuse allowed values {#using-variable-references-for-enum-values}

Define a shared list and reference it from an enum:

```yaml
version: '3'

vars:
  ALLOWED_ENVS: [dev, staging, prod]

tasks:
  deploy:
    requires:
      vars:
        - name: ENV
          enum:
            ref: .ALLOWED_ENVS
    cmds:
      - echo "Deploying to {{.ENV}}"
```

References can also transform values. Given a `config.json` file containing
`{"allowed_environments": ["dev", "prod"]}`, this reads the list from it:

```yaml
version: '3'

vars:
  CONFIG:
    sh: cat config.json

tasks:
  deploy:
    requires:
      vars:
        - name: ENV
          enum:
            ref: ( .CONFIG | fromJson ).allowed_environments
    cmds:
      - echo "Deploying to {{.ENV}}"
```

For newline-separated command output, convert it to a list before using it as an
enum. This example uses the entries in an existing `services/` directory:

```yaml
version: '3'

vars:
  AVAILABLE_SERVICES:
    sh: ls services/

tasks:
  deploy:
    requires:
      vars:
        - name: SERVICE
          enum:
            ref: .AVAILABLE_SERVICES | splitLines | compact
    cmds:
      - echo "Deploying {{.SERVICE}}"
```

::: tip Related guide

<span id="warning-prompts"></span>

A prompt for input fills in a value. A
[confirmation prompt](./conditional-execution.md#confirmation-prompts) asks
whether execution should proceed, even if all input values are already present.

:::
