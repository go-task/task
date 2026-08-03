---
title: Configuration Reference
description: Complete reference for the Task config files and env vars
permalink: /reference/config/
outline: deep
---

# Configuration Reference

Task has multiple ways of being configured. These methods are parsed, in
sequence, in the following order with the highest priority last:

- _Configuration files_
- [Environment variables](./environment.md)
- [Command-line flags](./cli.md)

In this document, we will look at the first of the three options, configuration
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
- **Environment variable**: [`TASK_VERBOSE`](./environment.md#task-verbose)

```yaml
verbose: true
```

### `silent`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Disables echoing of commands
- **CLI equivalent**: [`-s, --silent`](./cli.md#-s---silent)
- **Environment variable**: [`TASK_SILENT`](./environment.md#task-silent)

```yaml
silent: true
```

### `color`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable colored output. Colors are automatically enabled in CI
  environments (`CI=true`).
- **CLI equivalent**: [`-c, --color`](./cli.md#-c---color)
- **Environment variable**: [`TASK_COLOR`](./environment.md#task-color)

```yaml
color: false
```

### `disable-fuzzy`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Disable fuzzy matching for task names. When enabled, Task
  will not suggest similar task names when you mistype a task name.
- **CLI equivalent**: [`--disable-fuzzy`](./cli.md#--disable-fuzzy)
- **Environment variable**:
  [`TASK_DISABLE_FUZZY`](./environment.md#task-disable-fuzzy)

```yaml
disable-fuzzy: true
```

### `concurrency`

- **Type**: `integer`
- **Minimum**: `1`
- **Description**: Number of concurrent tasks to run
- **CLI equivalent**: [`-C, --concurrency`](./cli.md#-c---concurrency-number)
- **Environment variable**:
  [`TASK_CONCURRENCY`](./environment.md#task-concurrency)

```yaml
concurrency: 4
```

### `failfast`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Stop executing dependencies as soon as one of them fail
- **CLI equivalent**: [`-F, --failfast`](./cli.md#-f---failfast)
- **Environment variable**: [`TASK_FAILFAST`](./environment.md#task-failfast)

```yaml
failfast: true
```

### `interactive`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Prompt for missing required variables instead of failing.
  When enabled, Task will display an interactive prompt for any missing required
  variable. Requires a TTY. Task automatically detects non-TTY environments (CI
  pipelines, etc.) and skips prompts.
- **CLI equivalent**: [`--interactive`](./cli.md#--interactive)

```yaml
interactive: true
```

### `temp-dir`

- **Type**: `string`
- **Default**: `./.task`
- **Description**: Directory to store Task temporary files, such as checksums
  and temporary metadata. Relative paths are relative to the root Taskfile.
- **Environment variable**: [`TASK_TEMP_DIR`](./environment.md#task-temp-dir)

```yaml
temp-dir: .task
```

### `remote`

- **Type**: `object`
- **Description**: Remote configuration settings for handling
  [remote Taskfiles](../remote-taskfiles.md).

#### `remote.insecure`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Allow insecure connections when fetching remote Taskfiles
- **CLI equivalent**: `--insecure`
- **Environment variable**:
  [`TASK_REMOTE_INSECURE`](./environment.md#task_remote_insecure)

```yaml
remote:
  insecure: true
```

#### `remote.offline`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Work in offline mode, preventing remote Taskfile fetching
- **CLI equivalent**: `--offline`
- **Environment variable**:
  [`TASK_REMOTE_OFFLINE`](./environment.md#task_remote_offline)

```yaml
remote:
  offline: true
```

#### `remote.timeout`

- **Type**: `string`
- **Default**: 10s
- **Pattern**: `^[0-9]+(ns|us|µs|ms|s|m|h)$`
- **Description**: Timeout duration for remote operations (e.g., '30s', '5m')
- **CLI equivalent**: `--timeout`
- **Environment variable**:
  [`TASK_REMOTE_TIMEOUT`](./environment.md#task_remote_timeout)

```yaml
remote:
  timeout: '1m'
```

#### `remote.cache-expiry`

- **Type**: `string`
- **Default**: 0s (no cache)
- **Pattern**: `^[0-9]+(ns|us|µs|ms|s|m|h)$`
- **Description**: Cache expiry duration for remote Taskfiles (e.g., '1h',
  '24h')
- **CLI equivalent**: `--expiry`
- **Environment variable**:
  [`TASK_REMOTE_CACHE_EXPIRY`](./environment.md#task_remote_cache_expiry)

```yaml
remote:
  cache-expiry: '6h'
```

#### `remote.cache-dir`

- **Type**: `string`
- **Default**: `.task`
- **Description**: Directory where remote Taskfiles are cached. Can be an
  absolute path (e.g., `/var/cache/task`) or relative to the Taskfile directory.
- **CLI equivalent**: `--remote-cache-dir`
- **Environment variable**:
  [`TASK_REMOTE_CACHE_DIR`](./environment.md#task_remote_cache_dir)

```yaml
remote:
  cache-dir: ~/.task
```

#### `remote.trusted-hosts`

- **Type**: `array of strings`
- **Default**: `[]` (empty list)
- **Description**: List of trusted hosts for remote Taskfiles. Hosts in this
  list will not prompt for confirmation when downloading Taskfiles
- **CLI equivalent**: `--trusted-hosts`
- **Environment variable**:
  [`TASK_REMOTE_TRUSTED_HOSTS`](./environment.md#task_remote_trusted_hosts)
  (comma-separated)

```yaml
remote:
  trusted-hosts:
    - github.com
    - gitlab.com
    - raw.githubusercontent.com
    - example.com:8080
```

Hosts in the trusted hosts list will automatically be trusted without prompting
for confirmation when they are first downloaded or when their checksums change.
The host matching includes the port if specified in the URL. Use with caution
and only add hosts you fully trust.

You can also specify trusted hosts via the command line:

```shell
# Trust specific host for this execution
task --trusted-hosts github.com -t https://github.com/user/repo.git//Taskfile.yml

# Trust multiple hosts (comma-separated)
task --trusted-hosts github.com,gitlab.com -t https://github.com/user/repo.git//Taskfile.yml

# Trust a host with a specific port
task --trusted-hosts example.com:8080 -t https://example.com:8080/Taskfile.yml
```

#### `remote.cacert`

- **Type**: `string`
- **Default**: `""`
- **Description**: Path to a custom CA certificate file for TLS verification

```yaml
remote:
  cacert: '/path/to/ca.crt'
```

#### `remote.cert`

- **Type**: `string`
- **Default**: `""`
- **Description**: Path to a client certificate file for mTLS authentication

```yaml
remote:
  cert: '/path/to/client.crt'
```

#### `remote.cert-key`

- **Type**: `string`
- **Default**: `""`
- **Description**: Path to the client certificate private key file

```yaml
remote:
  cert-key: '/path/to/client.key'
```

## Example Configuration

Here's a complete example of a `.taskrc.yml` file with all available options:

```yaml
# Global settings
verbose: true
silent: false
color: true
disable-fuzzy: false
concurrency: 2
temp-dir: .task
remote:
  insecure: false
  offline: false
  timeout: '30s'
  cache-expiry: '24h'
  cache-dir: ~/.task
  trusted-hosts:
    - github.com
    - gitlab.com
  cacert: ''
  cert: ''
  cert-key: ''

# Enable experimental features
experiments:
  GENTLE_FORCE: 1
```
