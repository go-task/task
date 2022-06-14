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

If `--` is given, all remaning arguments to assigned to a special `CLI_ARGS`
variable

:::

| Short | Flag | Type | Default | Description |
| - | - | - | - | - |
| `-c` | `--color` | `bool` | `true` | Colored output. Enabled by default. Set flag to `false` or use `NO_COLOR=1` to disable. |
| `-C` | `--concurrency` | `int` | `0` | Limit number tasks to run concurrently. Zero means unlimited. |
| `-d` | `--dir` | `string` | Working directory | Sets directory of execution. |
| '-n' | `--dry` | `bool` | `false` | Compiles and prints tasks in the order that they would be run, without executing them. |
| `-x` | `--exit-code` | `bool` | `false` | Pass-through the exit code of the task command. |
| `-f` | `--force` | `bool` | `false` | Forces execution even when the task is up-to-date. |
| `-h` | `--help` | `bool` | `false` | Shows Task usage. |
| `-i` | `--init` | `bool` | `false` | Creates a new Taskfile.yaml in the current folder. |
| `-l` | `--list` | `bool` | `false` | Lists tasks with description of current Taskfile. |
| `-a` | `--list-all` | `bool` | `false` | Lists tasks with or without a description. |
| `-o` | `--output` | `string` | Default set in the Taskfile or `intervealed` | Sets output style: [`interleaved`/`group`/`prefixed`]. |
|      | `--output-group-begin` | `string` | | Message template to print before a task's grouped output. |
|      | `--output-group-end `  | `string` | | Message template to print after a task's grouped output. |
| `-p` | `--parallel` | `bool` | `false` | Executes tasks provided on command line in parallel. |
| `-s` | `--silent` | `bool` | `false` | Disables echoing. |
|      | `--status` | `bool` | `false` | Exits with non-zero exit code if any of the given tasks is not up-to-date. |
|      | `--summary` | `bool` | `false` | Show summary about a task. |
| `-t` | `--taskfile` | `string` | `Taskfile.yml` or `Taskfile.yaml` | |
| `-v` | `--verbose` | `bool` | `false` | Enables verbose mode. |
|      | `--version` | `bool` | `false` | Show Task version. |
| `-w` | `--watch` | `bool` | `false` | Enables watch of the given task. |

## Schema

### Taskfile

| Attribute | Type | Default | Description |
| - | - | - | - |
| `version` | `string` | | Version of the Taskfile. The current version is `3`. |
| `includes` | [`map[string]Include`](#include) | | Additional Taskfiles to be included. |
| `output` | `string` | `interleaved` | Output mode. Available options: `interleaved`, `group` and `prefixed`. |
| `method` | `string` | `checksum` | Default method in this Taskfile. Can be overriden in a task by task basis. Available options: `checksum`, `timestamp` and `none`. |
| `silent` | `bool` | `false` | Default "silent" options for this Taskfile. If `false`, can be overidden with `true` in a task by task basis. |
| `run` | `string` | `always` | Default "run" option for this Taskfile. Available options: `always`, `once` and `when_changed`. |
| `vars` | [`map[string]Variable`](#variable) | | Global variables. |
| `env` | [`map[string]Variable`](#variable) | | Global environment. |
| `dotenv` | `[]string` | | A list of `.env` file paths to be parsed. |
| `tasks` | [`map[string]Task`](#task) | | The task definitions. |


### Include

| Attribute | Type | Default | Description |
| - | - | - | - |
| `taskfile` | `string` | | The path for the Taskfile or directory to be included. If a directory, Task will look for files named `Taskfile.yml` or `Taskfile.yaml` inside that directory. |
| `dir` | `string` | The parent Taskfile directory | The working directory of the included tasks when run. |
| `optional` | `bool` | `false` | If `true`, no errors will be thrown if the specified file does not exist. |

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
| `desc` | `string` | | A short description of the task. This is listed when calling `task --list`. |
| `summary` | `string` | | A longer description of the task. This is listed when calling `task --summary [task]`. |
| `sources` | `[]string` | | List of sources to check before running this task. Relevant for `checksum` and `timestamp` methods. Can be file paths or star globs. |
| `dir` | `string` | | The current directory which this task should run. |
| `method` | `string` | `checksum` | Method used by this task. Default to the one declared globally or `checksum`. Available options: `checksum`, `timestamp` and `none` |
| `silent` | `bool` | `false` | Skips some output for this task. Note that STDOUT and STDERR of the commands will still be redirected. |
| `run` | `string` | The one declared globally in the Taskfile or `always` | Specifies whether the task should run again or not if called more than once. Available options: `always`, `once` and `when_changed`. |
| `prefix` | `string` | | Allows to override the prefix print before the STDOUT. Only relevant when using the `prefixed` output mode. |
| `ignore_error` | `bool` | `false` | Continue execution if errors happen while executing the commands. |
| `generates` | `[]string` | | List of files meant to be generated by this task. Relevant for `timestamp` method. Can be file paths or star globs. |
| `status` | `[]string` | | List of commands to check if this task should run. The task is skipped otherwise. This overrides `method`, `sources` and `generates`. |
| `preconditions` | [`[]Precondition`](#precondition) | | List of commands to check if this task should run. The task errors otherwise. |
| `vars` | [`map[string]Variable`](#variable) | | Task variables. |
| `env` | [`map[string]Variable`](#variable) | | Task environment. |
| `deps` | [`[]Dependency`](#dependency) | | List of dependencies of this task. |
| `cmds` | [`[]Command`](#command) | | List of commands to be executed. |

:::info

These alternative syntaxes are available. They will set the given values to
`cmds` and everything else will be set to their default values:

```yaml
tasks:
  foo: echo "foo"

  foobar:
    - echo "foo"
    - echo "bar"
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
| `defer` | `string` | | Alternative to `cmd`, but schedules the command to be executed at the end of this task instead of immediately. This cannot be used together with `cmd`. |
| `silent` | `bool` | `false` | Skips some output for this command. Note that STDOUT and STDERR of the commands will still be redirected. |
| `ignore_error` | `bool` | `false` | Continue execution if errors happen while executing the command. |
| `task` | `string` | | Set this to trigger execution of another task instead of running a command. This cannot be set together with `cmd`. |
| `vars` | [`map[string]Variable`](#variable) | | Optional additional variables to be passed to the referenced task. Only relevant when setting `task` instead of `cmd`. |

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
