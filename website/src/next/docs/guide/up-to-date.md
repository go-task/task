---
title: Up-to-date checks
description:
  Skip unchanged builds, check whether work is already done, and choose between
  source fingerprints and status commands.
section: Guide
docType: guide
outline: deep
---

# Up-to-date checks {#skipping-work-that-is-up-to-date}

A build should run when its inputs change or its output is missing. Declare
those files with `sources` and `generates`, and Task can skip the commands on
your next run. For other kinds of work, use `status` to check whether the result
is already available.

## Skip unchanged builds {#by-fingerprinting-locally-generated-files-and-their-sources}

For a Go application in `main.go`, declare the source file and the executable
that the build produces:

```yaml
version: '3'

tasks:
  build:
    sources: [main.go]
    generates: ['app{{exeExt}}']
    cmds:
      - go build -o app{{exeExt}} main.go
```

Run `task build` to create the executable. Run it again without changing
`main.go`, and Task reports that `build` is up to date. Edit `main.go` or delete
the executable, and the next call builds it again.

The default method, `checksum`, compares the content of source files with a
saved fingerprint. It checks that declared outputs exist; it does not compare
their contents. Editing the executable alone will not trigger a rebuild.

These checks skip the task's commands. Its [dependencies](./dependencies.md)
still run first and perform their own up-to-date checks.

### Track the right files

`sources` and `generates` accept files and glob patterns, relative to the task's
[working directory](./defining-tasks.md#task-directory). Include every file that
affects the result, such as source code, configuration, and dependency
lockfiles.

For a Go module with several source files, replace `sources` and `cmds` in the
example above with:

```yaml
sources:
  - '**/*.go'
  - go.mod
  - go.sum
  - exclude: '**/*_test.go'
cmds:
  - go build -o app{{exeExt}} .
```

Patterns are evaluated in order. Put each `exclude` after the positive glob it
narrows. Here, test files are excluded because this task builds the application;
a test task should track them.

Set `use_gitignore: true` at the root of the Taskfile to filter glob matches
using `.gitignore` rules. A task can override that setting. Keep it disabled for
tasks that need to track ignored files as inputs.

`generates` is optional. A task such as a test suite can use `sources` alone to
skip work when its inputs have not changed. `generates` alone does not provide a
file fingerprint; use `status` if you only need to check that a result exists.

### Choose a check method

| Method               | How it checks sources                                                |
| -------------------- | -------------------------------------------------------------------- |
| `checksum` (default) | Compares file contents with a saved fingerprint.                     |
| `timestamp`          | Compares modification times with outputs and Task's saved timestamp. |
| `none`               | Disables file-based skipping.                                        |

To use modification times for the build above, add `method: timestamp` to the
task:

```yaml
tasks:
  build:
    method: timestamp
    sources: [main.go]
    generates: ['app{{exeExt}}']
    cmds:
      - go build -o app{{exeExt}} main.go
```

You can also set `method` at the root of the Taskfile to change the default for
all tasks. A task's own setting takes precedence.

With `timestamp`, touching a source can cause a rerun even if its content has
not changed. Without `generates`, Task uses its saved timestamp for the
comparison. Use `method: none` when `sources` should drive
[watch mode](./watch.md) without making the task up to date.

## Check existing work {#using-programmatic-checks-to-indicate-a-task-is-up-to-date}

Use `status` when a command can determine whether work is needed. Every check
must exit with code `0` for Task to skip the commands. If any check fails, Task
runs the commands:

```yaml
version: '3'

tasks:
  prepare:
    status:
      - test -d directory
      - test -f directory/file1.txt
      - test -f directory/file2.txt
    cmds:
      - mkdir -p directory
      - touch directory/file1.txt directory/file2.txt
```

Run `task prepare` twice: the first call creates the files, and the second skips
the commands. Delete either file, and Task runs the commands again.

Keep status checks fast and read-only: they may run each time the task is
called. They can inspect files, installed tools, services, or remote artifacts.
A failed check means "do the work", so the commands should handle partially
completed work, as `mkdir -p` does above.

If a failed check should stop execution with an error, use
[preconditions](./conditional-execution.md#using-programmatic-checks-to-cancel-the-execution-of-a-task-and-its-dependencies).

### Combine files and status

Use both `sources` and `status` when the result depends on local files but
cannot be checked with `generates`. This Docker example tags the image with a
checksum of its build inputs:

```yaml
version: '3'

tasks:
  image:
    sources:
      - Dockerfile
      - src/**/*
    status:
      - docker image inspect example/app:{{.CHECKSUM}} > /dev/null 2>&1
    cmds:
      - docker build -t example/app:{{.CHECKSUM}} .
```

Task skips the build only when the sources are unchanged **and** the status
check succeeds. A changed source produces a new tag; a missing image makes the
check fail. Include any other files used by your Dockerfile in `sources`.

You can also combine `sources`, `generates`, and `status`: every configured
check must pass for Task to consider the work up to date.

### Use source values

With `method: checksum`, <span v-pre>`{{.CHECKSUM}}`</span> contains the source
fingerprint. With `method: timestamp`, <span v-pre>`{{.TIMESTAMP}}`</span>
contains the latest source modification time. Both can be used in `cmds` and
`status`; they describe the sources, not the generated files.

`.TIMESTAMP` is a Go `time.Time` value. For example,
<span v-pre>`{{.TIMESTAMP.Unix}}`</span> produces a Unix timestamp, and
<span v-pre>`{{.TIMESTAMP.Format "2006-01-02"}}`</span> formats a date.

## Check or force a run

To check whether tasks are up to date without running their commands:

```shell
task --status build
```

You can pass multiple task names. The command exits with a non-zero
[exit code](../reference/cli.md#exit-codes) if any task is not up to date.

To run a task even when its checks say the work is already done:

```shell
task --force build
```

`-f` is the short form of `--force`. To rerun automatically when files change,
use [Watch mode](./watch.md).

## Track different inputs

With the checksum method, Task stores one source fingerprint per task label. If
build variables affect the result, include them in the label and use separate
output paths. For example, to build with different Go build tags:

```yaml
version: '3'

tasks:
  build:
    label: 'build-{{.MODE}}'
    sources: [main.go]
    generates: ['app-{{.MODE}}{{exeExt}}']
    cmds:
      - go build -tags '{{.MODE}}' -o app-{{.MODE}}{{exeExt}} main.go
```

Run `task build MODE=debug`, then `task build MODE=release`. Each mode has its
own executable and saved checksum, so a later `task build MODE=debug` can skip
the build. A change to `main.go` makes both modes need a rebuild when next
called.

These saved checks apply across invocations. To limit repeated task calls within
a single invocation, use
[`run: once` or `run: when_changed`](./dependencies.md#repeated-calls).

## Store fingerprints

Task stores its file fingerprints in `.task/` under the project directory.
Usually, add this directory to `.gitignore`. For generated code committed to
version control, you may choose to commit the corresponding checksum too.

Set `TASK_TEMP_DIR` to store fingerprints elsewhere:

```shell
export TASK_TEMP_DIR='~/.task'
```

Relative paths, such as `tmp/task`, are resolved from the project directory.
Absolute paths and paths starting with `~` use a separate subdirectory for each
project.
