---
slug: /usage/
sidebar_position: 3
---

# 使用指南

## 快速入门

在项目的根目录中创建一个名为 `Taskfile.yml` 的文件。 `cmds` 属性应包含 task 的命令。 下面的示例允许编译 Go 应用程序并使用 [esbuild](https://esbuild.github.io/) 将多个 CSS 文件合并并缩小为一个文件。

```yaml
version: '3'

tasks:
  build:
    cmds:
      - go build -v -i main.go

  assets:
    cmds:
      - esbuild --bundle --minify css/index.css > public/bundle.css
```

运行 task 就这样简单：

```bash
task assets build
```

Task 使用 [mvdan.cc/sh](https://mvdan.cc/sh/)，一个原生的 Go sh 解释器。 因此，您可以编写 sh/bash 命令，它甚至可以在 Windows 上运行，而 `sh` 或 `bash` 通常不可用。 请记住，任何被调用的可执行文件都必须在操作系统或 PATH 中可用。

如果不传 task 的名字，默认会调用 "default"。

## 支持的文件名称

Task 会按以下顺序查找配置文件:

- Taskfile.yml
- taskfile.yml
- Taskfile.yaml
- taskfile.yaml
- Taskfile.dist.yml
- taskfile.dist.yml
- Taskfile.dist.yaml
- taskfile.dist.yaml

使用 `.dist` 变体的目的是允许项目有一个提交版本 (`.dist`)，同时仍然允许个人用户通过添加额外的 `Taskfile.yml`（将在 `.gitignore` 上）来覆盖 Taskfile。

### 从子目录运行 Taskfile

如果在当前工作目录中找不到 Taskfile，它将沿着文件树向上查找，直到找到一个（类似于 `git` 的工作方式）。 当从这样的子目录运行 Task 时，它的行为就像从包含 Taskfile 的目录运行它一样。

您可以将此功能与特殊的 `{{.USER_WORKING_DIR}}` 变量一起使用来创建一些非常有用的可重用 task。 例如，如果你有一个包含每个微服务目录的 monorepo，你可以 `cd` 进入一个微服务目录并运行一个 task 命令来启动它，而不必创建多个 task 或具有相同内容的 Taskfile。 例如：

```yaml
version: '3'

tasks:
  up:
    dir: '{{.USER_WORKING_DIR}}'
    preconditions:
      - test -f docker-compose.yml
    cmds:
      - docker-compose up -d
```

在此示例中，我们可以运行 `cd <service>` 和 `task up`，只要 `<service>` 目录包含 `docker-compose.yml`，就会启动 Docker Compose。

### 运行全局 Taskfile

如果您使用 `--global`（别名 `-g`）标志调用 Task，它将查找您的 home 目录而不是您的工作目录。 简而言之，task 将寻找匹配 `$HOME/{T,t}askfile.{yml,yaml}` 的 配置文件。

这对于您可以在系统的任何地方运行的自动化很有用！

:::info

当使用 `-g` 运行全局 Taskfile 时，task 将默认在 `$HOME` 上运行，而不是在您的工作目录上！

如前一节所述，`{{.USER_WORKING_DIR}}` 特殊变量在这里可以非常方便地在您从中调用 `task -g` 的目录中运行内容。

```yaml
version: '3'

tasks:
  from-home:
    cmds:
      - pwd

  from-working-directory:
    dir: '{{.USER_WORKING_DIR}}'
    cmds:
      - pwd
```

:::

## 环境变量

### Task

你可以使用 `env` 给每个 task 设置自定义环境变量:

```yaml
version: '3'

tasks:
  greet:
    cmds:
      - echo $GREETING
    env:
      GREETING: Hey, there!
```

此外，您可以设置可用于所有 task 的全局环境变量：

```yaml
version: '3'

env:
  GREETING: Hey, there!

tasks:
  greet:
    cmds:
      - echo $GREETING
```

:::info

`env` 支持扩展和检索 shell 命令的输出，就像变量一样，如您在 [变量](#变量) 部分中看到的那样。

:::

### .env 文件

您还可以使用 `dotenv:` 设置要求 tasks 包含 `.env` 之类的文件

```bash title=".env"
KEYNAME=VALUE
```

```bash title="testing/.env"
ENDPOINT=testing.com
```

```yaml title="Taskfile.yml"
version: '3'

env:
  ENV: testing

dotenv: ['.env', '{{.ENV}}/.env.', '{{.HOME}}/.env']

tasks:
  greet:
    cmds:
      - echo "Using $KEYNAME and endpoint $ENDPOINT"
```

也可以在 task 级别指定 .env 文件：

```yaml
version: '3'

env:
  ENV: testing

tasks:
  greet:
    dotenv: ['.env', '{{.ENV}}/.env.', '{{.HOME}}/.env']
    cmds:
      - echo "Using $KEYNAME and endpoint $ENDPOINT"
```

在 task 级别明确指定的环境变量将覆盖点文件中定义的变量：

```yaml
version: '3'

env:
  ENV: testing

tasks:
  greet:
    dotenv: ['.env', '{{.ENV}}/.env.', '{{.HOME}}/.env']
    env:
      KEYNAME: DIFFERENT_VALUE
    cmds:
      - echo "Using $KEYNAME and endpoint $ENDPOINT"
```

:::info

请注意，您目前无法在包含的 Taskfile 中使用 `dotenv` 键。

:::

## 包含其他 Taskfile

如果要在不同项目（Taskfile）之间共享任务，可以使用导入机制使用 `includes` 关键字包含其他任务文件：

```yaml
version: '3'

includes:
  docs: ./documentation # will look for ./documentation/Taskfile.yml
  docker: ./DockerTasks.yml
```

给定的 Taskfile 中描述的任务将在指定的命名空间中提供。 因此，您可以调用 `task docs:serve` 从 `documentation/Taskfile.yml` 运行 `serve` task，或者调用 `task docker:build` 从 `DockerTasks.yml` 文件运行 `build` task。

相对路径是相对于包含包含 Taskfile 的目录解析的。

### 操作系统特定 Taskfile

在 `version: '2'` 中，task 会自动尝试引入 `Taskfile_{{OS}}.yml` 文件 （例如`Taskfile_windows.yml`, `Taskfile_linux.yml` 或 `Taskfile_darwin.yml`）。 但是因为过于隐晦，在版本 3 中被移除了， 在版本 3 可以通过明确的引用来实现类似功能:

```yaml
version: '3'

includes:
  build: ./Taskfile_{{OS}}.yml
```

### 包含 Taskfile 的目录

默认情况下，包含的 Taskfile 的 task 在当前目录中运行，即使 Taskfile 在另一个目录中，但您可以使用以下替代语法强制其 task 在另一个目录中运行：

```yaml
version: '3'

includes:
  docs:
    taskfile: ./docs/Taskfile.yml
    dir: ./docs
```

:::info

包含的 Taskfile 必须使用与主 Taskfile 使用的相同规则版本。

:::

### 可选 includes

如果包含文件丢失，标记为可选的包含将允许 task 继续正常执行。

```yaml
version: '3'

includes:
  tests:
    taskfile: ./tests/Taskfile.yml
    optional: true

tasks:
  greet:
    cmds:
      - echo "This command can still be successfully executed if
        ./tests/Taskfile.yml does not exist"
```

### 内部 includes

标记为 internal 的包含会将包含文件的所有 task 也设置为内部 task（请参阅下面的 [内部-tasks](#内部-tasks) 部分）。 这在包含不打算由用户直接使用的实用程序任务时很有用。

```yaml
version: '3'

includes:
  tests:
    taskfile: ./taskfiles/Utils.yml
    internal: true
```

### 包含 Taskfile 的变量

您还可以在包含 Taskfile 时指定变量。 这对于拥有可以调整甚至多次包含的可重用 Taskfile 可能很有用：

```yaml
version: '3'

includes:
  backend:
    taskfile: ./taskfiles/Docker.yml
    vars:
      DOCKER_IMAGE: backend_image

  frontend:
    taskfile: ./taskfiles/Docker.yml
    vars:
      DOCKER_IMAGE: frontend_image
```

### 命名空间别名

包含 Taskfile 时，您可以为命名空间提供一个 `aliases` 列表。 这与 [task 别名](#task-别名) 的工作方式相同，可以一起使用来创建更短且更易于键入的命令。

```yaml
version: '3'

includes:
  generate:
    taskfile: ./taskfiles/Generate.yml
    aliases: [gen]
```

:::info

在包含的 Taskfile 中声明的变量优先于包含 Taskfile 中的变量！ 如果您希望包含的 Taskfile 中的变量可被覆盖，请使用 [默认方法](https://go-task.github.io/slim-sprig/defaults.html)：`MY_VAR: '{{.MY_VAR | default "my-default-value"}}'`。

:::

## 内部 tasks

内部 task 是用户不能直接调用的 task。 运行 `task --list|--list-all` 时，它们不会出现在输出中。 其他 task 可以照常调用内部 task。 这对于创建在命令行上没有用处的可重用、类似函数的 task 很有用。

```yaml
version: '3'

tasks:
  build-image-1:
    cmds:
      - task: build-image
        vars:
          DOCKER_IMAGE: image-1

  build-image:
    internal: true
    cmds:
      - docker build -t {{.DOCKER_IMAGE}} .
```

## Task 目录

默认情况下，tasks 将在 Taskfile 所在的目录中执行。 但是您可以轻松地让 task 在另一个目录中运行，指定 `dir`：

```yaml
version: '3'

tasks:
  serve:
    dir: public/www
    cmds:
      # run http server
      - caddy
```

如果该目录不存在，`task` 会创建它。

## Task 依赖

> 依赖项并行运行，因此一项 task 的依赖项不应相互依赖。 如果您想强制任务顺序运行，请查看下面的 [调用另一个 task](#调用另一个-task) 部分。

您可能有依赖于其它的 task。 将它们指向 `deps` 将使它们在运行父 task 之前自动运行：

```yaml
version: '3'

tasks:
  build:
    deps: [assets]
    cmds:
      - go build -v -i main.go

  assets:
    cmds:
      - esbuild --bundle --minify css/index.css > public/bundle.css
```

在上面的示例中，如果您运行 `task build`，`assets` 将始终在 `build` 之前运行。

一个 task 只能有依赖关系，没有命令来将 task 组合在一起：

```yaml
version: '3'

tasks:
  assets:
    deps: [js, css]

  js:
    cmds:
      - esbuild --bundle --minify js/index.js > public/bundle.js

  css:
    cmds:
      - esbuild --bundle --minify css/index.css > public/bundle.css
```

如果有多个依赖项，它们总是并行运行以获得更好的性能。

:::tip

您还可以使用 `--parallel` 标志（别名 `-p`）使命令行给出的 task 并行运行。 例如: `task --parallel js css`。

:::

如果你想将信息传递给依赖项，你可以像 [调用另一个 task](#调用另一个-task) 一样以相同的方式进行：

```yaml
version: '3'

tasks:
  default:
    deps:
      - task: echo_sth
        vars: { TEXT: 'before 1' }
      - task: echo_sth
        vars: { TEXT: 'before 2' }
        silent: true
    cmds:
      - echo "after"

  echo_sth:
    cmds:
      - echo {{.TEXT}}
```

## 平台特定的 tasks 和 cmds

如果您想将 task 的运行限制在明确的平台上，可以使用 `platforms:` 键来实现。 Task 可以限制在特定的操作系统、架构或两者的组合中。 如果不匹配，任务或命令将被跳过，并且不会抛出任何错误。

允许作为 OS 或 Arch 的值是有效的 `GOOS` 和 `GOARCH` 值，正如 [此处](https://github.com/golang/go/blob/master/src/go/build/syslist.go) 的 Go 语言所定义的那样。

下面的 `build-windows` task 将仅在 Windows 所有架构上运行：

```yaml
version: '3'

tasks:
  build-windows:
    platforms: [windows]
    cmds:
      - echo 'Running command on Windows'
```

这可以限制为特定的架构，如下所示：

```yaml
version: '3'

tasks:
  build-windows-amd64:
    platforms: [windows/amd64]
    cmds:
      - echo 'Running command on Windows (amd64)'
```

也可以将 task 限制在特定的架构中：

```yaml
version: '3'

tasks:
  build-amd64:
    platforms: [amd64]
    cmds:
      - echo 'Running command on amd64'
```

可以指定多个平台，如下所示：

```yaml
version: '3'

tasks:
  build:
    platforms: [windows/amd64, darwin]
    cmds:
      - echo 'Running command on Windows (amd64) and macOS'
```

个别命令也可以限制在特定平台上：

```yaml
version: '3'

tasks:
  build:
    cmds:
      - cmd: echo 'Running command on Windows (amd64) and macOS'
        platforms: [windows/amd64, darwin]
      - cmd: echo 'Running on all platforms'
```

## 调用另一个 task

当一个 task 有很多依赖时，它们是并发执行的。 这通常会导致更快的构建管道。 但是，在某些情况下，您可能需要串行调用其他 task。 在这种情况下，请使用以下语法：

```yaml
version: '3'

tasks:
  main-task:
    cmds:
      - task: task-to-be-called
      - task: another-task
      - echo "Both done"

  task-to-be-called:
    cmds:
      - echo "Task to be called"

  another-task:
    cmds:
      - echo "Another task"
```

使用 `vars` 和 `silent` 属性，您可以选择在逐个调用的基础上传递变量和切换 [静默模式](#静默模式)：

```yaml
version: '3'

tasks:
  greet:
    vars:
      RECIPIENT: '{{default "World" .RECIPIENT}}'
    cmds:
      - echo "Hello, {{.RECIPIENT}}!"

  greet-pessimistically:
    cmds:
      - task: greet
        vars: { RECIPIENT: 'Cruel World' }
        silent: true
```

`deps` 也支持上述语法。

:::tip

注意：如果您想从 [包含的 Taskfile](#包含其他-taskfile) 中调用在根 Taskfile 中声明的 task，请像这样添加 `:` 前缀：`task: :task-name`。

:::

## 减少不必要的工作

### 通过指纹识别本地生成的文件及其来源

如果一个 task 生成了一些东西，你可以通知 task 源和生成的文件，这样 task 就会在不需要的时候阻止运行它们。

```yaml
version: '3'

tasks:
  build:
    deps: [js, css]
    cmds:
      - go build -v -i main.go

  js:
    cmds:
      - esbuild --bundle --minify js/index.js > public/bundle.js
    sources:
      - src/js/**/*.js
    generates:
      - public/bundle.js

  css:
    cmds:
      - esbuild --bundle --minify css/index.css > public/bundle.css
    sources:
      - src/css/**/*.css
    generates:
      - public/bundle.css
```

`sources` 和 `generates` 可以配置具体文件或者使用匹配模式。 设置后， Task 会根据源文件的 checksum 来确定是否需要执行当前任务。 如果不需要执行， 则会输出像 `Task "js" is up to date` 这样的信息。

如果您希望通过文件的修改 timestamp 而不是其 checksum（内容）来进行此检查，只需将 `method` 属性设置为 `timestamp` 即可。

```yaml
version: '3'

tasks:
  build:
    cmds:
      - go build .
    sources:
      - ./*.go
    generates:
      - app{{exeExt}}
    method: timestamp
```

在需要更大灵活性的情况下，可以使用 `status` 关键字。 您甚至可以将两者结合起来。 有关示例，请参阅 [状态](#使用程序检查来表示任务是最新的) 文档。

:::info

默认情况，task 在本地项目的 `.task` 目录保存 checksums 值。 一般都会在 `.gitignore`（或类似配置）中忽略掉这个目录，这样它就不会被提交。 (如果您有一个已提交的代码生成任务，那么提交该任务的校验和也是有意义的)。

如果你想要将这些文件存储在另一个目录中，你可以在你的机器中设置一个 `TASK_TEMP_DIR` 环境变量。 可以使用相对路径，比如 `tmp/task`，相对项目根目录，也可以用绝对路径、用户目录路径，比如 `/tmp/.task` 或 `~/.task`（每个项目单独创建子目录）。

```bash
export TASK_TEMP_DIR='~/.task'
```

:::

:::info

每个 task 只为其 `sources` 存储一个 checksum。 如果您想通过任何输入变量来区分 task，您可以将这些变量添加为 task 标签的一部分，它将被视为不同的 task。

如果您想为每个不同的输入集运行一次 task，直到 sources 实际发生变化，这将很有用。 例如，如果 sources 依赖于变量的值，或者您希望在某些参数发生变化时重新运行 task，即使 sources 没有发生变化也是如此。

:::

:::tip

将 method 设置为 `none` 会跳过任何验证并始终运行任务。

:::

:::info

要使 `checksum`（默认）或 `timestamp` 方法起作用，只需要通知 source 文件即可。 当使用 `timestamp` 方法时，最后一次运行 task 被认为是一次生成。

:::

### 使用程序检查来表示任务是最新的

或者，您可以通知一系列测试作为 `status`。 如果没有错误返回（退出状态 0），task 被认为是最新的：

```yaml
version: '3'

tasks:
  generate-files:
    cmds:
      - mkdir directory
      - touch directory/file1.txt
      - touch directory/file2.txt
    # test existence of files
    status:
      - test -d directory
      - test -f directory/file1.txt
      - test -f directory/file2.txt
```

通常，您会将 `sources` 与 `generates` 结合使用 - 但对于生成远程工件（Docker 映像、部署、CD 版本）的 task，checksum source 和 timestamps 需要访问工件或 `.checksum` 指纹文件。

两个特殊变量 `{{.CHECKSUM}}` 和 `{{.TIMESTAMP}}` 可用于 `status` 命令中的插值，具体取决于分配给 sources 的指纹方法。 只有 `source` 块才能生成指纹。

请注意，`{{.TIMESTAMP}}` 变量是一个“实时”Go `time.Time` 结构，可以使用 `time.Time` 响应的任何方法进行格式化。

有关详细信息，请参阅 [Go Time 文档](https://golang.org/pkg/time/)。

如果你想强制任务运行，即使是最新的，你也可以使用 `--force` 或 `-f`。

此外，如果任何 task 不是最新的，`task --status [tasks]...` 将以非零退出代码退出。

如果 source/generated 的工件发生变化，或者程序检查失败，`status` 可以与 [指纹](#通过指纹识别本地生成的文件及其来源) 结合以运行任务：

```yaml
version: '3'

tasks:
  build:prod:
    desc: Build for production usage.
    cmds:
      - composer install
    # Run this task if source files changes.
    sources:
      - composer.json
      - composer.lock
    generates:
      - ./vendor/composer/installed.json
      - ./vendor/autoload.php
    # But also run the task if the last build was not a production build.
    status:
      - grep -q '"dev": false' ./vendor/composer/installed.json
```

### 使用程序检查取消任务及其依赖项的执行

除了 `status` 检查之外，`preconditions` 检查是 `status` 检查的逻辑逆过程。 也就是说，如果您需要一组特定的条件为 _true_，您可以使用 `preconditions`。 `preconditions` 类似于 `status` 行，除了它们支持 `sh` 扩展，并且它们应该全部返回 0。

```yaml
version: '3'

tasks:
  generate-files:
    cmds:
      - mkdir directory
      - touch directory/file1.txt
      - touch directory/file2.txt
    # test existence of files
    preconditions:
      - test -f .env
      - sh: '[ 1 = 0 ]'
        msg: "One doesn't equal Zero, Halting"
```

先决条件可以设置特定的失败消息，这些消息可以使用 `msg` 字段告诉用户要采取什么步骤。

如果一个 task 依赖于一个具有前提条件的子 task，并且不满足该前提条件 - 调用 task 将失败。 请注意，除非给出 `--force` ，否则以失败的前提条件执行的 task 将不会运行。

与 `status` 判断 task 是最新状态时会跳过并继续执行不同， `precondition` 失败会导致 task 失败，以及所有依赖它的 task 都会失败。

```yaml
version: '3'

tasks:
  task-will-fail:
    preconditions:
      - sh: 'exit 1'

  task-will-also-fail:
    deps:
      - task-will-fail

  task-will-still-fail:
    cmds:
      - task: task-will-fail
      - echo "I will not run"
```

### 在任务运行时限制

如果 task 由多个 `cmd` 或多个 `deps` 执行，您可以使用 `run` 控制何时执行。 `run` 也可以设置在 Taskfile 的根目录以更改所有任务的行为，除非被明确覆盖。

`run` 支持的值：

- `always` （默认）总是尝试调用 task，无论先前执行的次数如何
- `once` 只调用一次这个任务，不管引用的数量
- `when_changed` 只为传递给 task 的每个唯一变量集调用一次 task

```yaml
version: '3'

tasks:
  default:
    cmds:
      - task: generate-file
        vars: { CONTENT: '1' }
      - task: generate-file
        vars: { CONTENT: '2' }
      - task: generate-file
        vars: { CONTENT: '2' }

  generate-file:
    run: when_changed
    deps:
      - install-deps
    cmds:
      - echo {{.CONTENT}}

  install-deps:
    run: once
    cmds:
      - sleep 5 # long operation like installing packages
```

### 确保设置所需变量

如果想要在运行任务之前检查是否设置了某些变量，那么 您可以使用 `requires`。 这可以显示一个明确的消息，帮助用户了解哪些变量是必需的。 比如，一些任务如果使用未设置的变量，可能会产生危险的副作用。

Using `requires` you specify an array of strings in the `vars` sub-section under `requires`, these strings are variable names which are checked prior to running the task. If any variables are un-set the the task will error and not run.

Environmental variables are also checked.

Syntax:

```yaml
requires:
  vars: [] # Array of strings
```

:::note

Variables set to empty zero length strings, will pass the `requires` check.

:::

Example of using `requires`:

```yaml
version: '3'

tasks:
  docker-build:
    cmds:
      - 'docker build . -t {{.IMAGE_NAME}}:{{.IMAGE_TAG}}'

    # Make sure these variables are set before running
    requires:
      vars: [IMAGE_NAME, IMAGE_TAG]
```

## 变量

在进行变量插值时，Task 将查找以下内容。 它们按权重顺序列在下面（即最重要的第一位）：

- task 内部定义的变量
- 被其它 task 调用时传入的变量(查看 [调用另一个 task](#调用另一个-task))
- [包含其他 Taskfile](#包含其他-taskfile) 中的变量 (当包含其他 task 时)
- [包含 Taskfile](#包含-taskfile-的变量) 的变量（包含 task 时）
- 全局变量 (在 Taskfile 的 `vars:` 中声明)
- 环境变量

使用环境变量传输参数的示例：

```bash
$ TASK_VARIABLE=a-value task do-something
```

:::tip

包含任务名称的特殊变量 `.TASK` 始终可用。

:::

由于某些 shell 不支持上述语法来设置环境变量 (Windows)，task 在不在命令开头时也接受类似的样式。

```bash
$ task write-file FILE=file.txt "CONTENT=Hello, World!" print "MESSAGE=All done!"
```

本地声明的变量示例：

```yaml
version: '3'

tasks:
  print-var:
    cmds:
      - echo "{{.VAR}}"
    vars:
      VAR: Hello!
```

`Taskfile.yml` 中的全局变量示例：

```yaml
version: '3'

vars:
  GREETING: Hello from Taskfile!

tasks:
  greet:
    cmds:
      - echo "{{.GREETING}}"
```

### 动态变量

以下语法 (`sh:` prop in a variable) 被认为是动态变量。 该值将被视为命令并产生输出结果用于赋值。 如果有一个或多个尾随换行符，最后一个换行符将被修剪。

```yaml
version: '3'

tasks:
  build:
    cmds:
      - go build -ldflags="-X main.Version={{.GIT_COMMIT}}" main.go
    vars:
      GIT_COMMIT:
        sh: git log -n 1 --format=%h
```

这适用于所有类型的变量。

## Looping over values

As of v3.28.0, Task allows you to loop over certain values and execute a command for each. There are a number of ways to do this depending on the type of value you want to loop over.

### Looping over a static list

The simplest kind of loop is an explicit one. This is useful when you want to loop over a set of values that are known ahead of time.

```yaml
version: '3'

tasks:
  default:
    cmds:
      - for: ['foo.txt', 'bar.txt']
        cmd: cat {{ .ITEM }}
```

### Looping over your task's sources

You are also able to loop over the sources of your task:

```yaml
version: '3'

tasks:
  default:
    sources:
      - foo.txt
      - bar.txt
    cmds:
      - for: sources
        cmd: cat {{ .ITEM }}
```

This will also work if you use globbing syntax in your sources. For example, if you specify a source for `*.txt`, the loop will iterate over all files that match that glob.

Source paths will always be returned as paths relative to the task directory. If you need to convert this to an absolute path, you can use the built-in `joinPath` function. There are some [special variables](/api/#special-variables) that you may find useful for this.

```yaml
version: '3'

tasks:
  default:
    vars:
      MY_DIR: /path/to/dir
    dir: '{{.MY_DIR}}'
    sources:
      - foo.txt
      - bar.txt
    cmds:
      - for: sources
        cmd: cat {{joinPath .MY_DIR .ITEM}}
```

### Looping over variables

To loop over the contents of a variable, you simply need to specify the variable you want to loop over. By default, variables will be split on any whitespace characters.

```yaml
version: '3'

tasks:
  default:
    vars:
      MY_VAR: foo.txt bar.txt
    cmds:
      - for: { var: MY_VAR }
        cmd: cat {{.ITEM}}
```

If you need to split on a different character, you can do this by specifying the `split` property:

```yaml
version: '3'

tasks:
  default:
    vars:
      MY_VAR: foo.txt,bar.txt
    cmds:
      - for: { var: MY_VAR, split: ',' }
        cmd: cat {{.ITEM}}
```

All of this also works with dynamic variables!

```yaml
version: '3'

tasks:
  default:
    vars:
      MY_VAR:
        sh: find -type f -name '*.txt'
    cmds:
      - for: { var: MY_VAR }
        cmd: cat {{.ITEM}}
```

### Renaming variables

If you want to rename the iterator variable to make it clearer what the value contains, you can do so by specifying the `as` property:

```yaml
version: '3'

tasks:
  default:
    vars:
      MY_VAR: foo.txt bar.txt
    cmds:
      - for: { var: MY_VAR, as: FILE }
        cmd: cat {{.FILE}}
```

### Looping over tasks

Because the `for` property is defined at the `cmds` level, you can also use it alongside the `task` keyword to run tasks multiple times with different variables.

```yaml
version: '3'

tasks:
  default:
    cmds:
      - for: [foo, bar]
        task: my-task
        vars:
          FILE: '{{.ITEM}}'

  my-task:
    cmds:
      - echo '{{.FILE}}'
```

Or if you want to run different tasks depending on the value of the loop:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - for: [foo, bar]
        task: task-{{.ITEM}}

  task-foo:
    cmds:
      - echo 'foo'

  task-bar:
    cmds:
      - echo 'bar'
```

## 将 CLI 参数转发到 cmds

如果 `--` 在 CLI 中给出，则所有以下参数都将添加到特殊的 `.CLI_ARGS` 变量中。 这对于将参数转发给另一个命令很有用。

下面的示例将运行 `yarn install`。

```bash
$ task yarn -- install
```

```yaml
version: '3'

tasks:
  yarn:
    cmds:
      - yarn {{.CLI_ARGS}}
```

## 使用 `defer` 做 task 清理

使用 `defer` 关键字，可以安排在 task 完成后运行清理。 与仅将其作为最后一个命令的不同之处在于，即使 task 失败，该命令也会运行。

在下面的示例中，即使第三个命令失败，`rm -rf tmpdir/` 也会运行：

```yaml
version: '3'

tasks:
  default:
    cmds:
      - mkdir -p tmpdir/
      - defer: rm -rf tmpdir/
      - echo 'Do work on tmpdir/'
```

使用其它 task 作为清理任务的命令时，可以这样：

```yaml
version: '3'

tasks:
  default:
    cmds:
      - mkdir -p tmpdir/
      - defer: { task: cleanup }
      - echo 'Do work on tmpdir/'

  cleanup: rm -rf tmpdir/
```

:::info

由于 [Go 自身的 `defer` 工作方式](https://go.dev/tour/flowcontrol/13) 的性质，如果您安排多个 defer命令，则 defer 命令将以相反的顺序执行。

:::

## Go 的模板引擎

Task 在执行命令之前将命令解析为 [Go 的模板引擎][gotemplate]。 可以通过点语法 (`.VARNAME`) 访问变量。

Go 的 [slim-sprig 库](https://go-task.github.io/slim-sprig/) 的所有功能都可用。 以下示例按照给定格式获取当前日期：

```yaml
version: '3'

tasks:
  print-date:
    cmds:
      - echo {{now | date "2006-01-02"}}
```

Task 还增加了以下功能：

- `OS`：返回操作系统。 Possible values are `windows`, `linux`, `darwin` (macOS) and `freebsd`.
- `ARCH`: return the architecture Task was compiled to: `386`, `amd64`, `arm` or `s390x`.
- `splitLines`: Splits Unix (`\n`) and Windows (`\r\n`) styled newlines.
- `catLines`: Replaces Unix (`\n`) and Windows (`\r\n`) styled newlines with a space.
- `toSlash`：在 Unix 上不执行任何操作，但在 Windows 上将字符串从 `\` 路径格式转换为 `/`。
- `fromSlash`：与 `toSlash` 相反。 在 Unix 上不执行任何操作，但在 Windows 上将字符串从 `/` 路径格式转换为 `\`。
- `exeExt`：返回当前操作系统的正确可执行文件扩展名（Windows 为`“.exe”`，其他操作系统为`“”`）。
- `shellQuote`：引用一个字符串以使其在 shell 脚本中安全使用。 Task 为此使用了 [这个 Go 函数](https://pkg.go.dev/mvdan.cc/sh/v3@v3.4.0/syntax#Quote)。 假定使用 Bash 语法。
- `splitArgs`：将字符串作为命令的参数进行拆分。 Task 使用 [这个 Go 函数](https://pkg.go.dev/mvdan.cc/sh/v3@v3.4.0/shell#Fields)

示例：

```yaml
version: '3'

tasks:
  print-os:
    cmds:
      - echo '{{OS}} {{ARCH}}'
      - echo '{{if eq OS "windows"}}windows-command{{else}}unix-command{{end}}'
      # This will be path/to/file on Unix but path\to\file on Windows
      - echo '{{fromSlash "path/to/file"}}'
  enumerated-file:
    vars:
      CONTENT: |
        foo
        bar
    cmds:
      - |
        cat << EOF > output.txt
        {{range $i, $line := .CONTENT | splitLines -}}
        {{printf "%3d" $i}}: {{$line}}
        {{end}}EOF
```

## 帮助

运行 `task --list`（或 `task -l`）列出所有带有描述的任务。 以下 Taskfile：

```yaml
version: '3'

tasks:
  build:
    desc: Build the go binary.
    cmds:
      - go build -v -i main.go

  test:
    desc: Run all the go tests.
    cmds:
      - go test -race ./...

  js:
    cmds:
      - esbuild --bundle --minify js/index.js > public/bundle.js

  css:
    cmds:
      - esbuild --bundle --minify css/index.css > public/bundle.css
```

将打印以下输出：

```bash
* build:   Build the go binary.
* test:    Run all the go tests.
```

如果您想查看所有任务，还有一个 `--list-all`（别名 `-a`）标志。

## 显示任务摘要

运行 `task --summary task-name` 将显示任务的摘要。 以下 Taskfile：

```yaml
version: '3'

tasks:
  release:
    deps: [build]
    summary: |
      Release your project to github

      It will build your project before starting the release.
      Please make sure that you have set GITHUB_TOKEN before starting.
    cmds:
      - your-release-tool

  build:
    cmds:
      - your-build-tool
```

运行 `task --summary release` 将打印以下输出：

```
task: release

Release your project to github

It will build your project before starting the release.
Please make sure that you have set GITHUB_TOKEN before starting.

dependencies:
 - build

commands:
 - your-release-tool
```

如果缺少摘要，将打印描述。 如果任务没有摘要或描述，则会打印一条警告。

请注意：_显示摘要不会执行命令_。

## Task 别名

Aliases 是 task 的替代名称。 它们可以使运行具有长名称或难以键入名称的 task 变得更加容易和快速。 您可以在命令行上使用它们，在您的 Taskfile 中 [调用子任务](#调用另一个-task) 时以及在 [包含来自另一个 Taskfile](#包含其他-taskfile) 的别名 task 时。 它们也可以与 [命名空间别名](#命名空间别名) 一起使用。

```yaml
version: '3'

tasks:
  generate:
    aliases: [gen]
    cmds:
      - task: gen-mocks

  generate-mocks:
    aliases: [gen-mocks]
    cmds:
      - echo "generating..."
```

## 覆盖 Task 名称

有时你可能想覆盖打印在摘要上的 task 名称，最新消息到 STDOUT 等。 在这种情况下，你可以只设置 `label:`，也可以用变量进行插值：

```yaml
version: '3'

tasks:
  default:
    - task: print
      vars:
        MESSAGE: hello
    - task: print
      vars:
        MESSAGE: world

  print:
    label: 'print-{{.MESSAGE}}'
    cmds:
      - echo "{{.MESSAGE}}"
```

## 警告提示

Warning Prompts are used to prompt a user for confirmation before a task is executed.

Below is an example using `prompt` with a dangerous command, that is called between two safe commands:

```yaml
version: '3'

tasks:
  example:
    cmds:
      - task: not-dangerous
      - task: dangerous
      - task: another-not-dangerous

  not-dangerous:
    cmds:
      - echo 'not dangerous command'

  another-not-dangerous:
    cmds:
      - echo 'another not dangerous command'

  dangerous:
    prompt: This is a dangerous command... Do you want to continue?
    cmds:
      - echo 'dangerous command'
```

```bash
❯ task dangerous
task: "This is a dangerous command... Do you want to continue?" [y/N]
```

Warning prompts are called before executing a task. If a prompt is denied Task will exit with [exit code](api_reference.md#exit-codes) 205. If approved, Task will continue as normal.

```bash
❯ task example
not dangerous command
task: "This is a dangerous command. Do you want to continue?" [y/N]
y
dangerous command
another not dangerous command
```

To skip warning prompts automatically, you can use the `--yes` (alias `-y`) option when calling the task. By including this option, all warnings, will be automatically confirmed, and no prompts will be shown.

:::caution

Tasks with prompts always fail by default on non-terminal environments, like a CI, where an `stdin` won't be available for the user to answer. In those cases, use `--yes` (`-y`) to force all tasks with a prompt to run.

:::

## 静默模式

静默模式在 Task 运行命令之前禁用命令回显。 对于以下 Taskfile：

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "Print something"
```

通常这将打印：

```sh
echo "Print something"
Print something
```

开启静默模式后，将打印以下内容：

```sh
Print something
```

开启静默模式有四种方式：

- 在 cmds 级别：

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - cmd: echo "Print something"
        silent: true
```

- 在 task 级别：

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "Print something"
    silent: true
```

- 在 Taskfile 全局级别：

```yaml
version: '3'

silent: true

tasks:
  echo:
    cmds:
      - echo "Print something"
```

- 或者全局使用 `--silent` 或 `-s` 标志

如果您想改为禁止 STDOUT，只需将命令重定向到 `/dev/null`：

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "This will print nothing" > /dev/null
```

## 试运行模式

试运行模式 (`--dry`) 编译并逐步完成每个 task，打印将运行但不执行它们的命令。 这对于调试您的 Taskfile 很有用。

## 忽略错误

您可以选择在命令执行期间忽略错误。 给定以下 Taskfile：

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - exit 1
      - echo "Hello World"
```

Task 将在运行 `exit 1` 后中止执行，因为状态代码 `1` 代表 `EXIT_FAILURE`。 但是，可以使用 `ignore_error` 继续执行：

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - cmd: exit 1
        ignore_error: true
      - echo "Hello World"
```

也可以为 task 设置 `ignore_error`，这意味着所有命令的错误都将被忽略。 不过，请记住，此选项不会传播到由 deps 或 cmds 调用的其他 task！

## 输出语法

默认情况下，Task 只是将正在运行的命令的 STDOUT 和 STDERR 实时重定向到 shell。 这有利于通过命令打印日志记录的实时反馈，但如果同时运行多个命令并打印大量内容，输出可能会变得混乱。

为了使其更具可定制性，目前您可以选择三种不同的输出选项：

- `interleaved` (默认)
- `group`
- `prefixed`

要选择另一个，只需在 Taskfile 根目录中设置即可：

```yaml
version: '3'

output: 'group'

tasks:
  # ...
```

`group` 输出将在命令完成后打印一次命令的全部输出，因此您不会对需要很长时间运行的命令有实时反馈。

使用 `group` 输出时，您可以选择提供模板化消息以在组的开始和结束处打印。 这对于指示 CI 系统对给定任务的所有输出进行分组非常有用，例如使用 [GitHub Actions 的 `::group::` 命令](https://docs.github.com/en/actions/learn-github-actions/workflow-commands-for-github-actions#grouping-log-lines) 或 [Azure Pipelines](https://docs.microsoft.com/en-us/azure/devops/pipelines/scripts/logging-commands?expand=1&view=azure-devops&tabs=bash#formatting-commands)。

```yaml
version: '3'

output:
  group:
    begin: '::group::{{.TASK}}'
    end: '::endgroup::'

tasks:
  default:
    cmds:
      - echo 'Hello, World!'
    silent: true
```

```bash
$ task default
::group::default
Hello, World!
::endgroup::
```

使用 `group` 输出时，如果没有失败（零退出代码），您可以在标准输出和标准错误上执行命令的输出。

```yaml
version: '3'

silent: true

output:
  group:
    error_only: true

tasks:
  passes: echo 'output-of-passes'
  errors: echo 'output-of-errors' && exit 1
```

```bash
$ task passes
$ task errors
output-of-errors
task: Failed to run task "errors": exit status 1
```

`prefix` 输出将为命令打印的每一行添加前缀 `[task-name]` 作为前缀，但您可以使用 `prefix:` 属性自定义命令的前缀：

```yaml
version: '3'

output: prefixed

tasks:
  default:
    deps:
      - task: print
        vars: { TEXT: foo }
      - task: print
        vars: { TEXT: bar }
      - task: print
        vars: { TEXT: baz }

  print:
    cmds:
      - echo "{{.TEXT}}"
    prefix: 'print-{{.TEXT}}'
    silent: true
```

```bash
$ task default
[print-foo] foo
[print-bar] bar
[print-baz] baz
```

:::tip

`output` 选项也可以由 `--output` 或 `-o` 标志指定。

:::

## 交互式 CLI 应用

Task 执行包含交互式的命令时有时会出现奇怪的结果， 尤其当 [输出模式](#输出语法) 设置的不是 `interleaved` （默认）， 或者当交互式应用与其它 task 并发执行时。

`interactive: true` 告诉 Task 这是一个交互式应用程序，Task 将尝试针对它进行优化：

```yaml
version: '3'

tasks:
  default:
    cmds:
      - vim my-file.txt
    interactive: true
```

如果您在通过 Task 运行交互式应用程序时仍然遇到问题，请打开一个关于它的 Issue。

## 短 Task 语法

从 Task v3 开始，如果 task 具有默认设置（例如：没有自定义 `env:`、`vars:`、`desc:`、`silent:` 等），您现在可以使用更短的语法编写task：

```yaml
version: '3'

tasks:
  build: go build -v -o ./app{{exeExt}} .

  run:
    - task: build
    - ./app{{exeExt}} -h localhost -p 8080
```

## `set` 和 `shopt`

可以为 [`set`](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html) 和 [`shopt`](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html) 内置函数指定选项。 这可以在全局、task 或命令级别添加。

```yaml
version: '3'

set: [pipefail]
shopt: [globstar]

tasks:
  # `globstar` required for double star globs to work
  default: echo **/*.go
```

:::info

请记住，并非所有选项在 Task 使用的 [shell 解释器库](https://github.com/mvdan/sh) 中都可用。

:::

## 观察 task

使用 `--watch` 或 `-w` 参数可以观察文件变化，然后重新执行 task。 这需要配置 `sources` 属性，task 才知道观察哪些文件。

默认监控的时间间隔是 5 秒，但可以通过 Taskfile 中根属性 `interval: '500ms'` 设置，也可以通过命令行 参数 `--interval=500ms` 设置。

Also, it's possible to set `watch: true` in a given task and it'll automatically run in watch mode:

```yaml
version: '3'

interval: 500ms

tasks:
  build:
    desc: Builds the Go application
    watch: true
    sources:
      - '**/*.go'
    cmds:
      - go build  # ...
```

:::info

Note that when setting `watch: true` to a task, it'll only run in watch mode when running from the CLI via `task my-watch-task`, but won't run in watch mode if called by another task, either directly or as a dependency.

:::

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[gotemplate]: https://golang.org/pkg/text/template/
