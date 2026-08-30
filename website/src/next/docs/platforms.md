---
title: Platform-specific behaviour
description:
  Restrict tasks and commands to an operating system or architecture, and set
  shell options with `set` and `shopt`.
outline: deep
---

# Platform-specific behaviour

The same Taskfile often has to behave differently depending on where it runs.

## Platform specific tasks and commands

If you want to restrict the running of tasks to explicit platforms, this can be
achieved using the `platforms:` key. Tasks can be restricted to a specific OS,
architecture or a combination of both. On a mismatch, the task or command will
be skipped, and no error will be thrown.

The values allowed as OS or Arch are valid `GOOS` and `GOARCH` values, as
defined by the Go language
[here](https://github.com/golang/go/blob/master/src/internal/syslist/syslist.go).

The `build-windows` task below will run only on Windows, and on any
architecture:

```yaml
version: '3'

tasks:
  build-windows:
    platforms: [windows]
    cmds:
      - echo 'Running command on Windows'
```

This can be restricted to a specific architecture as follows:

```yaml
version: '3'

tasks:
  build-windows-amd64:
    platforms: [windows/amd64]
    cmds:
      - echo 'Running command on Windows (amd64)'
```

It is also possible to restrict the task to specific architectures:

```yaml
version: '3'

tasks:
  build-amd64:
    platforms: [amd64]
    cmds:
      - echo 'Running command on amd64'
```

Multiple platforms can be specified as follows:

```yaml
version: '3'

tasks:
  build:
    platforms: [windows/amd64, darwin]
    cmds:
      - echo 'Running command on Windows (amd64) and macOS'
```

Individual commands can also be restricted to specific platforms:

```yaml
version: '3'

tasks:
  build:
    cmds:
      - cmd: echo 'Running command on Windows (amd64) and macOS'
        platforms: [windows/amd64, darwin]
      - cmd: echo 'Running on all platforms'
```

## `set` and `shopt`

It's possible to specify options to the
[`set`](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html)
and
[`shopt`](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html)
builtins. This can be added at global, task or command level.

```yaml
version: '3'

set: [pipefail]
shopt: [globstar]

tasks:
  # `globstar` required for double star globs to work
  default: echo **/*.go
```

::: info

Keep in mind that not all options are available in the
[shell interpreter library](https://github.com/mvdan/sh) that Task uses.

:::
