---
title: Configuration Reference
description: Complete reference for the Task config files and env vars
permalink: /reference/config/
outline: deep
---

# Configuration Reference

Task has multiple ways of being configured. These methods are parsed, in
sequence, in the following order with the highest priority last:

- [Environment variables](./environment.md)
- _Configuration files_
- [Command-line flags](./cli.md)

In this document, we will look at the second of the three options, configuration
files.

## File Precedence

Task will automatically look for directories containing configuration files in
the following order with the highest priority first:

- Current directory (or the one specified by the `--taskfile`/`--entrypoint`
  flags).
- Each directory walking up the file tree from the current directory (or the one
  specified by the `--taskfile`/`--entrypoint` flags) until we reach the user's
  home directory or the root directory of that drive.
- The users `$HOME` directory.
- The `$XDG_CONFIG_HOME/task` directory.

Config files in the current directory, its parent folders or home directory
should be called `.taskrc.yml` or `.taskrc.yaml`. Config files in the
`$XDG_CONFIG_HOME/task` directory are named the same way, but should not contain
the `.` prefix.

All config files will be merged together into a unified config, starting with
the lowest priority file in `$XDG_CONFIG_HOME/task` with each subsequent file
overwriting the previous one if values are set.

For example, given the following files:

```yaml [$XDG_CONFIG_HOME/task/taskrc.yml]
# lowest priority global config
option_1: foo
option_2: foo
option_3: foo
```

```yaml [$HOME/.taskrc.yml]
option_1: bar
option_2: bar
```

```yaml [$HOME/path/to/project/.taskrc.yml]
# highest priority project config
option_1: baz
```

You would end up with the following configuration:

```yaml
option_1: baz # Taken from $HOME/path/to/project/.taskrc.yml
option_2: bar # Taken from $HOME/.taskrc.yml
option_3: foo # Taken from $XDG_CONFIG_HOME/task/.taskrc.yml
```

## Configuration Options

### `experiments`

The experiments section allows you to enable Task's experimental features. These
options are not enumerated here. Instead, please refer to our
[experiments documentation](../experiments/index.md) for more information.

```yaml
experiments:
  feature_name: 1
  another_feature: 2
```

### `verbose`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Enable verbose output for all tasks
- **CLI equivalent**: [`-v, --verbose`](./cli.md#-v---verbose)

```yaml
verbose: true
```

### `color`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable colored output. Colors are automatically enabled in CI environments (`CI=true`).
- **CLI equivalent**: [`-c, --color`](./cli.md#-c---color)

```yaml
color: false
```

### `disable-fuzzy`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Disable fuzzy matching for task names. When enabled, Task will not suggest similar task names when you mistype a task name.
- **CLI equivalent**: [`--disable-fuzzy`](./cli.md#--disable-fuzzy)

```yaml
disable-fuzzy: true
```

### `concurrency`

- **Type**: `integer`
- **Minimum**: `1`
- **Description**: Number of concurrent tasks to run
- **CLI equivalent**: [`-C, --concurrency`](./cli.md#-c---concurrency-number)

```yaml
concurrency: 4
```

### `failfast`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Stop executing dependencies as soon as one of them fail
- **CLI equivalent**: [`-F, --failfast`](./cli.md#f-failfast)

```yaml
failfast: true
```

## Example Configuration

Here's a complete example of a `.taskrc.yml` file with all available options:

```yaml
# Global settings
verbose: true
color: true
disable-fuzzy: false
concurrency: 2

# Enable experimental features
experiments:
  REMOTE_TASKFILES: 1
