---
title: Passing arguments
description:
  Forward command line arguments to a task with `--`, and match part of a task's
  name with a wildcard.
section: Guide
docType: guide
outline: deep
---

# Passing arguments

Tasks can take input from the command line in two ways: everything after `--`,
or a pattern in the task name itself.

## Forwarding CLI arguments to commands

If `--` is given in the CLI, all following parameters are added to a special
`.CLI_ARGS` variable. This is useful to forward arguments to another command.

The below example will run `yarn install`.

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

## Wildcard arguments

Another way to parse arguments into a task is to use a wildcard in your task's
name. Wildcards are denoted by an asterisk (`*`) and can be used multiple times
in a task's name to pass in multiple arguments.

Matching arguments will be captured and stored in the `.MATCH` variable and can
then be used in your task's commands like any other variable. This variable is
an array of strings and so will need to be indexed to access the individual
arguments. We suggest creating a named variable for each argument to make it
clear what they contain:

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

This call matches the `start:*` task and the string "foo" is captured by the
wildcard and stored in the `.MATCH` variable. We then index the `.MATCH` array
and store the result in the `.SERVICE` variable which is then echoed out in the
cmds:

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

Using wildcards with aliases Wildcards also work with aliases. If a task has an
alias, you can use the alias name with wildcards to capture arguments. For
example:

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

In this example, you can call the task using the alias run:\*:

```shell
$ task run:foo
Running foo
```
