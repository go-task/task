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

If `--` is given, all remaining arguments will be assigned to a special `CLI_ARGS` variable

:::

| 缩写   | 标志                          | 类型       | 默认                               | 描述                                                                                                                                                                                                             |
| ---- | --------------------------- | -------- | -------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `-c` | `--color`                   | `bool`   | `true`                           | 彩色输出。 默认开启。 设置为 `false` 或使用 `NO_COLOR=1` 禁用。                                                                                                                                                                   |
| `-C` | `--concurrency`             | `int`    | `0`                              | 限制并发运行的 task 数。 零意味着无限。                                                                                                                                                                                        |
| `-d` | `--dir`                     | `string` | 工作目录                             | 设置执行目录。                                                                                                                                                                                                        |
| `-n` | `--dry`                     | `bool`   | `false`                          | 按运行顺序编译和打印 task，而不执行它们。                                                                                                                                                                                        |
| `-x` | `--exit-code`               | `bool`   | `false`                          | 传递 task 命令的退出代码。                                                                                                                                                                                               |
| `-f` | `--force`                   | `bool`   | `false`                          | 即使 task 是最新的也强制执行。                                                                                                                                                                                             |
| `-g` | `--global`                  | `bool`   | `false`                          | 从 `$HOME/Taskfile.{yml,yaml}` 运行全局 Taskfile。                                                                                                                                                                   |
| `-h` | `--help`                    | `bool`   | `false`                          | 显示 Task 使用情况。                                                                                                                                                                                                  |
| `-i` | `--init`                    | `bool`   | `false`                          | 在当前目录创建一个新的 Taskfile.yml。                                                                                                                                                                                      |
| `-I` | `--interval`                | `string` | `5s`                             | 使用 `--watch` 设置不同的观察间隔，默认为 5 秒。 这个字符串应该是一个有效的 [Go Duration](https://pkg.go.dev/time#ParseDuration)。                                                                                                            |
| `-l` | `--list`                    | `bool`   | `false`                          | 列出当前文件的全部 task 及对应描述。                                                                                                                                                                                          |
| `-a` | `--list-all`                | `bool`   | `false`                          | 列出无论有没有描述的所有 task。                                                                                                                                                                                             |
|      | `--sort`                    | `string` | `default`                        | Changes the order of the tasks when listed.<br />`default` - Alphanumeric with root tasks first<br />`alphanumeric` - Alphanumeric<br />`none` - No sorting (As they appear in the Taskfile) |
|      | `--json`                    | `bool`   | `false`                          | 查看 [JSON 输出](#json-输出)                                                                                                                                                                                         |
| `-o` | `--output`                  | `string` | 在 Taskfile 中设置默认值或 `intervealed` | 设置输出样式：[`interleaved`/`group`/`prefixed`]。                                                                                                                                                                     |
|      | `--output-group-begin`      | `string` |                                  | 在任务组输出前打印的消息模板。                                                                                                                                                                                                |
|      | `--output-group-end`        | `string` |                                  | 在任务组输出后打印的消息模板。                                                                                                                                                                                                |
|      | `--output-group-error-only` | `bool`   | `false`                          | 在退出码为 0 时忽略命令输出。                                                                                                                                                                                               |
| `-p` | `--parallel`                | `bool`   | `false`                          | 并行执行命令行上提供的 task。                                                                                                                                                                                              |
| `-s` | `--silent`                  | `bool`   | `false`                          | 禁用回显。                                                                                                                                                                                                          |
| `-y` | `--yes`                     | `bool`   | `false`                          | Assume "yes" as answer to all prompts.                                                                                                                                                                         |
|      | `--status`                  | `bool`   | `false`                          | 如果任何给定 task 不是最新的，则以非 0 退出码退出。                                                                                                                                                                                 |
|      | `--summary`                 | `bool`   | `false`                          | 显示有关 task 的摘要。                                                                                                                                                                                                 |
| `-t` | `--taskfile`                | `string` | `Taskfile.yml` 或 `Taskfile.yaml` |                                                                                                                                                                                                                |
| `-v` | `--verbose`                 | `bool`   | `false`                          | 启用详细模式。                                                                                                                                                                                                        |
|      | `--version`                 | `bool`   | `false`                          | 显示 Task 版本。                                                                                                                                                                                                    |
| `-w` | `--watch`                   | `bool`   | `false`                          | 启用给定 task 的观察器。                                                                                                                                                                                                |

## 退出码

Task 有时会以特定的退出代码退出。 These codes are split into three groups with the following ranges:

- General errors (0-99)
- Taskfile errors (100-199)
- Task errors (200-299)

可以在下面找到退出代码及其描述的完整列表：

| 代码  | 描述                                                       |
| --- | -------------------------------------------------------- |
| 0   | 成功                                                       |
| 1   | 出现未知错误                                                   |
| 100 | 找不到 Taskfile                                             |
| 101 | 尝试初始化一个 Taskfile 时已经存在                                   |
| 102 | Taskfile 无效或无法解析                                         |
| 103 | A remote Taskfile could not be downlaoded                |
| 104 | A remote Taskfile was not trusted by the user            |
| 105 | A remote Taskfile was could not be fetched securely      |
| 106 | No cache was found for a remote Taskfile in offline mode |
| 107 | No schema version was defined in the Taskfile            |
| 200 | 找不到指定的 task                                              |
| 201 | 在 task 中执行命令时出错                                          |
| 202 | 用户试图调用内部 task                                            |
| 203 | 有多个具有相同名称或别名的 task                                       |
| 204 | 一个 task 被调用了太多次                                          |
| 205 | 操作被用户取消                                                  |
| 206 | 由于缺少所需变量，任务未执行                                           |

这些代码也可以在代码库的 [`errors/errors.go`](https://github.com/go-task/task/blob/main/errors/errors.go) 文件中找到。

:::info

当使用 `-x`/`--exit-code` 标志运行 Task 时，任何失败命令的退出代码都将传递给用户。

:::

## JSON 输出

将 `--json` 标志与 `--list` 或 `--list-all` 标志结合使用时，将输出具有以下结构的 JSON 对象：

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

## 特殊变量

模板系统上有一些可用的特殊变量：

| 变量                 | 描述                                                                                                                         |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| `CLI_ARGS`         | 当通过 CLI 调用 Task 时，传递包含在 `--` 之后的所有额外参数。                                                                                    |
| `TASK`             | 当前 task 的名称。                                                                                                               |
| `ROOT_DIR`         | 根 Taskfile 的绝对路径。                                                                                                          |
| `TASKFILE_DIR`     | 包含 Taskfile 的绝对路径                                                                                                          |
| `USER_WORKING_DIR` | 调用 `task` 的目录的绝对路径。                                                                                                        |
| `CHECKSUM`         | 在 `sources` 中列出的文件的 checksum。 仅在 `status` 参数中可用，并且如果 method 设置为 `checksum`。                                                |
| `TIMESTAMP`        | The date object of the greatest timestamp of the files listed in `sources`. 仅在 `status` 参数中可用，并且如果 method 设置为 `timestamp`。 |
| `TASK_VERSION`     | Task 的当前版本。                                                                                                                |
| `ITEM`             | The value of the current iteration when using the `for` property. Can be changed to a different variable name using `as:`. |

## 环境变量

Some environment variables can be overridden to adjust Task behavior.

| ENV                  | 默认      | 描述                                                           |
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
| `method`   | `string`                           | `checksum`    | Taskfile 中的默认方法。 Can be overridden in a task by task basis. 可用选项：`checksum`、`timestamp` 和 `none`。 |
| `includes` | [`map[string]Include`](#include)   |               | 要包含的其他 Taskfile。                                                                                  |
| `vars`     | [`map[string]Variable`](#variable) |               | 一组全局变量。                                                                                           |
| `env`      | [`map[string]Variable`](#variable) |               | 一组全局环境变量。                                                                                         |
| `tasks`    | [`map[string]Task`](#task)         |               | 一组 task 定义。                                                                                       |
| `silent`   | `bool`                             | `false`       | 此 Taskfile 的默认“silent”选项。 If `false`, can be overridden with `true` in a task by task basis.      |
| `dotenv`   | `[]string`                         |               | 要解析的 `.env` 文件路径列表。                                                                               |
| `run`      | `string`                           | `always`      | Taskfile 中默认的 'run' 选项。 可用选项: `always`、`once` 和 `when_changed`。                                   |
| `interval` | `string`                           | `5s`          | 设置 `--watch` 模式下的观察时间，默认 5 秒。 这个字符串应该是一个有效的 [Go Duration](https://pkg.go.dev/time#ParseDuration)。 |
| `set`      | `[]string`                         |               | 为 [内置 `set`](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html) 指定选项。       |
| `shopt`    | `[]string`                         |               | 为 [内置 `shopt`](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html) 指定选项。   |

### Include

| 属性         | 类型                    | 默认             | 描述                                                                                                                 |
| ---------- | --------------------- | -------------- | ------------------------------------------------------------------------------------------------------------------ |
| `taskfile` | `string`              |                | 要包含的 Taskfile 或目录的路径。 如果是目录，Task 将在该目录中查找名为 `Taskfile.yml` 或 `Taskfile.yaml` 的文件。 如果是相对路径，则相对于包含 Taskfile 的目录进行解析。 |
| `dir`      | `string`              | Taskfile 文件父目录 | 运行时包含的 task 的工作目录。                                                                                                 |
| `optional` | `bool`                | `false`        | 设置为 `true` 时, 文件不存在也不会报错                                                                                           |
| `internal` | `bool`                | `false`        | 停止在命令行上调用包含的 Taskfile 中的任何 task。 当与 `--list` 一起使用时，这些命令也将从输出中省略。                                                   |
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
| _itself_ | `string` |    | 将设置为变量的静态值。                        |
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

| 属性              | 类型                                 | 默认                         | 描述                                                                                                                                          |
| --------------- | ---------------------------------- | -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `cmds`          | [`[]Command`](#command)            |                            | 要执行的 shell 命令列表。                                                                                                                            |
| `deps`          | [`[]Dependency`](#dependency)      |                            | 此 task 的依赖项列表。 此处定义的 task 将在此 task 之前并行运行。                                                                                                  |
| `label`         | `string`                           |                            | 运行 task 时覆盖输出中的 task 名称。 支持变量。                                                                                                              |
| `desc`          | `string`                           |                            | task 的简短描述。 这在调用 `task --list` 时显示。                                                                                                         |
| `prompt`        | `string`                           |                            | A prompt that will be presented before a task is run. Declining will cancel running the current and any subsequent tasks.                   |
| `summary`       | `string`                           |                            | task 的较长描述。 这在调用 `task --summary [task]` 时显示。                                                                                               |
| `aliases`       | `[]string`                         |                            | 可以调用 task 的别名列表。                                                                                                                            |
| `sources`       | `[]string`                         |                            | 运行此 task 之前要检查的源列表。 与 `checksum` 和 `timestamp` 方法相关。 可以是文件路径或星号。                                                                            |
| `generates`     | `[]string`                         |                            | 此 task 要生成的文件列表。 与 `timestamp` 方法相关。 可以是文件路径或星号。                                                                                            |
| `status`        | `[]string`                         |                            | 用于检查此 task 是否应运行的命令列表。 否则跳过该 task。 这个方法会覆盖 `method`、`sources` 和 `generates`。                                                                |
| `requires`      | `[]string`                         |                            | A list of variables which should be set if this task is to run, if any of these variables are unset the task will error and not run.        |
| `preconditions` | [`[]Precondition`](#precondition)  |                            | 用于检查此 task 是否应运行的命令列表。 如果不满足条件，task 将出错。                                                                                                    |
| `requires`      | [`Requires`](#requires)            |                            | A list of required variables which should be set if this task is to run, if any variables listed are unset the task will error and not run. |
| `dir`           | `string`                           |                            | 此 task 应运行的目录。 默认为当前工作目录。                                                                                                                   |
| `vars`          | [`map[string]Variable`](#variable) |                            | 可在 task 中使用的一组变量。                                                                                                                           |
| `env`           | [`map[string]Variable`](#variable) |                            | 一组可用于 shell 命令的环境变量。                                                                                                                        |
| `dotenv`        | `[]string`                         |                            | 要解析的 `.env` 文件路径列表。                                                                                                                         |
| `silent`        | `bool`                             | `false`                    | 从输出中隐藏 task 名称和命令。 命令的输出仍将重定向到 `STDOUT` 和 `STDERR`。 当与 `--list` 标志结合使用时，task 描述将被隐藏。                                                        |
| `interactive`   | `bool`                             | `false`                    | 告诉 task 该命令是交互式的。                                                                                                                           |
| `internal`      | `bool`                             | `false`                    | 停止在命令行上调用 task。 当与 `--list` 一起使用时，它也会从输出中省略。                                                                                                |
| `method`        | `string`                           | `checksum`                 | 定义用于检查 task 是最新的方法。 `timestamp` 将比较 sources 的时间戳并生成文件。 `checksum` 将检查 checksum（您可能想忽略 .gitignore 文件中的 .task 文件夹）。 `none` 跳过任何验证并始终运行 task。  |
| `prefix`        | `string`                           |                            | 定义一个字符串作为并行运行 task 输出的前缀。 仅在输出模式是 `prefixed` 时使用。                                                                                           |
| `ignore_error`  | `bool`                             | `false`                    | 如果执行命令时发生错误，则继续执行。                                                                                                                          |
| `run`           | `string`                           | Taskfile 中全局声明的值或 `always` | 指定如果多次调用该 task 是否应再次运行。 可用选项：`always`、`once` 和 `when_changed`。                                                                              |
| `platforms`     | `[]string`                         | 所有平台                       | 指定应在哪些平台上运行 task。 允许使用 [有效的 GOOS 和 GOARCH 值](https://github.com/golang/go/blob/main/src/go/build/syslist.go)。 否则将跳过 task。                   |
| `set`           | `[]string`                         |                            | 为 [内置 `set`](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html) 指定选项。                                                 |
| `shopt`         | `[]string`                         |                            | 为 [内置 `shopt`](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html) 指定选项。                                             |

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

| 属性             | 类型                                 | 默认      | 描述                                                                                                                   |
| -------------- | ---------------------------------- | ------- | -------------------------------------------------------------------------------------------------------------------- |
| `cmd`          | `string`                           |         | 要执行的 shell 命令                                                                                                        |
| `task`         | `string`                           |         | 执行另一个 task，而不执行命令。 不能与 `cmd` 同时设置。                                                                                   |
| `for`          | [`For`](#for)                      |         | Runs the command once for each given value.                                                                          |
| `silent`       | `bool`                             | `false` | 跳过此命令的一些输出。 请注意，命令的 STDOUT 和 STDERR 仍将被重定向。                                                                          |
| `vars`         | [`map[string]Variable`](#variable) |         | 要传递给引用 task 的可选附加变量。 仅在设置 `task` 而不是 `cmd` 时相关。                                                                      |
| `ignore_error` | `bool`                             | `false` | 执行命令的时候忽略错误，继续执行                                                                                                     |
| `defer`        | `string`                           |         | `cmd` 的替代方法，但安排命令在此 task 结束时执行，而不是立即执行。 不能与 `cmd` 一同使用。                                                              |
| `platforms`    | `[]string`                         | 所有平台    | 指定应在哪些平台上运行该命令。 允许使用 [有效的 GOOS 和 GOARCH 值](https://github.com/golang/go/blob/main/src/go/build/syslist.go)。 否则将跳过命令。 |
| `set`          | `[]string`                         |         | 为 [内置 `set`](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html) 指定选项。                          |
| `shopt`        | `[]string`                         |         | 为 [内置 `shopt`](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html) 指定选项。                      |

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

| 属性       | 类型                                 | 默认      | 描述                                                  |
| -------- | ---------------------------------- | ------- | --------------------------------------------------- |
| `task`   | `string`                           |         | 要作为依赖项执行的 task。                                     |
| `vars`   | [`map[string]Variable`](#variable) |         | 要传递给此 task 的可选附加变量。                                 |
| `silent` | `bool`                             | `false` | 从输出中隐藏 task 名称和命令。 命令的输出仍将重定向到 `STDOUT` 和 `STDERR`。 |

:::tip

如果你不想设置额外的变量，将依赖关系声明为一个字符串列表就足够了（它们将被分配给 `task`）。

```yaml
tasks:
  foo:
    deps: [foo, bar]
```

:::

#### For

The `for` parameter can be defined as a string, a list of strings or a map. If it is defined as a string, you can give it any of the following values:

- `source` - Will run the command for each source file defined on the task. (Glob patterns will be resolved, so `*.go` will run for every Go file that matches).

If it is defined as a list of strings, the command will be run for each value.

Finally, the `for` parameter can be defined as a map when you want to use a variable to define the values to loop over:

| 属性      | 类型       | 默认               | 描述                                           |
| ------- | -------- | ---------------- | -------------------------------------------- |
| `var`   | `string` |                  | The name of the variable to use as an input. |
| `split` | `string` | (any whitespace) | What string the variable should be split on. |
| `as`    | `string` | `ITEM`           | The name of the iterator variable.           |

#### Precondition

| Attribute | Type     | Default | Description                             |
| --------- | -------- | ------- | --------------------------------------- |
| `sh`      | `string` |         | 要执行的命令。 如果返回非零退出码， task 将在不执行其命令的情况下出错。 |
| `msg`     | `string` |         | 如果不满足先决条件，则打印可选消息。                      |

:::tip

如果你不想设置不同的消息，你可以像这样声明一个前提条件，值将被分配给 `sh`：

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
