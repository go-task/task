---
title: Command Line Interface Reference
description: Complete reference for Task CLI commands, flags, and exit codes
permalink: /reference/cli/
outline: deep
---

# Command Line Interface Reference

Task has multiple ways of being configured. These methods are parsed, in
sequence, in the following order with the highest priority last:

- [Environment variables](./environment.md)
- [Configuration files](./config.md)
- _Command-line flags_

In this document, we will look at the last of the three options, command-line
flags. All CLI commands override their configuration file and environment
variable equivalents.

## Format

Task commands have the following syntax:

```bash
task [options] [tasks...] [-- CLI_ARGS...]
```

::: tip

If `--` is given, all remaining arguments will be assigned to a special
`CLI_ARGS` variable.

:::

## Commands

### `task [tasks...]`

Run one or more tasks defined in your Taskfile.

```bash
task build
task test lint
task deploy --force
```

### `task --list`

List all available tasks with their descriptions.

```bash
task --list
task -l
```

### `task --list-all`

List all tasks, including those without descriptions.

```bash
task --list-all
task -a
```

### `task --init`

Create a new Taskfile.yml in the current directory.

```bash
task --init
task -i
```

## Options

### General

#### `-h, --help`

Show help information.

```bash
task --help
```

#### `--version`

Show Task version.

```bash
task --version
```

#### `-v, --verbose`

Enable verbose mode for detailed output.

```bash
task build --verbose
```

#### `-s, --silent`

Disable command echoing.

```bash
task deploy --silent
```

#### `--disable-fuzzy`

Disable fuzzy matching for task names. When enabled, Task will not suggest similar task names when you mistype a task name.

```bash
task buidl --disable-fuzzy
# Output: Task "buidl" does not exist
# (without "Did you mean 'build'?" suggestion)
```

### Execution Control

#### `-F, --failfast`

Stop executing dependencies as soon as one of them fails.

```bash
task build --failfast
```

#### `-f, --force`

Force execution even when the task is up-to-date.

```bash
task build --force
```

#### `-n, --dry`

Compile and print tasks without executing them.

```bash
task deploy --dry
```

#### `-p, --parallel`

Execute multiple tasks in parallel.

```bash
task test lint --parallel
```

#### `-C, --concurrency <number>`

Limit the number of concurrent tasks. Zero means unlimited.

```bash
task test --concurrency 4
```

#### `-x, --exit-code`

Pass through the exit code of failed commands.

```bash
task test --exit-code
```

### File and Directory

#### `-d, --dir <path>`

Set the directory where Task will run and look for Taskfiles.

```bash
task build --dir ./backend
```

#### `-t, --taskfile <file>`

Specify a custom Taskfile path.

```bash
task build --taskfile ./custom/Taskfile.yml
```

#### `-g, --global`

Run the global Taskfile from `$HOME/Taskfile.{yml,yaml}`.

```bash
task backup --global
```

### Output Control

#### `-o, --output <mode>`

Set output style. Available modes: `interleaved`, `group`, `prefixed`.

```bash
task test --output group
```

#### `--output-group-begin <template>`

Message template to print before grouped output.

```bash
task test --output group --output-group-begin "::group::{{.TASK}}"
```

#### `--output-group-end <template>`

Message template to print after grouped output.

```bash
task test --output group --output-group-end "::endgroup::"
```

#### `--output-group-error-only`

Only show command output on non-zero exit codes.

```bash
task test --output group --output-group-error-only
```

#### `-c, --color`

Control colored output. Enabled by default.

```bash
task build --color=false
# or use environment variable
NO_COLOR=1 task build
```

### Task Information

#### `--status`

Check if tasks are up-to-date without running them.

```bash
task build --status
```

#### `--summary`

Show detailed information about a task.

```bash
task build --summary
```

#### `--json`

Output task information in JSON format (use with `--list` or `--list-all`).

```bash
task --list --json
```

#### `--sort <mode>`

Change task listing order. Available modes: `default`, `alphanumeric`, `none`.

```bash
task --list --sort alphanumeric
```

### Watch Mode

#### `-w, --watch`

Watch for file changes and re-run tasks automatically.

```bash
task build --watch
```

#### `-I, --interval <duration>`

Set watch interval (default: `5s`). Must be a valid
[Go duration](https://pkg.go.dev/time#ParseDuration).

```bash
task build --watch --interval 1s
```

### Interactive

#### `-y, --yes`

Automatically answer "yes" to all prompts.

```bash
task deploy --yes
```

## Exit Codes

Task uses specific exit codes to indicate different types of errors:

### Success

- **0** - Success

### General Errors (1-99)

- **1** - Unknown error occurred

### Taskfile Errors (100-199)

- **100** - No Taskfile found
- **101** - Taskfile already exists (when using `--init`)
- **102** - Invalid or unparseable Taskfile
- **103** - Remote Taskfile download failed
- **104** - Remote Taskfile not trusted
- **105** - Remote Taskfile fetch not secure
- **106** - No cache for remote Taskfile in offline mode
- **107** - No schema version defined in Taskfile

### Task Errors (200-255)

- **200** - Task not found
- **201** - Command execution error
- **202** - Attempted to run internal task
- **203** - Multiple tasks with same name/alias
- **204** - Task called too many times (recursion limit)
- **205** - Task cancelled by user
- **206** - Missing required variables
- **207** - Variable has incorrect value

::: info

When using `-x/--exit-code`, failed command exit codes are passed through
instead of the above codes.

:::

::: tip

The complete list of exit codes is available in the repository at
[`errors/errors.go`](https://github.com/go-task/task/blob/main/errors/errors.go).

:::

## JSON Output Format

When using `--json` with `--list` or `--list-all`:

```json
{
  "tasks": [
    {
      "name": "build",
      "task": "build",
      "desc": "Build the application",
      "summary": "Compiles the source code and generates binaries",
      "up_to_date": false,
      "location": {
        "line": 12,
        "column": 3,
        "taskfile": "/path/to/Taskfile.yml"
      }
    }
  ],
  "location": "/path/to/Taskfile.yml"
}
```
