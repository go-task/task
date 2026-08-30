---
title: Running tasks
description:
  How Task finds a Taskfile, and how to run one from a subdirectory, from your
  home directory, from standard input or as a dry run.
outline: deep
---

# Running tasks

Task looks for a Taskfile in the current directory, but it can run one from
almost anywhere else too.

Specific Taskfiles can be called by specifying the `--taskfile` flag. If you
don't specify a Taskfile, Task will automatically look for a file with one of
the [supported file names](#supported-file-names) in the current directory. If
you want to search in a different directory, you can use the `--dir` flag.

## Supported file names

Task looks for files with the following names, in order of priority:

- `Taskfile.yml`
- `taskfile.yml`
- `Taskfile.yaml`
- `taskfile.yaml`
- `Taskfile.dist.yml`
- `taskfile.dist.yml`
- `Taskfile.dist.yaml`
- `taskfile.dist.yaml`

The `.dist` variants allow projects to have one committed file (`.dist`) while
still allowing individual users to override the Taskfile by adding an additional
`Taskfile.yml` (which would be in your `.gitignore`).

## Running a Taskfile from a subdirectory

If a Taskfile cannot be found in the current working directory, it will walk up
the file tree until it finds one (similar to how `git` works). When running Task
from a subdirectory like this, it will behave as if you ran it from the
directory containing the Taskfile.

You can use this functionality along with the special
<span v-pre>`{{.USER_WORKING_DIR}}`</span> variable to create some very useful
reusable tasks. For example, if you have a monorepo with directories for each
microservice, you can `cd` into a microservice directory and run a task command
to bring it up without having to create multiple tasks or Taskfiles with
identical content. For example:

```yaml
version: '3'

tasks:
  up:
    dir: '{{.USER_WORKING_DIR}}'
    preconditions:
      - test -f docker-compose.yml
    cmds:
      - docker-compose up -d
```

In this example, we can run `cd <service>` and `task up` and as long as the
`<service>` directory contains a `docker-compose.yml`, the Docker composition
will be brought up.

## Running a global Taskfile

If you call Task with the `--global` (alias `-g`) flag, it will look for your
home directory instead of your working directory. In short, Task will look for a
Taskfile that matches `$HOME/{T,t}askfile.{yml,yaml}` .

This is useful to have automation that you can run from anywhere in your system!

::: info

When running your global Taskfile with `-g`, tasks will run on `$HOME` by
default, and not on your working directory!

As mentioned in the previous section, the
<span v-pre>`{{.USER_WORKING_DIR}}`</span> special variable can be very handy
here to run stuff on the directory you're calling `task -g` from.

```yaml
version: '3'

tasks:
  from-home:
    cmds:
      - pwd

  from-working-directory:
    dir: '{{.USER_WORKING_DIR}}'
    cmds:
      - pwd
```

:::

## Running a Taskfile from stdin

Taskfile also supports reading from stdin. This is useful if you are generating
Taskfiles dynamically and don't want write them to disk. To tell task to read
from stdin, you must specify the `-t/--taskfile` flag with the special `-`
value. You may then pipe into Task as you would any other program:

```shell
task -t - < ./Taskfile.yml
# OR
cat ./Taskfile.yml | task -t -
```

## Dry run mode

Dry run mode (`--dry`) compiles and steps through each task, printing the
commands that would be run without executing them. This is useful for debugging
your Taskfiles.

## Interactive CLI application

When running interactive CLI applications inside Task they can sometimes behave
weirdly, especially when the [output mode](./output.md#output-syntax) is set to
something other than `interleaved` (the default), or when interactive apps are
run in parallel with other tasks.

The `interactive: true` tells Task this is an interactive application and Task
will try to optimize for it:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - vim my-file.txt
    interactive: true
```

If you still have problems running an interactive app through Task, please open
an issue about it.
