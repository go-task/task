---
title: Required variables and prompts
description:
  Require variables to be set, restrict them to a list of allowed values, and
  prompt for them interactively.
section: Guide
docType: guide
outline: deep
---

# Required variables and prompts

A task can refuse to run until it has what it needs, and it can ask the caller
for it.

## Ensuring required variables are set

If you want to check that certain variables are set before running a task then
you can use `requires`. This is useful when might not be clear to users which
variables are needed, or if you want clear message about what is required. Also
some tasks could have dangerous side effects if run with un-set variables.

Using `requires` you specify an array of strings in the `vars` sub-section under
`requires`, these strings are variable names which are checked prior to running
the task. If any variables are un-set then the task will error and not run.

Environmental variables are also checked.

Syntax:

```yaml
requires:
  vars: [] # Array of strings
```

::: info

Variables set to empty zero length strings, will pass the `requires` check.

:::

Example of using `requires`:

```yaml
version: '3'

tasks:
  docker-build:
    cmds:
      - 'docker build . -t {{.IMAGE_NAME}}:{{.IMAGE_TAG}}'

    # Make sure these variables are set before running
    requires:
      vars: [IMAGE_NAME, IMAGE_TAG]
```

## Ensuring required variables have allowed values

If you want to ensure that a variable is set to one of a predefined set of valid
values before executing a task, you can use requires. This is particularly
useful when there are strict requirements for what values a variable can take,
and you want to provide clear feedback to the user when an invalid value is
detected.

To use `requires`, you specify an array of allowed values in the vars
sub-section under requires. Task will check if the variable is set to one of the
allowed values. If the variable does not match any of these values, the task
will raise an error and stop execution.

This check applies both to user-defined variables and environment variables.

Example of using `requires`:

```yaml
version: '3'

tasks:
  deploy:
    cmds:
      - echo "deploying to {{.ENV}}"

    requires:
      vars:
        - name: ENV
          enum: [dev, beta, prod]
```

If `ENV` is not one of 'dev', 'beta' or 'prod' an error will be raised.

::: info

This is supported only for string variables.

:::

## Using variable references for enum values

Instead of hardcoding enum values, you can reference a variable containing the
allowed values. This is useful when you want to define allowed values once and
reuse them, or when the values are computed dynamically.

Use the `ref` key to reference a variable:

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

You can also use template expressions to transform the value:

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

Or generate values dynamically from a shell command:

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

## Prompting for missing variables interactively

If you want Task to prompt users for missing required variables instead of
failing, you can enable interactive mode in your `.taskrc.yml`:

```yaml
# ~/.taskrc.yml
interactive: true
```

When enabled, Task will display an interactive prompt for any missing required
variable. For variables with an `enum`, a selection menu is shown. For variables
without an enum, a text input is displayed.

```yaml
# Taskfile.yml
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

```shell
$ task deploy
? Select value for ENVIRONMENT:
❯ dev
  staging
  prod
? Enter value for VERSION: 1.0.0
Deploying 1.0.0 to prod
```

If the variable is already set (via CLI, environment, or Taskfile), no prompt is
shown:

```shell
$ task deploy ENVIRONMENT=prod VERSION=1.0.0
Deploying 1.0.0 to prod
```

::: info

Interactive prompts require a TTY (terminal). Task automatically detects
non-interactive environments like GitHub Actions, GitLab CI, and other CI
pipelines where stdin/stdout are not connected to a terminal. In these cases,
prompts are skipped and missing variables will cause an error as usual.

You can enable prompts from the command line with `--interactive` or by setting
`interactive: true` in your `.taskrc.yml`.

:::

## Warning Prompts

Warning Prompts are used to prompt a user for confirmation before a task is
executed.

Below is an example using `prompt` with a dangerous command, that is called
between two safe commands:

```yaml
version: '3'

tasks:
  example:
    cmds:
      - task: not-dangerous
      - task: dangerous
      - task: another-not-dangerous

  not-dangerous:
    cmds:
      - echo 'not dangerous command'

  another-not-dangerous:
    cmds:
      - echo 'another not dangerous command'

  dangerous:
    prompt: This is a dangerous command... Do you want to continue?
    cmds:
      - echo 'dangerous command'
```

```shell
❯ task dangerous
task: "This is a dangerous command... Do you want to continue?" [y/N]
```

Prompts can be a single value or a list of prompts, like below:

```yaml
version: '3'

tasks:
  example:
    cmds:
      - task: dangerous

  dangerous:
    prompt:
      - This is a dangerous command... Do you want to continue?
      - Are you sure?
    cmds:
      - echo 'dangerous command'
```

Warning prompts are called before executing a task. If a prompt is denied Task
will exit with [exit code](../reference/cli.md#exit-codes) 205. If approved,
Task will continue as normal.

```shell
❯ task example
not dangerous command
task: "This is a dangerous command. Do you want to continue?" [y/N]
y
dangerous command
another not dangerous command
```

To skip warning prompts automatically, you can use the `--yes` (alias `-y`)
option when calling the task. By including this option, all warnings, will be
automatically confirmed, and no prompts will be shown.

::: warning

Tasks with prompts always fail by default on non-terminal environments, like a
CI, where an `stdin` won't be available for the user to answer. In those cases,
use `--yes` (`-y`) to force all tasks with a prompt to run.

:::
