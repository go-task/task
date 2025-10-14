---
title: Environment Reference
description: A reference for the Taskfile environment variables
outline: deep
---

# Environment Reference

Task has multiple ways of being configured. These methods are parsed, in
sequence, in the following order with the highest priority last:

- _Environment variables_
- [Configuration files](./config.md)
- [Command-line flags](./cli.md)

In this document, we will look at the first of the three options, environment
variables. All Task-specific variables are prefixed with `TASK_` and override
their configuration file equivalents.

## Variables

### `TASK_TEMP_DIR`

Defines the location of Task's temporary directory which is used for storing
checksums and temporary metadata. Can be relative like `tmp/task` or absolute
like `/tmp/.task` or `~/.task`. Relative paths are relative to the root
Taskfile, not the working directory. Defaults to: `./.task`.

### `TASK_CORE_UTILS`

This env controls whether the Bash interpreter will use its own
core utilities implemented in Go, or the ones available in the system.
Valid values are `true` (`1`) or `false` (`0`). By default, this is `true` on
Windows and `false` on other operating systems. We might consider making this
enabled by default on all platforms in the future.

### `TASK_REMOTE_DIR`

Defines the location of Task's remote temporary directory which is used for
caching remote files. Can be relative like `tmp/task` or absolute like
`/tmp/.task` or `~/.task`. Relative paths are relative to the root Taskfile, not
the working directory. Defaults to: `./.task`.

### `TASK_OFFLINE`

Set the `--offline` flag through the environment variable. Only for remote
experiment. CLI flag `--offline` takes precedence over the env variable.

### `FORCE_COLOR`

Force color output usage.

### Custom Colors

All color variables are [ANSI color codes][ansi]. You can specify multiple codes
separated by a semicolon. For example: `31;1` will make the text bold and red.
Task also supports 8-bit color (256 colors). You can specify these colors by
using the sequence `38;2;R:G:B` for foreground colors and `48;2;R:G:B` for
background colors where `R`, `G` and `B` should be replaced with values between
0 and 255.

For convenience, we allow foreground colors to be specified using shorthand,
comma-separated syntax: `R,G,B`. For example, `255,0,0` is equivalent to
`38;2;255:0:0`.

A table of variables and their defaults can be found below:

| ENV                         | Default |
| --------------------------- | ------- |
| `TASK_COLOR_RESET`          | `0`     |
| `TASK_COLOR_RED`            | `31`    |
| `TASK_COLOR_GREEN`          | `32`    |
| `TASK_COLOR_YELLOW`         | `33`    |
| `TASK_COLOR_BLUE`           | `34`    |
| `TASK_COLOR_MAGENTA`        | `35`    |
| `TASK_COLOR_CYAN`           | `36`    |
| `TASK_COLOR_BRIGHT_RED`     | `91`    |
| `TASK_COLOR_BRIGHT_GREEN`   | `92`    |
| `TASK_COLOR_BRIGHT_YELLOW`  | `93`    |
| `TASK_COLOR_BRIGHT_BLUE`    | `94`    |
| `TASK_COLOR_BRIGHT_MAGENTA` | `95`    |
| `TASK_COLOR_BRIGHT_CYAN`    | `96`    |

[ansi]: https://en.wikipedia.org/wiki/ANSI_escape_code
