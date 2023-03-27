---
slug: /api/
sidebar_position: 4
toc_min_heading_level: 2
toc_max_heading_level: 5
---

# API 参考

## 命令行

该命令的语法如下：

```bash
task [--flags] [tasks...] [-- CLI_ARGS...]
```

:::tip


如果 `--` 给出，所有剩余参数将被分配给一个特殊的 `CLI_ARGS` 变量

:::


| 缩写   | 标志                          | 类型       | 默认                               | 描述                                                                                                  |
| ---- | --------------------------- | -------- | -------------------------------- | --------------------------------------------------------------------------------------------------- |
| `-c` | `--color`                   | `bool`   | `true`                           | 彩色输出。 默认开启。 设置为 `false` 或使用 `NO_COLOR=1` 禁用。                                                        |
| `-C` | `--concurrency`             | `int`    | `0`                              | 限制并发运行的任务数。 零意味着无限。                                                                                 |
| `-d` | `--dir`                     | `string` | 工作目录                             | 设置执行目录。                                                                                             |
| `-n` | `--dry`                     | `bool`   | `false`                          | 按运行顺序编译和打印任务，而不执行它们。                                                                                |
| `-x` | `--exit-code`               | `bool`   | `false`                          | 传递任务命令的退出代码。                                                                                        |
| `-f` | `--force`                   | `bool`   | `false`                          | 即使任务是最新的也强制执行。                                                                                      |
| `-g` | `--global`                  | `bool`   | `false`                          | 从 `$HOME/Taskfile.{yml,yaml}` 运行全局任务文件。                                                             |
| `-h` | `--help`                    | `bool`   | `false`                          | 显示任务使用情况。                                                                                           |
| `-i` | `--init`                    | `bool`   | `false`                          | 在当前目录创建一个新的 Taskfile.yml。                                                                           |
| `-I` | `--interval`                | `string` | `5s`                             | 使用 `--watch` 设置不同的观察间隔，默认为 5 秒。 这个字符串应该是一个有效的 [Go Duration](https://pkg.go.dev/time#ParseDuration)。 |
| `-l` | `--list`                    | `bool`   | `false`                          | 列出当前文件的全部任务及对应描述。                                                                                   |
| `-a` | `--list-all`                | `bool`   | `false`                          | 列出无论有没有描述的所有任务。                                                                                     |
|      | `--json`                    | `bool`   | `false`                          | 查看 [JSON 输出](#json-输出)                                                                              |
| `-o` | `--output`                  | `string` | 在 Taskfile 中设置默认值或 `intervealed` | 设置输出样式：[`interleaved`/`group`/`prefixed`]。                                                          |
|      | `--output-group-begin`      | `string` |                                  | 在任务组输出前打印的消息模板。                                                                                     |
|      | `--output-group-end`        | `string` |                                  | 在任务组输出后打印的消息模板。                                                                                     |
|      | `--output-group-error-only` | `bool`   | `false`                          | 在退出码为 0 时忽略命令输出。                                                                                    |
| `-p` | `--parallel`                | `bool`   | `false`                          | 并行执行命令行上提供的任务。                                                                                      |
| `-s` | `--silent`                  | `bool`   | `false`                          | 禁用回显。                                                                                               |
|      | `--status`                  | `bool`   | `false`                          | 如果任何给定任务不是最新的，则以非 0 退出码退出。                                                                          |
|      | `--summary`                 | `bool`   | `false`                          | 显示有关任务的摘要。                                                                                          |
| `-t` | `--taskfile`                | `string` | `Taskfile.yml` 或 `Taskfile.yaml` |                                                                                                     |
| `-v` | `--verbose`                 | `bool`   | `false`                          | 启用详细模式。                                                                                             |
|      | `--version`                 | `bool`   | `false`                          | 显示 Task 版本。                                                                                         |
| `-w` | `--watch`                   | `bool`   | `false`                          | 启用给定任务的观察器。                                                                                         |

## JSON 输出

将 `--json` 标志与 `--list` 或 `--list-all` 标志结合使用时，将输出具有以下结构的 JSON 对象：

```jsonc
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
    },
    // ...
  ],
  "location": "/path/to/Taskfile.yml"
}
```

## 特殊变量

模板系统上有一些可用的特殊变量：

| 变量                 | 描述                                                                    |
| ------------------ | --------------------------------------------------------------------- |
| `CLI_ARGS`         | 当通过 CLI 调用 Task 时，传递包含在 `--` 之后的所有额外参数。                               |
| `TASK`             | 当前任务的名称。                                                              |
| `ROOT_DIR`         | 根 Taskfile 的绝对路径。                                                     |
| `TASKFILE_DIR`     | 包含 Taskfile 的绝对路径                                                     |
| `USER_WORKING_DIR` | 调用 `task` 的目录的绝对路径。                                                   |
| `CHECKSUM`         | 在 `sources` 中列出的文件的 checksum。 仅在 `status` 参数中可用，并且如果方法设置为 `checksum`。 |
| `TIMESTAMP`        | `sources` 中列出的文件的最大时间戳的日期对象。 仅在 `status` 参数中可用，并且如果方法设置为 `timestamp`。 |
| `TASK_VERSION`     | Task 的当前版本。                                                           |

## 环境变量

可以覆盖某些环境变量以调整 Task 行为。

| 环境变量                 | 默认      | 描述                                                           |
| -------------------- | ------- | ------------------------------------------------------------ |
| `TASK_TEMP_DIR`      | `.task` | 临时目录的位置。 可以相对于项目比如 `tmp/task` 或绝对如 `/tmp/.task` 或 `~/.task`。 |
| `TASK_COLOR_RESET`   | `0`     | 用于白色的颜色。                                                     |
| `TASK_COLOR_BLUE`    | `34`    | 用于蓝色的颜色。                                                     |
| `TASK_COLOR_GREEN`   | `32`    | 用于绿色的颜色。                                                     |
| `TASK_COLOR_CYAN`    | `36`    | 用于青色的颜色。                                                     |
| `TASK_COLOR_YELLOW`  | `33`    | 用于黄色的颜色。                                                     |
| `TASK_COLOR_MAGENTA` | `35`    | 用于洋红色的颜色。                                                    |
| `TASK_COLOR_RED`     | `31`    | 用于红色的颜色。                                                     |
| `FORCE_COLOR`        |         | 强制使用颜色输出。                                                    |

## Taskfile Schema

| 属性         | 类型                                 | 默认            | 描述                                                                                                |
| ---------- | ---------------------------------- | ------------- | ------------------------------------------------------------------------------------------------- |
| `version`  | `string`                           |               | Taskfile 的版本。 当前版本是 `3`。                                                                          |
| `output`   | `string`                           | `interleaved` | 输出模式。 可用选项: `interleaved`、`group` 和 `prefixed`                                                    |
| `method`   | `string`                           | `checksum`    | Taskfile 中的默认方法。 可以在任务基础上覆盖。 可用选项：`checksum`、`timestamp` 和 `none`。                                |
| `includes` | [`map[string]Include`](#include)   |               | 要包含的其他 Taskfile。                                                                                  |
| `vars`     | [`map[string]Variable`](#variable) |               | 一组全局变量。                                                                                           |
| `env`      | [`map[string]Variable`](#variable) |               | 一组全局环境变量。                                                                                         |
| `tasks`    | [`map[string]Task`](#task)         |               | 一组任务定义。                                                                                           |
| `silent`   | `bool`                             | `false`       | 此任务文件的默认“silent”选项。 如果为 `false`，则可以在任务的基础上用 `true` 覆盖。                                            |
| `dotenv`   | `[]string`                         |               | 要解析的 `.env` 文件路径列表。                                                                               |
| `run`      | `string`                           | `always`      | Taskfile 中默认的 'run' 选项。 可用选项: `always`、`once` 和 `when_changed`。                                   |
| `interval` | `string`                           | `5s`          | 设置 `--watch` 模式下的观察时间，默认 5 秒。 这个字符串应该是一个有效的 [Go Duration](https://pkg.go.dev/time#ParseDuration)。 |
| `set`      | `[]string`                         |               | 为 [内置 `set`](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html) 指定选项。       |
| `shopt`    | `[]string`                         |               | 为 [内置 `shopt`](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html) 指定选项。   |

### Include

| 属性         | 类型                    | 默认             | 描述                                                                                                                 |
| ---------- | --------------------- | -------------- | ------------------------------------------------------------------------------------------------------------------ |
| `taskfile` | `string`              |                | 要包含的 Taskfile 或目录的路径。 如果是目录，Task 将在该目录中查找名为 `Taskfile.yml` 或 `Taskfile.yaml` 的文件。 如果是相对路径，则相对于包含 Taskfile 的目录进行解析。 |
| `dir`      | `string`              | Taskfile 文件父目录 | 运行时包含的任务的工作目录。                                                                                                     |
| `optional` | `bool`                | `false`        | 设置为 `true` 时, 文件不存在也不会报错                                                                                           |
| `internal` | `bool`                | `false`        | 停止在命令行上调用包含的任务文件中的任何任务。 当与 `--list` 一起使用时，这些命令也将从输出中省略。                                                            |
| `aliases`  | `[]string`            |                | 包含的 Taskfile 的命名空间的替代名称。                                                                                           |
| `vars`     | `map[string]Variable` |                | 一组应用于包含的 Taskfile 的变量。                                                                                             |

:::info


像下面这样只赋值一个字符串，和把这个值设置到 `taskfile` 属性是一样的。

```yaml
includes:
  foo: ./path
```

:::


### Variable

| 属性       | 类型       | 默认 | 描述                                 |
| -------- | -------- | -- | ---------------------------------- |
| *itself* | `string` |    | 将设置为变量的静态值。                        |
| `sh`     | `string` |    | 一个 shell 命令。 输出 (`STDOUT`) 将分配给变量。 |

:::info


静态和动态变量有不同的语法，如下所示：

```yaml
vars:
  STATIC: static
  DYNAMIC:
    sh: echo "dynamic"
```

:::


### Task

| 属性              | 类型                                 | 默认                         | 描述                                                                                                                          |
| --------------- | ---------------------------------- | -------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| `cmds`          | [`[]Command`](#command)            |                            | 要执行的 shell 命令列表。                                                                                                            |
| `deps`          | [`[]Dependency`](#dependency)      |                            | 此任务的依赖项列表。 此处定义的任务将在此任务之前并行运行。                                                                                              |
| `label`         | `string`                           |                            | 运行任务时覆盖输出中的任务名称。 支持变量。                                                                                                      |
| `desc`          | `string`                           |                            | Task 的简短描述。 这在调用 `task --list` 时显示。                                                                                         |
| `summary`       | `string`                           |                            | 任务的较长描述。 这在调用 `task --summary [task]` 时显示。                                                                                  |
| `aliases`       | `[]string`                         |                            | 可以调用任务的别名列表。                                                                                                                |
| `sources`       | `[]string`                         |                            | 运行此任务之前要检查的源列表。 与 `checksum` 和 `timestamp` 相关。 可以是文件路径或星号。                                                                  |
| `generates`     | `[]string`                         |                            | 此任务要生成的文件列表。 与 `timestamp` 方法相关。 可以是文件路径或星号。                                                                                |
| `status`        | `[]string`                         |                            | 用于检查此 task 是否应运行的命令列表。 否则跳过该任务。 这个方法会覆盖 `method`、`sources` 和 `generates`。                                                   |
| `preconditions` | [`[]Precondition`](#precondition)  |                            | 用于检查此任务是否应运行的命令列表。 如果不满足条件，任务将出错。                                                                                           |
| `dir`           | `string`                           |                            | 此 task 应运行的目录。 默认为当前工作目录。                                                                                                   |
| `vars`          | [`map[string]Variable`](#variable) |                            | 可在 task 中使用的一组变量。                                                                                                           |
| `env`           | [`map[string]Variable`](#variable) |                            | 一组可用于 shell 命令的环境变量。                                                                                                        |
| `dotenv`        | `[]string`                         |                            | 要解析的 `.env` 文件路径列表。                                                                                                         |
| `silent`        | `bool`                             | `false`                    | 从输出中隐藏任务名称和命令。 命令的输出仍将重定向到 `STDOUT` 和 `STDERR`。 当与 `--list` 标志结合使用时，任务描述将被隐藏。                                               |
| `interactive`   | `bool`                             | `false`                    | 告诉任务该命令是交互式的。                                                                                                               |
| `internal`      | `bool`                             | `false`                    | 停止在命令行上调用任务。 当与 `--list` 一起使用时，它也会从输出中省略。                                                                                   |
| `method`        | `string`                           | `checksum`                 | 定义用于检查任务是最新的方法。 `timestamp` 将比较源的时间戳并生成文件。 `checksum` 将检查 checksum（您可能想忽略 .gitignore 文件中的 .task 文件夹）。 `none` 跳过任何验证并始终运行任务。 |
| `prefix`        | `string`                           |                            | 定义一个字符串作为并行运行 task 输出的前缀。 仅在输出模式是 `prefixed` 时使用。                                                                           |
| `ignore_error`  | `bool`                             | `false`                    | 如果执行命令时发生错误，则继续执行。                                                                                                          |
| `run`           | `string`                           | Taskfile 中全局声明的值或 `always` | 指定如果多次调用该任务是否应再次运行。 可用选项：`always`、`once` 和 `when_changed`。                                                                  |
| `platforms`     | `[]string`                         | 所有平台                       | 指定应在哪些平台上运行任务。 允许使用 [有效的 GOOS 和 GOARCH 值](https://github.com/golang/go/blob/master/src/go/build/syslist.go)。 否则将跳过任务。       |
| `set`           | `[]string`                         |                            | 为 [内置 `set`](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html) 指定选项。                                 |
| `shopt`         | `[]string`                         |                            | 为 [内置 `shopt`](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html) 指定选项。                             |

:::info


这些替代语法可用。 他们会将给定值设置为 `cmds`，其他所有内容都将设置为其默认值：

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

| 属性             | 类型                                 | 默认      | 描述                                                                                                                     |
| -------------- | ---------------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------- |
| `cmd`          | `string`                           |         | 要执行的 shell 命令                                                                                                          |
| `silent`       | `bool`                             | `false` | 跳过此命令的一些输出。 请注意，命令的 STDOUT 和 STDERR 仍将被重定向。                                                                            |
| `task`         | `string`                           |         | 执行另一个 task，而不执行命令。 不能与 `cmd` 同时设置。                                                                                     |
| `vars`         | [`map[string]Variable`](#variable) |         | 要传递给引用任务的可选附加变量。 仅在设置 `task` 而不是 `cmd` 时相关。                                                                            |
| `ignore_error` | `bool`                             | `false` | 执行命令的时候忽略错误，继续执行                                                                                                       |
| `defer`        | `string`                           |         | `cmd` 的替代方法，但安排命令在此任务结束时执行，而不是立即执行。 不能与 `cmd` 一同使用。                                                                    |
| `platforms`    | `[]string`                         | 所有平台    | 指定应在哪些平台上运行该命令。 允许使用 [有效的 GOOS 和 GOARCH 值](https://github.com/golang/go/blob/master/src/go/build/syslist.go)。 否则将跳过命令。 |
| `set`          | `[]string`                         |         | 为 [内置 `set`](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html) 指定选项。                            |
| `shopt`        | `[]string`                         |         | 为 [内置 `shopt`](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html) 指定选项。                        |

:::info


如果以字符串形式给出，该值将分配给 `cmd`：

```yaml
tasks:
  foo:
    cmds:
      - echo "foo"
      - echo "bar"
```

:::


#### Dependency

| 属性     | 类型                                 | 默认 | 描述              |
| ------ | ---------------------------------- | -- | --------------- |
| `task` | `string`                           |    | 要作为依赖项执行的任务。    |
| `vars` | [`map[string]Variable`](#variable) |    | 要传递给此任务的可选附加变量。 |

:::tip


如果你不想设置额外的变量，将依赖关系声明为一个字符串列表就足够了（它们将被分配给 `task`）。

```yaml
tasks:
  foo:
    deps: [foo, bar]
```

:::


#### Precondition

| 属性    | 类型       | 默认 | 描述                                  |
| ----- | -------- | -- | ----------------------------------- |
| `sh`  | `string` |    | 要执行的命令。 如果返回非零退出码，任务将在不执行其命令的情况下出错。 |
| `msg` | `string` |    | 如果不满足先决条件，则打印可选消息。                  |

:::tip


如果你不想设置不同的消息，你可以像这样声明一个前提条件，值将被分配给 `sh`：

```yaml
tasks:
  foo:
    precondition: test -f Taskfile.yml
```

:::
