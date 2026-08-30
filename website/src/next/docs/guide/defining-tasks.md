---
title: Defining tasks
description:
  Task syntax shortcuts, internal tasks, aliases, the directory a task runs in,
  and the help text Task shows for it.
section: Guide
docType: guide
outline: deep
---

# Defining tasks

Beyond a name and a list of commands, a task carries a handful of properties
that control how it is written, named and presented.

## Short task syntax

Starting on Task v3, you can now write tasks with a shorter syntax if they have
the default settings (e.g. no custom `env:`, `vars:`, `desc:`, `silent:` , etc):

```yaml
version: '3'

tasks:
  build: go build -v -o ./app{{exeExt}} .

  run:
    - task: build
    - ./app{{exeExt}} -h localhost -p 8080
```

## Internal tasks

Internal tasks are tasks that cannot be called directly by the user. They will
not appear in the output when running `task --list|--list-all`. Other tasks may
call internal tasks in the usual way. This is useful for creating reusable,
function-like tasks that have no useful purpose on the command line.

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

## Task directory

By default, tasks will be executed in the directory where the Taskfile is
located. But you can easily make the task run in another folder, informing
`dir`:

```yaml
version: '3'

tasks:
  serve:
    dir: public/www
    cmds:
      # run http server
      - caddy
```

If the directory does not exist, `task` creates it.

## Task aliases

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

## Overriding task name

Sometimes you may want to override the task name printed on the summary,
up-to-date messages to STDOUT, etc. In this case, you can just set `label:`,
which can also be interpolated with variables:

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

## Help

Running `task --list` (or `task -l`) lists all tasks with a description. The
following Taskfile:

```yaml
version: '3'

tasks:
  build:
    desc: Build the go binary.
    cmds:
      - go build -v -i main.go

  test:
    desc: Run all the go tests.
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
* build:   Build the go binary.
* test:    Run all the go tests.
```

If you want to see all tasks, there's a `--list-all` (alias `-a`) flag as well.

## Display summary of task

Running `task --summary task-name` will show a summary of a task. The following
Taskfile:

```yaml
version: '3'

tasks:
  release:
    deps: [build]
    summary: |
      Release your project to github

      It will build your project before starting the release.
      Please make sure that you have set GITHUB_TOKEN before starting.
    cmds:
      - your-release-tool

  build:
    cmds:
      - your-build-tool
```

with running `task --summary release` would print the following output:

```
task: release

Release your project to github

It will build your project before starting the release.
Please make sure that you have set GITHUB_TOKEN before starting.

dependencies:
 - build

commands:
 - your-release-tool
```

If a summary is missing, the description will be printed. If the task does not
have a summary or a description, a warning is printed.

Please note: _showing the summary will not execute the command_.
