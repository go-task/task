---
slug: /taskfile-versions/
sidebar_position: 5
---

# Taskfile 版本

Taskfile 语法和功能随着时间的推移而改变。 本文档解释了每个版本的变化以及如何升级您的任务文件。

## Taskfile 版本的含义

Taskfile 版本遵循 Task 版本。 例如， taskfile version `2` 意味着应该切换为 Task `v2.0.0` 以支持它。

Taskfile 文件的 `version:` 关键字接受语义化字符串， 所以 `2`, `2.0` 或 `2.0.0` 都可以。 如果使用版本号 `2.0`，那么 Task 就不会使用 `2.1` 的功能， 但如果使用版本号 `2`, 那么任意 `2.x.x` 版本中的功能都是可用的， 但 `3.0.0+` 的功能不可用。

## Version 3 ![latest](https://img.shields.io/badge/latest-brightgreen)

以下是 `v3` 所做的一些主要变更：

- Task 的日志使用彩色输出
- 支持类 `.env` 文件
- 添加 `label:` 设置后可以覆盖任务名称在日志中的显示方式
- 添加了全局 `method:` 允许设置默认方法，Task 的默认值更改为 `checksum`
- Two magic variables were added when using `status:`: `CHECKSUM` and `TIMESTAMP` which contains, respectively, the XXH3 checksum and greatest modification timestamp of the files listed on `sources:`
- 另外，`TASK` 变量总是可以使用当前的任务名称
- CLI 变量始终被视为全局变量
- 向 `includes` 添加了 `dir:` 选项，以允许选择包含的任务文件将在哪个目录上运行：

```yaml
includes:
  docs:
    taskfile: ./docs
    dir: ./docs
```

- 实现短任务语法。 以下所有语法都是等效的：

```yaml
version: '3'

tasks:
  print:
    cmds:
      - echo "Hello, World!"
```

```yaml
version: '3'

tasks:
  print:
    - echo "Hello, World!"
```

```yaml
version: '3'

tasks:
  print: echo "Hello, World!"
```

- 对变量的处理方式进行了重大重构。 现在它们更容易理解了。 `expansions:` 设置被移除了，因为它变得不必要。 这是 Task 处理变量的顺序，每一层都可以看到前一层设置的变量并覆盖这些变量。
  - 环境变量
  - 全局或 CLI 变量
  - 调用变量
  - Task 变量

## 版本 2.6

:::caution

v2 schema support is [deprecated][deprecate-version-2-schema] and will be removed in a future release.

:::

2.6 版本增加任务的先决条件字段 `preconditions`。

```yaml
version: '2'

tasks:
  upload_environment:
    preconditions:
      - test -f .env
    cmds:
      - aws s3 cp .env s3://myenvironment
```

请检查 [文档][includes]

## 版本 2.2

:::caution

v2 schema support is [deprecated][deprecate-version-2-schema] and will be removed in a future release.

:::

2.2 版带有全局 `includes` 选项来包含其他 Taskfiles：

```yaml
version: '2'

includes:
  docs: ./documentation # will look for ./documentation/Taskfile.yml
  docker: ./DockerTasks.yml
```

## 版本 2.1

:::caution

v2 schema support is [deprecated][deprecate-version-2-schema] and will be removed in a future release.

:::

2.1 版包括一个全局 `output` 选项，以允许更好地控制如何将命令输出打印到控制台（有关更多信息，请参阅 [文档][output]）：

```yaml
version: '2'

output: prefixed

tasks:
  server:
    cmds:
      - go run main.go
  prefix: server
```

从这个版本开始，也可以忽略命令或 task 的错误（在 [此处][ignore_errors] 查看文档）：

```yaml
version: '2'

tasks:
  example-1:
    cmds:
      - cmd: exit 1
        ignore_error: true
      - echo "This will be print"

  example-2:
    cmds:
      - exit 1
      - echo "This will be print"
    ignore_error: true
```

## 版本 2.0

:::caution

v2 schema support is [deprecated][deprecate-version-2-schema] and will be removed in a future release.

:::

到了 2.0 版本，我们引入了 `version:` 字段， 在不破坏已存在的 Taskfiles 的前提下，在 Task 中引入新功能。 新语法如下：

```yaml
version: '2'

tasks:
  echo:
    cmds:
      - echo "Hello, World!"
```

如果您不想创建 `Taskvars.yml`，版本 2 允许您直接在 Taskfile 中写入全局变量：

```yaml
version: '2'

vars:
  GREETING: Hello, World!

tasks:
  greet:
    cmds:
      - echo "{{.GREETING}}"
```

变量的优先级调整为：

1. Task 变量
2. Call variables
3. Taskfile 定义变量
4. Taskvars 文件定义变量
5. Environment variables

添加了一个新的全局配置项来配置变量扩展的数量（默认为 2）：

```yaml
version: '2'

expansions: 3

vars:
  FOO: foo
  BAR: bar
  BAZ: baz
  FOOBAR: '{{.FOO}}{{.BAR}}'
  FOOBARBAZ: '{{.FOOBAR}}{{.BAZ}}'

tasks:
  default:
    cmds:
      - echo "{{.FOOBARBAZ}}"
```

## 版本 1

:::caution

v1 schema support was removed in Task >= v3.0.0.

:::

最初的 `Taskfile` 并不支持 `version:` 关键字，因为 YAML 文档中的根属性都是 task。 就像这样：

```yaml
echo:
  cmds:
    - echo "Hello, World!"
```

变量的优先级也不同：

1. 调用变量
2. 环境变量
3. Task 变量
4. `Taskvars.yml` 定义变量

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[output]: usage.md#输出语法
[ignore_errors]: usage.md#忽略错误
[includes]: usage.md#包含其他-taskfile
[deprecate-version-2-schema]: https://github.com/go-task/task/issues/1197
