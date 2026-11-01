---
title: Platforms and shells
description:
  Restrict tasks and commands to an operating system or architecture, and set
  shell options with `set` and `shopt`.
section: Guide
docType: guide
outline: deep
---

# Platforms and shells {#platform-specific-behaviour}

Task runs on Linux, macOS, and Windows. A portable Taskfile also needs commands
and tools that work on the machines where you run it. Use `platforms` to select
OS-specific steps, and keep shared work in the same task.

## Select platform commands {#platform-specific-tasks-and-commands}

Set `platforms` on an individual command when only part of a task differs by
platform:

```yaml
version: '3'

tasks:
  setup:
    cmds:
      - cmd: echo 'Preparing Windows tools'
        platforms: [windows]
      - cmd: echo 'Preparing Unix tools'
        platforms: [linux, darwin]
      - echo 'Preparing shared files'
```

Run `task setup`. A command whose platform does not match is skipped without an
error; the shared command runs on every platform.

### Restrict a whole task

Put `platforms` on the task when none of its commands should run elsewhere:

```yaml
version: '3'

tasks:
  setup:windows:
    platforms: [windows/amd64]
    cmds:
      - echo 'Preparing tools for Windows on amd64'
```

| Restriction               | Matches                                        |
| ------------------------- | ---------------------------------------------- |
| `[windows]`               | Windows on any architecture                    |
| `[amd64]`                 | Any supported OS on amd64                      |
| `[windows/amd64]`         | Windows on amd64                               |
| `[windows/amd64, darwin]` | Windows on amd64, or macOS on any architecture |

OS and architecture names use Go's `GOOS` and `GOARCH` values. These
restrictions check the machine running Task; they do not configure a compiler's
target.

### Share platform values

The `OS` and `ARCH` template functions expose the current platform. Use them to
select [OS-specific Taskfiles](./includes.md#os-specific-taskfiles) or construct
arguments. Use `exeExt` for executable names that need `.exe` on Windows:

```yaml
version: '3'

tasks:
  build:
    cmds:
      - go build -o app{{exeExt}} main.go
```

## Understand command shells

Task runs commands through its embedded shell interpreter. It does not start
your interactive shell or load that shell's startup configuration. Use
[Taskfile `env` and `dotenv`](./environment.md) for values commands need.

Each `cmds` entry gets a separate shell context. A `cd` or `export` in one entry
does not carry over to the next. Set `dir` and `env` on the task instead:

```yaml
version: '3'

tasks:
  inspect:
    dir: src
    env:
      BUILD_MODE: development
    cmds:
      - pwd
      - echo "$BUILD_MODE"
```

If several commands need to share shell state, put them in one multiline `cmd`
or a script. Programs such as `go`, `npm`, or `docker` must still be installed.
Invoke `bash`, `pwsh`, or `cmd /c` explicitly when a command needs that
particular shell, and restrict it to machines where it is available.

## Configure shell options {#set-and-shopt}

Use `set` for POSIX shell options and `shopt` for supported Bash-style options.
For example, `pipefail` makes a pipeline fail when an earlier command fails, and
`globstar` enables recursive `**` globs in shell commands:

```yaml
version: '3'

set: [pipefail]
shopt: [globstar]

tasks:
  list:
    cmds:
      - echo **/*.go
```

These options can be set at the root, task, or command level. The shell
interpreter supports a subset of shell options; see the
[shell options reference](../reference/schema.md#shell-options).

`shopt` controls shell expansion inside commands. Globs in `sources` and
`generates` use [Task's file matching](./up-to-date.md#track-the-right-files)
and do not require `globstar`.
