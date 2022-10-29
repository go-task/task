---
slug: /api/
sidebar_position: 4
---

# API Reference

## CLI

Task command line tool has the following syntax:

```bash
task [--flags] [tasks...] [-- CLI_ARGS...]
```

:::tip

If `--` is given, all remaning arguments will be assigned to a special `CLI_ARGS`
variable

:::

| Short | Flag | Type | Default | Description |
| - | - | - | - | - |
| `-c` | `--color` | `bool` | `true` | Colored output. Enabled by default. Set flag to `false` or use `NO_COLOR=1` to disable. |
| `-C` | `--concurrency` | `int` | `0` | Limit number tasks to run concurrently. Zero means unlimited. |
| `-d` | `--dir` | `string` | Working directory | Sets directory of execution. |
| `-n` | `--dry` | `bool` | `false` | Compiles and prints tasks in the order that they would be run, without executing them. |
| `-x` | `--exit-code` | `bool` | `false` | Pass-through the exit code of the task command. |
| `-f` | `--force` | `bool` | `false` | Forces execution even when the task is up-to-date. |
| `-h` | `--help` | `bool` | `false` | Shows Task usage. |
| `-i` | `--init` | `bool` | `false` | Creates a new Taskfile.yaml in the current folder. |
| `-I` | `--interval` | `string` | `5s` | Sets a different watch interval when using `--watch`, the default being 5 seconds. This string should be a valid [Go Duration](https://pkg.go.dev/time#ParseDuration). |
| `-l` | `--list` | `bool` | `false` | Lists tasks with description of current Taskfile. |
| `-a` | `--list-all` | `bool` | `false` | Lists tasks with or without a description. |
| `-o` | `--output` | `string` | Default set in the Taskfile or `intervealed` | Sets output style: [`interleaved`/`group`/`prefixed`]. |
|      | `--output-group-begin` | `string` | | Message template to print before a task's grouped output. |
|      | `--output-group-end` | `string` | | Message template to print after a task's grouped output. |
| `-p` | `--parallel` | `bool` | `false` | Executes tasks provided on command line in parallel. |
| `-s` | `--silent` | `bool` | `false` | Disables echoing. |
|      | `--status` | `bool` | `false` | Exits with non-zero exit code if any of the given tasks is not up-to-date. |
|      | `--summary` | `bool` | `false` | Show summary about a task. |
| `-t` | `--taskfile` | `string` | `Taskfile.yml` or `Taskfile.yaml` | |
| `-v` | `--verbose` | `bool` | `false` | Enables verbose mode. |
|      | `--version` | `bool` | `false` | Show Task version. |
| `-w` | `--watch` | `bool` | `false` | Enables watch of the given task. |

## Special Variables

There are some special variables that is available on the templating system:

| Var | Description |
| - | - |
| `CLI_ARGS` | Contain all extra arguments passed after `--` when calling Task through the CLI. |
| `TASK` | The name of the current task. |
| `ROOT_DIR` | The absolute path of the root Taskfile. |
| `TASKFILE_DIR` | The absolute path of the included Taskfile. |
| `CHECKSUM` | The checksum of the files listed in `sources`. Only available within the `status` prop and if method is set to `checksum`. |
| `TIMESTAMP` | The date object of the greatest timestamp of the files listes in `sources`. Only available within the `status` prop and if method is set to `timestamp`. |

## ENV

Some environment variables can be overriden to adjust Task behavior.

| ENV | Default | Description |
| - | - | - |
| `TASK_TEMP_DIR` | `.task` | Location of the temp dir. Can relative to the project like `tmp/task` or absolute like `/tmp/.task` or `~/.task`. |
| `TASK_COLOR_RESET` | `0` | Color used for white. |
| `TASK_COLOR_BLUE` | `34` | Color used for blue. |
| `TASK_COLOR_GREEN` | `32` | Color used for green. |
| `TASK_COLOR_CYAN` | `36` | Color used for cyan. |
| `TASK_COLOR_YELLOW` | `33` | Color used for yellow. |
| `TASK_COLOR_MAGENTA` | `35` | Color used for magenta. |
| `TASK_COLOR_RED` | `31` | Color used for red. |

## Schema

### Taskfile

| Attribute | Type | Default | Description |
| - | - | - | - |
| `version` | `string` | | Version of the Taskfile. The current version is `3`. |
| `output` | `string` | `interleaved` | Output mode. Available options: `interleaved`, `group` and `prefixed`. |
| `method` | `string` | `checksum` | Default method in this Taskfile. Can be overriden in a task by task basis. Available options: `checksum`, `timestamp` and `none`. |
| `includes` | [`map[string]Include`](#include) | | Additional Taskfiles to be included. |
| `vars` | [`map[string]Variable`](#variable) | | A set of global variables. |
| `env` | [`map[string]Variable`](#variable) | | A set of global environment variables. |
| `tasks` | [`map[string]Task`](#task) | | A set of task definitions. |
| `silent` | `bool` | `false` | Default 'silent' options for this Taskfile. If `false`, can be overidden with `true` in a task by task basis. |
| `dotenv` | `[]string` | | A list of `.env` file paths to be parsed. |
| `run` | `string` | `always` | Default 'run' option for this Taskfile. Available options: `always`, `once` and `when_changed`. |
| `interval` | `string` | `5s` | Sets a different watch interval when using `--watch`, the default being 5 seconds. This string should be a valid [Go Duration](https://pkg.go.dev/time#ParseDuration). |

### Include

| Attribute | Type | Default | Description |
| - | - | - | - |
| `taskfile` | `string` | | The path for the Taskfile or directory to be included. If a directory, Task will look for files named `Taskfile.yml` or `Taskfile.yaml` inside that directory. If a relative path, resolved relative to the directory containing the including Taskfile. |
| `dir` | `string` | The parent Taskfile directory | The working directory of the included tasks when run. |
| `optional` | `bool` | `false` | If `true`, no errors will be thrown if the specified file does not exist. |
| `internal` | `bool` | `false` | Stops any task in the included Taskfile from being callable on the command line. These commands will also be omitted from the output when used with `--list`. |
| `aliases` | `[]string` | | Alternative names for the namespace of the included Taskfile. |
| `vars` | `map[string]Variable` | | A set of variables to apply to the included Taskfile. |

:::info

Informing only a string like below is equivalent to setting that value to the `taskfile` attribute.

```yaml
includes:
  foo: ./path
```

:::

### Task

| Attribute | Type | Default | Description |
| - | - | - | - |
| `cmds` | [`[]Command`](#command) | | A list of shell commands to be executed. |
| `deps` | [`[]Dependency`](#dependency) | | A list of dependencies of this task. Tasks defined here will run in parallel before this task. |
| `label` | `string` | | Overrides the name of the task in the output when a task is run. Supports variables. |
| `desc` | `string` | | A short description of the task. This is displayed when calling `task --list`. |
| `summary` | `string` | | A longer description of the task. This is displayed when calling `task --summary [task]`. |
| `aliases` | `[]string` | | A list of alternative names by which the task can be called. |
| `sources` | `[]string` | | A list of sources to check before running this task. Relevant for `checksum` and `timestamp` methods. Can be file paths or star globs. |
| `generates` | `[]string` | | A list of files meant to be generated by this task. Relevant for `timestamp` method. Can be file paths or star globs. |
| `status` | `[]string` | | A list of commands to check if this task should run. The task is skipped otherwise. This overrides `method`, `sources` and `generates`. |
| `preconditions` | [`[]Precondition`](#precondition) | | A list of commands to check if this task should run. If a condition is not met, the task will error. |
| `dir` | `string` | | The directory in which this task should run. Defaults to the current working directory. |
| `vars` | [`map[string]Variable`](#variable) | | A set of variables that can be used in the task. |
| `env` | [`map[string]Variable`](#variable) | | A set of environment variables that will be made available to shell commands. |
| `silent` | `bool` | `false` | Hides task name and command from output. The command's output will still be redirected to `STDOUT` and `STDERR`. When combined with the `--list` flag, task descriptions will be hidden. |
| `interactive` | `bool` | `false` | Tells task that the command is interactive. |
| `internal` | `bool` | `false` | Stops a task from being callable on the command line. It will also be omitted from the output when used with `--list`. |
| `method` | `string` | `checksum` | Defines which method is used to check the task is up-to-date. `timestamp` will compare the timestamp of the sources and generates files. `checksum` will check the checksum (You probably want to ignore the .task folder in your .gitignore file). `none` skips any validation and always run the task. |
| `prefix` | `string` | | Defines a string to prefix the output of tasks running in parallel. Only used when the output mode is `prefixed`. |
| `ignore_error` | `bool` | `false` | Continue execution if errors happen while executing commands. |
| `run` | `string` | The one declared globally in the Taskfile or `always` | Specifies whether the task should run again or not if called more than once. Available options: `always`, `once` and `when_changed`. |

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

### Dependency

| Attribute | Type | Default | Description |
| - | - | - | - |
| `task` | `string` | | The task to be execute as a dependency. |
| `vars` | [`map[string]Variable`](#variable) | | Optional additional variables to be passed to this task. |

:::tip

If you don't want to set additional variables, it's enough to declare the
dependency as a list of strings (they will be assigned to `task`):

```yaml
tasks:
  foo:
    deps: [foo, bar]
```

:::

### Command

| Attribute | Type | Default | Description |
| - | - | - | - |
| `cmd` | `string` | | The shell command to be executed. |
| `silent` | `bool` | `false` | Skips some output for this command. Note that STDOUT and STDERR of the commands will still be redirected. |
| `task` | `string` | | Set this to trigger execution of another task instead of running a command. This cannot be set together with `cmd`. |
| `vars` | [`map[string]Variable`](#variable) | | Optional additional variables to be passed to the referenced task. Only relevant when setting `task` instead of `cmd`. |
| `ignore_error` | `bool` | `false` | Continue execution if errors happen while executing the command. |
| `defer` | `string` | | Alternative to `cmd`, but schedules the command to be executed at the end of this task instead of immediately. This cannot be used together with `cmd`. |

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

### Variable

| Attribute | Type | Default | Description |
| - | - | - | - |
| *itself* | `string` | | A static value that will be set to the variable. |
| `sh` | `string` | | A shell command. The output (`STDOUT`) will be assigned to the variable. |

:::info

Static and dynamic variables have different syntaxes, like below:

```yaml
vars:
  STATIC: static
  DYNAMIC:
    sh: echo "dynamic"
```

:::

### Precondition

| Attribute | Type | Default | Description |
| - | - | - | - |
| `sh` | `string` | | Command to be executed. If a non-zero exit code is returned, the task errors without executing its commands. |
| `msg` | `string` | | Optional message to print if the precondition isn't met. |

:::tip

If you don't want to set a different message, you can declare a precondition
like this and the value will be assigned to `sh`:

```yaml
tasks:
  foo:
    precondition: test -f Taskfile.yml
```

:::
