---
title: Variables
description:
  Declare variables, compute values with shell commands, preserve types with
  references, and understand resolution order.
section: Guide
docType: guide
outline: deep
---

# Variables

Declare variables with `vars`, then read them in a command with a template:

```yaml
version: '3'

vars:
  NAME: World

tasks:
  greet:
    cmds:
      - echo "Hello, {{.NAME}}!"
```

Run `task greet` to print `Hello, World!`, or `task greet NAME=Bob` to print
`Hello, Bob!`. This example puts the default at the root of the Taskfile so
callers can override it.

Use a literal for a fixed value, `sh` to compute text with a command, and `ref`
to preserve arrays and maps. The examples come first;
[resolution order](#resolution-order) explains how values from different places
interact.

<span id="command-line-values"></span>

Pass `NAME=value` on the command line to supply variables to Task. See
[Command-line arguments](./arguments.md#variable-assignments) for quoting,
multiple tasks and the difference from arguments after `--`.

::: tip

The special variable `.TASK` contains the name of the task being run.

:::

## Choose a value type {#variable-types}

Variables support the following types:

- `string`
- `bool`
- `int`
- `float`
- `array`
- `map`

::: info

Defining a map requires that you use a special `map` subkey (see example below).

:::

```yaml
version: 3

tasks:
  foo:
    vars:
      STRING: 'Hello, World!'
      BOOL: true
      INT: 42
      FLOAT: 3.14
      ARRAY: [1, 2, 3]
      MAP:
        map: { A: 1, B: 2, C: 3 }
    cmds:
      - 'echo {{.STRING}}' # Hello, World!
      - 'echo {{.BOOL}}' # true
      - 'echo {{.INT}}' # 42
      - 'echo {{.FLOAT}}' # 3.14
      - 'echo {{.ARRAY}}' # [1 2 3]
      - 'echo {{index .ARRAY 0}}' # 1
      - 'echo {{.MAP}}' # map[A:1 B:2 C:3]
      - 'echo {{.MAP.A}}' # 1
```

## Compute a value {#dynamic-variables}

Use `sh` to compute a variable from a shell command. Task captures its output
and trims the final trailing newline, if present.

```yaml
version: '3'

tasks:
  build:
    cmds:
      - go build -ldflags="-X main.Version={{.GIT_COMMIT}}" main.go
    vars:
      GIT_COMMIT:
        sh: git log -n 1 --format=%h
```

The result is text. To convert JSON or YAML output into structured data, use
[parsing functions with `ref`](#parsing-json-yaml-into-map-variables).

## Pass lists and maps {#referencing-other-variables}

Use `ref` to pass an array or map to another task without converting it to text.
In this example, `show` receives the original list:

```yaml
version: '3'

tasks:
  default:
    vars:
      SERVICES: [api, worker]
    cmds:
      - task: show
        vars:
          SERVICES:
            ref: .SERVICES

  show:
    cmds:
      - echo '{{index .SERVICES 0}}'
```

Run `task` to print `api`. Replacing the reference with
<span v-pre>`SERVICES: '{{.SERVICES}}'`</span> would pass the string
`[api worker]`. Indexing that string returns a byte, not a list element.

References also work in dependency calls and variable definitions:

```yaml
version: '3'

tasks:
  default:
    vars:
      SERVICES: [api, worker]
      SELECTED:
        ref: .SERVICES
    deps:
      - task: show
        vars:
          SERVICE:
            ref: index .SELECTED 0

  show:
    cmds:
      - echo '{{.SERVICE}}'
```

Use subkeys, indexes, and functions inside `ref`, without the template braces.
See the [Templating reference][templating-reference] for available functions.

## Parse JSON or YAML {#parsing-json-yaml-into-map-variables}

If you have a raw JSON or YAML string that you want to process in Task, you can
use a combination of the `ref` keyword and the `fromJson` or `fromYaml`
templating functions to parse the string into a map variable. For example:

```yaml
version: '3'

tasks:
  task-with-map:
    vars:
      JSON: '{"a": 1, "b": 2, "c": 3}'
      FOO:
        ref: 'fromJson .JSON'
    cmds:
      - echo {{.FOO}}
```

```txt
map[a:1 b:2 c:3]
```

## Understand precedence {#resolution-order}

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

## Apply precedence rules {#what-this-means-in-practice}

### Let callers override {#a-task-s-own-variables-cannot-be-overridden-from-the-command-line}

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

To keep the default on the task itself, read the caller's value in a template:

```yaml
vars:
  NAME: '{{.NAME | default "World"}}'
```

### Configure include defaults {#let-callers-configure-an-included-taskfile}

Step 6 comes after step 5, so a constant declared in the included Taskfile's
`vars:` overrides the value supplied by `includes.vars`.

To keep a configurable default in the included Taskfile, use a template that
reads the previously resolved value:

```yaml
version: '3'

vars:
  DOCKER_IMAGE: '{{.DOCKER_IMAGE | default "app"}}'

tasks:
  build:
    cmds:
      - echo "building {{.DOCKER_IMAGE}}"
```

An inclusion that supplies `DOCKER_IMAGE: backend_image` now prints
`building backend_image`; without a supplied value, it prints `building app`.
The default stays in one place and applies to every task in the included file.

### Avoid global collisions {#global-variable-names-are-shared-across-every-taskfile-in-the-run}

Global `vars:` are merged into one set before any task runs, so a name declared
in both the entrypoint and an included Taskfile resolves to the included one,
including for tasks defined in the entrypoint.

Give globals that belong to an included Taskfile a distinctive name, or move
them onto the tasks that use them, where step 8 keeps them local.

### Choose vars or env {#env-and-vars-are-not-the-same-thing}

A `vars:` entry is not exported to commands in `cmds`, whichever level it was
declared at. Read it with a template instead of `$FOO`. Dynamic variables
(`sh:`) have a different environment, described below.

`env:` is exported to the environment of the commands Task runs, so `$FOO`
works. Whether a template also sees it depends on where it was declared:

| Declared at              | <span v-pre>`{{.FOO}}`</span>              | `$FOO` |
| ------------------------ | ------------------------------------------ | ------ |
| the root of the Taskfile | yes, it is step 3 above                    | yes    |
| on a task                | keeps the value from other sources, if any | yes    |

A task's `env:` is assembled after the variable set has been resolved, so it
never takes part in the order on this page. If global `env:` sets `FOO: root`
and a task's `env:` sets `FOO: task`, the template <span v-pre>`{{.FOO}}`</span>
renders `root`, while `$FOO` in a command reads `task` (assuming the process
environment does not already set `FOO`). The template only renders empty if no
other source defines the variable. Read a task-level environment value with
`$FOO`, or declare it in `vars:` if a template needs it.

## Understand evaluation {#when-values-are-computed}

Dynamic variables (`sh:`) are executed while the set is being built, in the
order above. Within a block, variables are resolved in declaration order, so a
`sh:` command can also reference variables declared earlier in the same block:

```yaml
vars:
  INPUT: hello
  OUTPUT:
    sh: echo {{.INPUT}}
```

Here `OUTPUT` resolves to `hello`. Variables from later declarations or later
steps are not available yet.

The shell used by `sh:` also receives previously resolved scalar variables, so
`sh: echo $INPUT` works in this example too. The process environment takes
precedence for shell lookups unless the
[Env Precedence experiment](../experiments/env-precedence.md) is enabled.
Commands in `cmds` only receive the process environment and Task's `env:` and
`dotenv:` values; they do not inherit `vars:` this way.

Results are cached for the run, keyed on the command string, so the same `sh:`
command appearing twice runs once.

To pass a variable without flattening it to text, an array or a map, use `ref:`
instead of <span v-pre>`{{ }}`</span>. A template renders a string; `ref:`
preserves the type.

::: tip Related guide

<span id="secret-variables"></span>

Mark sensitive values with `secret: true` to mask them in Task's own command
logs. See [Secret variables](./secret-variables.md) for examples, loading values
from external sources and the limits of masking.

:::

[templating-reference]: ../reference/templating.md
