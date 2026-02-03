---
title: Getting Started
description: Guide for getting started with Task
outline: deep
---

# Getting Started

The following guide will help introduce you to the basics of Task. We'll cover
how to create a Taskfile, how to write a basic task and how to call it. If you
haven't installed Task yet, head over to our [installation guide](installation).

## Creating your first Taskfile

Once Task is installed, you can create your first Taskfile by running:

```shell
task --init
```

This will create a file called `Taskfile.yml` in the current directory. If you
want to create the file in another directory, you can pass an absolute or
relative path to the directory into the command:

```shell
task --init ./subdirectory
```

Or if you want the Taskfile to have a specific name, you can pass in the name of
the file:

```shell
task --init Custom.yml
```

This will create a Taskfile that looks something like this:

```yaml [Taskfile.yml]
version: '3'

vars:
  GREETING: Hello, World!

tasks:
  default:
    desc: Print a greeting message
    cmds:
      - echo "{{.GREETING}}"
    silent: true
```

As you can see, all Taskfiles are written in [YAML format](https://yaml.org/).
The `version` attribute specifies the minimum version of Task that can be used
to run this file. The `vars` attribute is used to define variables that can be
used in tasks. In this case, we are creating a string variable called `GREETING`
with a value of `Hello, World!`.

Finally, the `tasks` attribute is used to define the tasks that can be run. In
this case, we have a task called `default` that echoes the value of the
`GREETING` variable. The `silent` attribute is set to `true`, which means that
the task metadata will not be printed when the task is run - only the output of
the commands.

## Calling a task

To call the task, invoke `task` followed by the name of the task you want to
run. In this case, the name of the task is `default`, so you should run:

```shell
task default
```

Note that we don't have to specify the name of the Taskfile. Task will
automatically look for a file called `Taskfile.yml` (or any of Task's
[supported file names](/docs/guide#supported-file-names)) in the current
directory. Additionally, tasks with the name `default` are special. They can
also be run without specifying the task name.

If you created a Taskfile in a different directory, you can run it by passing
the absolute or relative path to the directory as an argument using the `--dir`
flag:

```shell
task --dir ./subdirectory
```

Or if you created a Taskfile with a different name, you can run it by passing
the name of the Taskfile as an argument using the `--taskfile` flag:

```shell
task --taskfile Custom.yml
```

## Adding a build task

Let's create a task to build a program in Go. Start by adding a new task called
`build` below the existing `default` task. We can then add a `cmds` attribute
with a single command to build the program.

Task uses [mvdan/sh](https://github.com/mvdan/sh), a native Go sh interpreter.
So you can write sh/bash-like commands - even in environments where `sh` or
`bash` are usually not available (like Windows). Just remember any executables
called must be available as a built-in or in the system's `PATH`.

When you're done, it should look something like this:

```yaml
version: '3'

vars:
  GREETING: Hello, World!

tasks:
  default:
    desc: Print a greeting message
    cmds:
      - echo "{{.GREETING}}"
    silent: true

  build:
    cmds:
      - go build ./cmd/main.go
```

Call the task by running:

```shell
task build
```

That's about it for the basics, but there's _so much_ more that you can do with
Task. Check out the rest of the documentation to learn more about all the
features Task has to offer! We recommend taking a look at the
[usage guide](/docs/guide) next. Alternatively, you can check out our reference
docs for the [Taskfile schema](reference/schema) and [CLI](reference/cli).
