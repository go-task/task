---
title: Variable resolution
description:
  The single order Task uses to resolve a variable, and the consequences that
  surprise people most often.
section: Concepts
docType: concept
outline: deep
---

# Variable resolution

Task builds one flat set of variables for each task, just before running it.
Every source is applied to that set in a fixed order, and each one overwrites
what came before. There is no per-source scoping and no lookup chain at render
time: by the time a template runs, a name has exactly one value.

Understanding that single order explains almost every surprise on this page.

## The order

Applied first to last. Later wins.

| #   | Source                              | Set by                                                                        |
| --- | ----------------------------------- | ----------------------------------------------------------------------------- |
| 1   | The process environment             | the shell that ran `task`                                                     |
| 2   | Special variables                   | Task itself (`TASK`, `ROOT_DIR`, `CLI_ARGS`, …)                               |
| 3   | Taskfile `env:`                     | the `env:` block; `dotenv:` files fill only names `env:` does not already set |
| 4   | Global `vars:`                      | the `vars:` block of every Taskfile in the run                                |
| 5   | Include `vars:`                     | the `vars:` given on an `includes:` entry                                     |
| 6   | The included Taskfile's own `vars:` | the `vars:` block of the file being included                                  |
| 7   | Call variables                      | `task foo BAR=1`, or `vars:` on a `task:` command                             |
| 8   | The task's `vars:`                  | the `vars:` block of the task being run                                       |

## What this means in practice

### A task's own variables cannot be overridden from the command line

Step 8 comes after step 7, so a variable declared on the task always wins:

```yaml
version: '3'

tasks:
  greet:
    vars:
      NAME: from-task
    cmds:
      - echo "{{.NAME}}"
```

```shell
$ task greet NAME=from-cli
from-task
```

To let a caller supply a value, give the default somewhere earlier, in global
`vars:`, or use a template default:

```yaml
version: '3'

vars:
  NAME: from-global

tasks:
  greet:
    cmds:
      - echo "{{.NAME}}"
```

```shell
$ task greet NAME=from-cli
from-cli
```

### Variables on an `includes:` entry are defaults, not overrides

Step 6 comes after step 5, so the included Taskfile's own `vars:` win over the
values supplied where it is included. Passing `vars:` on an `includes:` entry
only takes effect for names the included Taskfile does not define itself.

If you are writing a Taskfile meant to be included and configured, leave the
configurable names out of `vars:` and give the default at the point of use
instead:

```yaml
version: '3'

tasks:
  build:
    cmds:
      - echo "building {{.DOCKER_IMAGE | default "app"}}"
```

Declaring `DOCKER_IMAGE` in that file's `vars:` would make every include site
that sets it silently get the declared value instead.

### Global variable names are shared across every Taskfile in the run

Global `vars:` are merged into one set before any task runs, so a name declared
in both the entrypoint and an included Taskfile resolves to the included one,
including for tasks defined in the entrypoint.

Give globals that belong to an included Taskfile a distinctive name, or move
them onto the tasks that use them, where step 8 keeps them local.

### `env:` and `vars:` are not the same thing

A `vars:` entry exists for templates only. `$FOO` in a command will not see it,
whichever level it was declared at.

`env:` is exported to the environment of the commands Task runs, so `$FOO`
works. Whether a template also sees it depends on where it was declared:

| Declared at              | <span v-pre>`{{.FOO}}`</span> | `$FOO` |
| ------------------------ | ----------------------------- | ------ |
| the root of the Taskfile | yes, it is step 3 above       | yes    |
| on a task                | **no, it renders empty**      | yes    |

A task's `env:` is assembled after the variable set has been resolved, so it
never takes part in the order on this page. Read a task-level value with `$FOO`,
or declare it in `vars:` if a template needs it.

## When values are computed

Dynamic variables (`sh:`) are executed while the set is being built, in the
order above. A `sh:` command can therefore only reference variables from an
earlier step, never a later one.

Results are cached for the run, keyed on the command string, so the same `sh:`
command appearing twice runs once.

To pass a variable without flattening it to text, an array or a map, use `ref:`
instead of <span v-pre>`{{ }}`</span>. A template renders a string; `ref:`
preserves the type.

## Related

- [Variables](../guide/variables.md): how to declare each kind.
- [Environment variables](../guide/environment.md): `env:` and `.env` files.
- [Including other Taskfiles](../guide/includes.md): namespaces and includes.
- [Taskfile Schema](../reference/schema.md): every key, with its type.
