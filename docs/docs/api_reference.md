---
slug: /api/
sidebar_position: 4
toc_min_heading_level: 2
toc_max_heading_level: 5
---

# API Reference

## CLI

Task command line tool has the following syntax:

```bash
task [--flags] [tasks...] [-- CLI_ARGS...]
```

:::tip

If `--` is given, all remaining arguments will be assigned to a special
`CLI_ARGS` variable

:::

| Short | Flag                        | Type     | Default                                      | Description                                                                                                                                                                                  |
| ----- | --------------------------- | -------- | -------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `-c`  | `--color`                   | `bool`   | `true`                                       | Colored output. Enabled by default. Set flag to `false` or use `NO_COLOR=1` to disable.                                                                                                      |
| `-C`  | `--concurrency`             | `int`    | `0`                                          | Limit number tasks to run concurrently. Zero means unlimited.                                                                                                                                |
| `-d`  | `--dir`                     | `string` | Working directory                            | Sets directory of execution.                                                                                                                                                                 |
| `-n`  | `--dry`                     | `bool`   | `false`                                      | Compiles and prints tasks in the order that they would be run, without executing them.                                                                                                       |
| `-x`  | `--exit-code`               | `bool`   | `false`                                      | Pass-through the exit code of the task command.                                                                                                                                              |
| `-f`  | `--force`                   | `bool`   | `false`                                      | Forces execution even when the task is up-to-date.                                                                                                                                           |
| `-g`  | `--global`                  | `bool`   | `false`                                      | Runs global Taskfile, from `$HOME/Taskfile.{yml,yaml}`.                                                                                                                                      |
| `-h`  | `--help`                    | `bool`   | `false`                                      | Shows Task usage.                                                                                                                                                                            |
| `-i`  | `--init`                    | `bool`   | `false`                                      | Creates a new Taskfile.yml in the current folder.                                                                                                                                            |
| `-I`  | `--interval`                | `string` | `5s`                                         | Sets a different watch interval when using `--watch`, the default being 5 seconds. This string should be a valid [Go Duration](https://pkg.go.dev/time#ParseDuration).                       |
| `-l`  | `--list`                    | `bool`   | `false`                                      | Lists tasks with description of current Taskfile.                                                                                                                                            |
| `-a`  | `--list-all`                | `bool`   | `false`                                      | Lists tasks with or without a description.                                                                                                                                                   |
|       | `--sort`                    | `string` | `default`                                    | Changes the order of the tasks when listed.<br />`default` - Alphanumeric with root tasks first<br />`alphanumeric` - Alphanumeric<br />`none` - No sorting (As they appear in the Taskfile) |
|       | `--json`                    | `bool`   | `false`                                      | See [JSON Output](#json-output)                                                                                                                                                              |
| `-o`  | `--output`                  | `string` | Default set in the Taskfile or `intervealed` | Sets output style: [`interleaved`/`group`/`prefixed`].                                                                                                                                       |
|       | `--output-group-begin`      | `string` |                                              | Message template to print before a task's grouped output.                                                                                                                                    |
|       | `--output-group-end`        | `string` |                                              | Message template to print after a task's grouped output.                                                                                                                                     |
|       | `--output-group-error-only` | `bool`   | `false`                                      | Swallow command output on zero exit code.                                                                                                                                                    |
| `-p`  | `--parallel`                | `bool`   | `false`                                      | Executes tasks provided on command line in parallel.                                                                                                                                         |
| `-s`  | `--silent`                  | `bool`   | `false`                                      | Disables echoing.                                                                                                                                                                            |
| `-y`  | `--yes`                     | `bool`   | `false`                                      | Assume "yes" as answer to all prompts.                                                                                                                                                       |
|       | `--status`                  | `bool`   | `false`                                      | Exits with non-zero exit code if any of the given tasks is not up-to-date.                                                                                                                   |
|       | `--summary`                 | `bool`   | `false`                                      | Show summary about a task.                                                                                                                                                                   |
| `-t`  | `--taskfile`                | `string` | `Taskfile.yml` or `Taskfile.yaml`            |                                                                                                                                                                                              |
| `-v`  | `--verbose`                 | `bool`   | `false`                                      | Enables verbose mode.                                                                                                                                                                        |
|       | `--version`                 | `bool`   | `false`                                      | Show Task version.                                                                                                                                                                           |
| `-w`  | `--watch`                   | `bool`   | `false`                                      | Enables watch of the given task.                                                                                                                                                             |

## Exit Codes

Task will sometimes exit with specific exit codes. These codes are split into
three groups with the following ranges:

- General errors (0-99)
- Taskfile errors (100-199)
- Task errors (200-299)

A full list of the exit codes and their descriptions can be found below:

| Code | Description                                                  |
| ---- | ------------------------------------------------------------ |
| 0    | Success                                                      |
| 1    | An unknown error occurred                                    |
| 100  | No Taskfile was found                                        |
| 101  | A Taskfile already exists when trying to initialize one      |
| 102  | The Taskfile is invalid or cannot be parsed                  |
| 103  | A remote Taskfile could not be downlaoded                    |
| 104  | A remote Taskfile was not trusted by the user                |
| 105  | A remote Taskfile was could not be fetched securely          |
| 106  | No cache was found for a remote Taskfile in offline mode     |
| 107  | No schema version was defined in the Taskfile                |
| 200  | The specified task could not be found                        |
| 201  | An error occurred while executing a command inside of a task |
| 202  | The user tried to invoke a task that is internal             |
| 203  | There a multiple tasks with the same name or alias           |
| 204  | A task was called too many times                             |
| 205  | A task was cancelled by the user                             |
| 206  | A task was not executed due to missing required variables    |

These codes can also be found in the repository in
[`errors/errors.go`](https://github.com/go-task/task/blob/main/errors/errors.go).

:::info

When Task is run with the `-x`/`--exit-code` flag, the exit code of any failed
commands will be passed through to the user instead.

:::

## JSON Output

When using the `--json` flag in combination with either the `--list` or
`--list-all` flags, the output will be a JSON object with the following
structure:

```json
{
  "tasks": [
    {
      "name": "",
      "desc": "",
      "summary": "",
      "up_to_date": false,
      "location": {
        "line": 54,
        "column": 3,
        "taskfile": "/path/to/Taskfile.yml"
      }
    }
    // ...
  ],
  "location": "/path/to/Taskfile.yml"
}
```

## Special Variables

There are some special variables that is available on the templating system:

| Var                | Description                                                                                                                                              |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `CLI_ARGS`         | Contain all extra arguments passed after `--` when calling Task through the CLI.                                                                         |
| `TASK`             | The name of the current task.                                                                                                                            |
| `ROOT_DIR`         | The absolute path of the root Taskfile.                                                                                                                  |
| `TASKFILE_DIR`     | The absolute path of the included Taskfile.                                                                                                              |
| `USER_WORKING_DIR` | The absolute path of the directory `task` was called from.                                                                                               |
| `CHECKSUM`         | The checksum of the files listed in `sources`. Only available within the `status` prop and if method is set to `checksum`.                               |
| `TIMESTAMP`        | The date object of the greatest timestamp of the files listed in `sources`. Only available within the `status` prop and if method is set to `timestamp`. |
| `TASK_VERSION`     | The current version of task.                                                                                                                             |
| `ITEM`             | The value of the current iteration when using the `for` property. Can be changed to a different variable name using `as:`.                               |

## ENV

Some environment variables can be overridden to adjust Task behavior.

| ENV                  | Default | Description                                                                                                       |
| -------------------- | ------- | ----------------------------------------------------------------------------------------------------------------- |
| `TASK_TEMP_DIR`      | `.task` | Location of the temp dir. Can relative to the project like `tmp/task` or absolute like `/tmp/.task` or `~/.task`. |
| `TASK_COLOR_RESET`   | `0`     | Color used for white.                                                                                             |
| `TASK_COLOR_BLUE`    | `34`    | Color used for blue.                                                                                              |
| `TASK_COLOR_GREEN`   | `32`    | Color used for green.                                                                                             |
| `TASK_COLOR_CYAN`    | `36`    | Color used for cyan.                                                                                              |
| `TASK_COLOR_YELLOW`  | `33`    | Color used for yellow.                                                                                            |
| `TASK_COLOR_MAGENTA` | `35`    | Color used for magenta.                                                                                           |
| `TASK_COLOR_RED`     | `31`    | Color used for red.                                                                                               |
| `FORCE_COLOR`        |         | Force color output usage.                                                                                         |

## Taskfile Schema

| Attribute  | Type                               | Default       | Description                                                                                                                                                            |
| ---------- | ---------------------------------- | ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `version`  | `string`                           |               | Version of the Taskfile. The current version is `3`.                                                                                                                   |
| `output`   | `string`                           | `interleaved` | Output mode. Available options: `interleaved`, `group` and `prefixed`.                                                                                                 |
| `method`   | `string`                           | `checksum`    | Default method in this Taskfile. Can be overridden in a task by task basis. Available options: `checksum`, `timestamp` and `none`.                                     |
| `includes` | [`map[string]Include`](#include)   |               | Additional Taskfiles to be included.                                                                                                                                   |
| `vars`     | [`map[string]Variable`](#variable) |               | A set of global variables.                                                                                                                                             |
| `env`      | [`map[string]Variable`](#variable) |               | A set of global environment variables.                                                                                                                                 |
| `tasks`    | [`map[string]Task`](#task)         |               | A set of task definitions.                                                                                                                                             |
| `silent`   | `bool`                             | `false`       | Default 'silent' options for this Taskfile. If `false`, can be overridden with `true` in a task by task basis.                                                         |
| `dotenv`   | `[]string`                         |               | A list of `.env` file paths to be parsed.                                                                                                                              |
| `run`      | `string`                           | `always`      | Default 'run' option for this Taskfile. Available options: `always`, `once` and `when_changed`.                                                                        |
| `interval` | `string`                           | `5s`          | Sets a different watch interval when using `--watch`, the default being 5 seconds. This string should be a valid [Go Duration](https://pkg.go.dev/time#ParseDuration). |
| `set`      | `[]string`                         |               | Specify options for the [`set` builtin](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html).                                                      |
| `shopt`    | `[]string`                         |               | Specify option for the [`shopt` builtin](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html).                                                   |

### Include

| Attribute  | Type                  | Default                       | Description                                                                                                                                                                                                                                              |
| ---------- | --------------------- | ----------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `taskfile` | `string`              |                               | The path for the Taskfile or directory to be included. If a directory, Task will look for files named `Taskfile.yml` or `Taskfile.yaml` inside that directory. If a relative path, resolved relative to the directory containing the including Taskfile. |
| `dir`      | `string`              | The parent Taskfile directory | The working directory of the included tasks when run.                                                                                                                                                                                                    |
| `optional` | `bool`                | `false`                       | If `true`, no errors will be thrown if the specified file does not exist.                                                                                                                                                                                |
| `internal` | `bool`                | `false`                       | Stops any task in the included Taskfile from being callable on the command line. These commands will also be omitted from the output when used with `--list`.                                                                                            |
| `aliases`  | `[]string`            |                               | Alternative names for the namespace of the included Taskfile.                                                                                                                                                                                            |
| `vars`     | `map[string]Variable` |                               | A set of variables to apply to the included Taskfile.                                                                                                                                                                                                    |

:::info

Informing only a string like below is equivalent to setting that value to the
`taskfile` attribute.

```yaml
includes:
  foo: ./path
```

:::

### Variable

| Attribute | Type     | Default | Description                                                              |
| --------- | -------- | ------- | ------------------------------------------------------------------------ |
| _itself_  | `string` |         | A static value that will be set to the variable.                         |
| `sh`      | `string` |         | A shell command. The output (`STDOUT`) will be assigned to the variable. |

:::info

Static and dynamic variables have different syntaxes, like below:

```yaml
vars:
  STATIC: static
  DYNAMIC:
    sh: echo "dynamic"
```

:::

### Task

| Attribute       | Type                               | Default                                               | Description                                                                                                                                                                                                                                                                                              |
| --------------- | ---------------------------------- | ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cmds`          | [`[]Command`](#command)            |                                                       | A list of shell commands to be executed.                                                                                                                                                                                                                                                                 |
| `deps`          | [`[]Dependency`](#dependency)      |                                                       | A list of dependencies of this task. Tasks defined here will run in parallel before this task.                                                                                                                                                                                                           |
| `label`         | `string`                           |                                                       | Overrides the name of the task in the output when a task is run. Supports variables.                                                                                                                                                                                                                     |
| `desc`          | `string`                           |                                                       | A short description of the task. This is displayed when calling `task --list`.                                                                                                                                                                                                                           |
| `prompt`        | `string`                           |                                                       | A prompt that will be presented before a task is run. Declining will cancel running the current and any subsequent tasks.                                                                                                                                                                                |
| `summary`       | `string`                           |                                                       | A longer description of the task. This is displayed when calling `task --summary [task]`.                                                                                                                                                                                                                |
| `aliases`       | `[]string`                         |                                                       | A list of alternative names by which the task can be called.                                                                                                                                                                                                                                             |
| `sources`       | `[]string`                         |                                                       | A list of sources to check before running this task. Relevant for `checksum` and `timestamp` methods. Can be file paths or star globs.                                                                                                                                                                   |
| `generates`     | `[]string`                         |                                                       | A list of files meant to be generated by this task. Relevant for `timestamp` method. Can be file paths or star globs.                                                                                                                                                                                    |
| `status`        | `[]string`                         |                                                       | A list of commands to check if this task should run. The task is skipped otherwise. This overrides `method`, `sources` and `generates`.                                                                                                                                                                  |
| `requires`      | `[]string`                         |                                                       | A list of variables which should be set if this task is to run, if any of these variables are unset the task will error and not run.                                                                                                                                                                     |
| `preconditions` | [`[]Precondition`](#precondition)  |                                                       | A list of commands to check if this task should run. If a condition is not met, the task will error.                                                                                                                                                                                                     |
| `requires`      | [`Requires`](#requires)            |                                                       | A list of required variables which should be set if this task is to run, if any variables listed are unset the task will error and not run.                                                                                                                                                              |
| `dir`           | `string`                           |                                                       | The directory in which this task should run. Defaults to the current working directory.                                                                                                                                                                                                                  |
| `vars`          | [`map[string]Variable`](#variable) |                                                       | A set of variables that can be used in the task.                                                                                                                                                                                                                                                         |
| `env`           | [`map[string]Variable`](#variable) |                                                       | A set of environment variables that will be made available to shell commands.                                                                                                                                                                                                                            |
| `dotenv`        | `[]string`                         |                                                       | A list of `.env` file paths to be parsed.                                                                                                                                                                                                                                                                |
| `silent`        | `bool`                             | `false`                                               | Hides task name and command from output. The command's output will still be redirected to `STDOUT` and `STDERR`. When combined with the `--list` flag, task descriptions will be hidden.                                                                                                                 |
| `interactive`   | `bool`                             | `false`                                               | Tells task that the command is interactive.                                                                                                                                                                                                                                                              |
| `internal`      | `bool`                             | `false`                                               | Stops a task from being callable on the command line. It will also be omitted from the output when used with `--list`.                                                                                                                                                                                   |
| `method`        | `string`                           | `checksum`                                            | Defines which method is used to check the task is up-to-date. `timestamp` will compare the timestamp of the sources and generates files. `checksum` will check the checksum (You probably want to ignore the .task folder in your .gitignore file). `none` skips any validation and always run the task. |
| `prefix`        | `string`                           |                                                       | Defines a string to prefix the output of tasks running in parallel. Only used when the output mode is `prefixed`.                                                                                                                                                                                        |
| `ignore_error`  | `bool`                             | `false`                                               | Continue execution if errors happen while executing commands.                                                                                                                                                                                                                                            |
| `run`           | `string`                           | The one declared globally in the Taskfile or `always` | Specifies whether the task should run again or not if called more than once. Available options: `always`, `once` and `when_changed`.                                                                                                                                                                     |
| `platforms`     | `[]string`                         | All platforms                                         | Specifies which platforms the task should be run on. [Valid GOOS and GOARCH values allowed](https://github.com/golang/go/blob/main/src/go/build/syslist.go). Task will be skipped otherwise.                                                                                                             |
| `set`           | `[]string`                         |                                                       | Specify options for the [`set` builtin](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html).                                                                                                                                                                                        |
| `shopt`         | `[]string`                         |                                                       | Specify option for the [`shopt` builtin](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html).                                                                                                                                                                                     |

:::info

These alternative syntaxes are available. They will set the given values to
`cmds` and everything else will be set to their default values:

```yaml
tasks:
  foo: echo "foo"

  foobar:
    - echo "foo"
    - echo "bar"

  baz:
    cmd: echo "baz"
```

:::

#### Command

| Attribute      | Type                               | Default       | Description                                                                                                                                                                                        |
| -------------- | ---------------------------------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cmd`          | `string`                           |               | The shell command to be executed.                                                                                                                                                                  |
| `task`         | `string`                           |               | Set this to trigger execution of another task instead of running a command. This cannot be set together with `cmd`.                                                                                |
| `for`          | [`For`](#for)                      |               | Runs the command once for each given value.                                                                                                                                                        |
| `silent`       | `bool`                             | `false`       | Skips some output for this command. Note that STDOUT and STDERR of the commands will still be redirected.                                                                                          |
| `vars`         | [`map[string]Variable`](#variable) |               | Optional additional variables to be passed to the referenced task. Only relevant when setting `task` instead of `cmd`.                                                                             |
| `ignore_error` | `bool`                             | `false`       | Continue execution if errors happen while executing the command.                                                                                                                                   |
| `defer`        | `string`                           |               | Alternative to `cmd`, but schedules the command to be executed at the end of this task instead of immediately. This cannot be used together with `cmd`.                                            |
| `platforms`    | `[]string`                         | All platforms | Specifies which platforms the command should be run on. [Valid GOOS and GOARCH values allowed](https://github.com/golang/go/blob/main/src/go/build/syslist.go). Command will be skipped otherwise. |
| `set`          | `[]string`                         |               | Specify options for the [`set` builtin](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html).                                                                                  |
| `shopt`        | `[]string`                         |               | Specify option for the [`shopt` builtin](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html).                                                                               |

:::info

If given as a a string, the value will be assigned to `cmd`:

```yaml
tasks:
  foo:
    cmds:
      - echo "foo"
      - echo "bar"
```

:::

#### Dependency

| Attribute | Type                               | Default | Description                                                                                                      |
| --------- | ---------------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------- |
| `task`    | `string`                           |         | The task to be execute as a dependency.                                                                          |
| `vars`    | [`map[string]Variable`](#variable) |         | Optional additional variables to be passed to this task.                                                         |
| `silent`  | `bool`                             | `false` | Hides task name and command from output. The command's output will still be redirected to `STDOUT` and `STDERR`. |

:::tip

If you don't want to set additional variables, it's enough to declare the
dependency as a list of strings (they will be assigned to `task`):

```yaml
tasks:
  foo:
    deps: [foo, bar]
```

:::

#### For

The `for` parameter can be defined as a string, a list of strings or a map. If
it is defined as a string, you can give it any of the following values:

- `source` - Will run the command for each source file defined on the task.
  (Glob patterns will be resolved, so `*.go` will run for every Go file that
  matches).

If it is defined as a list of strings, the command will be run for each value.

Finally, the `for` parameter can be defined as a map when you want to use a
variable to define the values to loop over:

| Attribute | Type     | Default          | Description                                  |
| --------- | -------- | ---------------- | -------------------------------------------- |
| `var`     | `string` |                  | The name of the variable to use as an input. |
| `split`   | `string` | (any whitespace) | What string the variable should be split on. |
| `as`      | `string` | `ITEM`           | The name of the iterator variable.           |

#### Precondition

| Attribute | Type     | Default | Description                                                                                                  |
| --------- | -------- | ------- | ------------------------------------------------------------------------------------------------------------ |
| `sh`      | `string` |         | Command to be executed. If a non-zero exit code is returned, the task errors without executing its commands. |
| `msg`     | `string` |         | Optional message to print if the precondition isn't met.                                                     |

:::tip

If you don't want to set a different message, you can declare a precondition
like this and the value will be assigned to `sh`:

```yaml
tasks:
  foo:
    precondition: test -f Taskfile.yml
```

:::

#### Requires

| Attribute | Type       | Default | Description                                                                                        |
| --------- | ---------- | ------- | -------------------------------------------------------------------------------------------------- |
| `vars`    | `[]string` |         | List of variable or environment variable names that must be set if this task is to execute and run |
