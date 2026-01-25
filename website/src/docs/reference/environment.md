---
title: Environment Reference
description: A reference for the Taskfile environment variables
outline: deep
---

# Environment Reference

Task has multiple ways of being configured. These methods are parsed, in
sequence, in the following order with the highest priority last:

- [Configuration files](./config.md)
- _Environment variables_
- [Command-line flags](./cli.md)

In this document, we will look at the second of the three options, environment
variables. All Task-specific variables are prefixed with `TASK_` and override
their configuration file equivalents.

## Variables

All [configuration file options](./config.md) can also be set via environment
variables. The priority order is: CLI flags > environment variables > config files > defaults.

### `TASK_VERBOSE`

- **Type**: `boolean` (`true`, `false`, `1`, `0`)
- **Default**: `false`
- **Description**: Enable verbose output for all tasks
- **Config equivalent**: [`verbose`](./config.md#verbose)

### `TASK_SILENT`

- **Type**: `boolean` (`true`, `false`, `1`, `0`)
- **Default**: `false`
- **Description**: Disables echoing of commands
- **Config equivalent**: [`silent`](./config.md#silent)

### `TASK_COLOR`

- **Type**: `boolean` (`true`, `false`, `1`, `0`)
- **Default**: `true`
- **Description**: Enable colored output
- **Config equivalent**: [`color`](./config.md#color)

### `TASK_DISABLE_FUZZY`

- **Type**: `boolean` (`true`, `false`, `1`, `0`)
- **Default**: `false`
- **Description**: Disable fuzzy matching for task names
- **Config equivalent**: [`disable-fuzzy`](./config.md#disable-fuzzy)

### `TASK_CONCURRENCY`

- **Type**: `integer`
- **Description**: Limit number of tasks to run concurrently
- **Config equivalent**: [`concurrency`](./config.md#concurrency)

### `TASK_FAILFAST`

- **Type**: `boolean` (`true`, `false`, `1`, `0`)
- **Default**: `false`
- **Description**: When running tasks in parallel, stop all tasks if one fails
- **Config equivalent**: [`failfast`](./config.md#failfast)

### `TASK_DRY`

- **Type**: `boolean` (`true`, `false`, `1`, `0`)
- **Default**: `false`
- **Description**: Compiles and prints tasks in the order that they would be run, without executing them

### `TASK_ASSUME_YES`

- **Type**: `boolean` (`true`, `false`, `1`, `0`)
- **Default**: `false`
- **Description**: Assume "yes" as answer to all prompts

### `TASK_INTERACTIVE`

- **Type**: `boolean` (`true`, `false`, `1`, `0`)
- **Default**: `false`
- **Description**: Prompt for missing required variables

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
