---
title: Loops
description:
  Repeat a command over a static list, a matrix, a variable, your task's
  sources, or other tasks.
section: Guide
docType: guide
outline: deep
---

# Loops

Use `for` to repeat a command or task call with different values. Start with a
list, then choose a different input source when the values come from a variable,
files, or several combinations.

## Repeat a command {#looping-over-a-static-list}

In a project containing `README.md` and `LICENSE`, this task saves a copy of
each file in `backup/`. Put `for` beside `cmd` and read the current filename
through `.ITEM`:

```yaml
version: '3'

tasks:
  backup:
    cmds:
      - mkdir -p backup
      - for: [README.md, LICENSE]
        cmd: cp '{{.ITEM}}' backup/
```

Run `task backup` to create `backup/README.md` and `backup/LICENSE`. Command
loops run in list order. Use [dependency loops](#looping-over-dependencies) for
iterations that may run concurrently.

For a changing set of files, use
[a source glob](#looping-over-your-task-s-sources-or-generated-files) instead of
maintaining a fixed list.

## Read values from variables {#looping-over-variables}

Use `var` to read an existing variable. Strings are split on whitespace:

```yaml
version: '3'

tasks:
  greet:
    vars:
      NAMES: Alice Bob
    cmds:
      - for: { var: NAMES }
        cmd: echo "Hello, {{.ITEM}}!"
```

For another separator, set `split`:

```yaml
version: '3'

tasks:
  greet:
    vars:
      NAMES: Alice,Bob
    cmds:
      - for: { var: NAMES, split: ',' }
        cmd: echo "Hello, {{.ITEM}}!"
```

Arrays preserve values containing spaces:

```yaml
version: '3'

tasks:
  greet:
    vars:
      NAMES: ['Alice Smith', 'Bob Jones']
    cmds:
      - for: { var: NAMES }
        cmd: echo "Hello, {{.ITEM}}!"
```

Maps expose the key as `.KEY` and the value as `.ITEM`. Their iteration order is
not guaranteed:

```yaml
version: '3'

tasks:
  services:
    vars:
      PORTS:
        map: { api: 8080, worker: 9090 }
    cmds:
      - for: { var: PORTS }
        cmd: echo "{{.KEY}} uses port {{.ITEM}}"
```

Dynamic variables work too. This example reads newline-separated names from an
existing `names.txt` file:

```yaml
version: '3'

tasks:
  greet:
    vars:
      NAMES:
        sh: cat names.txt
    cmds:
      - for: { var: NAMES, split: "\n" }
        cmd: echo "Hello, {{.ITEM}}!"
```

## Name the current item {#renaming-variables}

Use `as` to name the current item when that makes a command easier to read:

```yaml
version: '3'

tasks:
  greet:
    cmds:
      - for: { var: NAMES, as: NAME }
        cmd: echo "Hello, {{.NAME}}!"
    vars:
      NAMES: [Alice, Bob]
```

## Repeat a task call {#looping-over-tasks}

Put `task` beside `for` to reuse another task for each item:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - for: [api, worker]
        task: check
        vars:
          SERVICE: '{{.ITEM}}'

  check:
    cmds:
      - echo "Checking {{.SERVICE}}"
```

`task` checks `api` before `worker`. Each call finishes before the next starts.
You can also template the task name itself:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - for: [api, worker]
        task: check:{{.ITEM}}

  check:api: echo 'Checking API'
  check:worker: echo 'Checking worker'
```

## Run iterations in parallel {#looping-over-dependencies}

Use a loop in `deps` when the calls can run concurrently:

```yaml
version: '3'

tasks:
  default:
    deps:
      - for: [api, worker]
        task: check
        vars:
          SERVICE: '{{.ITEM}}'

  check:
    cmds:
      - echo "Checking {{.SERVICE}}"
```

The messages may appear in either order. All the input forms shown on this page
also work with dependency loops. See
[Dependencies and task calls](./dependencies.md) for concurrency limits and
repeated-call policies.

## Try every combination {#looping-over-a-matrix}

Use `matrix` to visit every combination of several lists:

```yaml
version: '3'

tasks:
  targets:
    cmds:
      - for:
          matrix:
            OS: [linux, darwin]
            ARCH: [amd64, arm64]
        cmd: echo "{{.ITEM.OS}}/{{.ITEM.ARCH}}"
```

`task targets` prints `linux/amd64`, `linux/arm64`, `darwin/amd64`, and
`darwin/arm64`. These are ordinary iteration values; they do not change the
platform on which Task runs.

Use `ref` to reuse lists defined elsewhere:

```yaml
version: '3'

vars:
  SYSTEMS: [linux, darwin]
  ARCHITECTURES: [amd64, arm64]

tasks:
  targets:
    cmds:
      - for:
          matrix:
            OS: { ref: .SYSTEMS }
            ARCH: { ref: .ARCHITECTURES }
        cmd: echo "{{.ITEM.OS}}/{{.ITEM.ARCH}}"
```

## Loop over files {#looping-over-your-task-s-sources-or-generated-files}

Use `for: sources` or `for: generates` to reuse the task's file declarations.
Globs are expanded before iteration:

::: code-group

```yaml [Sources]
version: '3'

tasks:
  show:
    sources: ['src/*.txt']
    cmds:
      - for: sources
        cmd: cat '{{.ITEM}}'
```

```yaml [Generates]
version: '3'

tasks:
  show:
    generates: ['output/*.txt']
    cmds:
      - for: generates
        cmd: cat '{{.ITEM}}'
```

:::

Run `task show` with matching files present. Paths are relative to the task's
working directory. Source declarations also enable
[up-to-date checks](./up-to-date.md); use `method: none` if this task should
print the files every time.

To build an absolute path, combine a directory with the item using the
`joinPath` template function, for example
<span v-pre>`{{joinPath .TASK_DIR .ITEM}}`</span>. See the
[file-path variables](../reference/templating.md#file-paths) for the available
directory values.
