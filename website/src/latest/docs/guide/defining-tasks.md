---
title: Defining tasks
description:
  Task syntax shortcuts, internal tasks, aliases, the directory a task runs in,
  and the help text Task shows for it.
section: Guide
docType: guide
outline: deep
searchKeywords:
  task-aliases: [aliases, task aliases]
---

# Defining tasks

Define tasks under `tasks`. Each task has a name and a list of commands in
`cmds`. Add `desc` to explain what a task does:

```yaml
version: '3'

tasks:
  greet:
    desc: Print a greeting
    cmds:
      - echo "Hello, World!"
```

Run `task greet` to execute it, or `task --list` to see its description. See
[Running tasks](./running-tasks.md) for selecting a Taskfile and previewing
commands.

## Choose a task name

The name under `tasks` is the name you pass to the CLI. Call your main task
`default` if it should run when someone types `task` without a name.

Use `desc` for the short explanation shown by `task --list`, `aliases` for
alternative names, and `label` to customize log messages. The sections below
show each independently.

## Use command shortcuts {#short-task-syntax}

Use shorthand syntax for tasks that only need commands. Use the full form when
you need properties such as `env`, `vars`, `desc` or `silent`:

```yaml
version: '3'

tasks:
  build: go build -v -o ./app{{exeExt}} .

  run:
    - task: build
    - ./app{{exeExt}} -h localhost -p 8080
```

## Keep helpers internal {#internal-tasks}

Set `internal: true` on a helper that should only be called by other tasks. It
stays out of `task --list` and `task --list-all`, and a direct CLI call fails.
Expose a public task that supplies the helper's inputs:

```yaml
version: '3'

tasks:
  build-image-1:
    cmds:
      - task: build-image
        vars:
          DOCKER_IMAGE: image-1

  build-image:
    internal: true
    cmds:
      - docker build -t {{.DOCKER_IMAGE}} .
```

## Set the working directory {#task-directory}

Set `dir` when commands need to run in a subdirectory. Relative paths are
resolved from the Taskfile's execution directory:

```yaml
version: '3'

tasks:
  serve:
    dir: public/www
    cmds:
      # run http server
      - caddy
```

Run `task serve` to start the server in `public/www`. Task creates the directory
if it does not exist. For included Taskfiles, configure the
[include directory](./includes.md#directory-of-included-taskfile).

## Add shorter names {#task-aliases}

Aliases are alternative names for tasks. They can be used to make it easier and
quicker to run tasks with long or hard-to-type names. You can use them on the
command line, when [calling sub-tasks](./dependencies.md#calling-another-task)
in your Taskfile and when [including tasks](./includes.md) with aliases from
another Taskfile. They can also be used together with
[namespace aliases](./includes.md#namespace-aliases).

```yaml
version: '3'

tasks:
  generate:
    aliases: [gen]
    cmds:
      - task: gen-mocks

  generate-mocks:
    aliases: [gen-mocks]
    cmds:
      - echo "generating..."
```

## Customize log labels {#overriding-task-name}

Use `label` to customize the name shown in summaries and status messages. Labels
can contain templates. They do not rename the task you call:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - task: print
        vars:
          MESSAGE: hello
      - task: print
        vars:
          MESSAGE: world

  print:
    label: 'print-{{.MESSAGE}}'
    cmds:
      - echo "{{.MESSAGE}}"
```

## Describe public tasks {#help}

Running `task --list` (or `task -l`) lists all tasks with a description. The
following Taskfile:

```yaml
version: '3'

tasks:
  build:
    desc: Build the Go binary.
    cmds:
      - go build -v main.go

  test:
    desc: Run all Go tests.
    cmds:
      - go test -race ./...

  js:
    cmds:
      - esbuild --bundle --minify js/index.js > public/bundle.js

  css:
    cmds:
      - esbuild --bundle --minify css/index.css > public/bundle.css
```

would print the following output:

```shell
* build:   Build the Go binary.
* test:    Run all Go tests.
```

Use `--list-all` (or `-a`) to also list tasks without a description. Internal
tasks remain hidden.

## Explain a task in detail {#display-summary-of-task}

Running `task --summary task-name` will show a summary of a task. The following
Taskfile:

```yaml
version: '3'

tasks:
  release:
    deps: [build]
    summary: |
      Release your project to GitHub

      It will build your project before starting the release.
      Please make sure that you have set GITHUB_TOKEN before starting.
    cmds:
      - your-release-tool

  build:
    cmds:
      - your-build-tool
```

`task --summary release` prints:

```
task: release

Release your project to GitHub

It will build your project before starting the release.
Please make sure that you have set GITHUB_TOKEN before starting.

dependencies:
 - build

commands:
 - your-release-tool
```

If a summary is missing, the description will be printed. If the task does not
have a summary or a description, a warning is printed.

Showing a summary does not execute the task.
