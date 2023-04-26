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
- Taskfile.yaml
- Taskfile.dist.yml
- Taskfile.dist.yaml

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

如果您使用 `--global`（别名 `-g`）标志调用 Task，它将查找您的 home 目录而不是您的工作目录。 简而言之，Task 将在 `$HOME/Taskfile.yml` 或 `$HOME/Taskfile.yaml` 路径上寻找 Taskfile。

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

给定的 Taskfile 中描述的任务将在指定的命名空间中提供。 因此，您可以从 `documentation/Taskfile.yml` 调用 `task docs:serve` 运行 `serve` task，或者从 `DockerTasks.yml` 文件调用 `task docker:build` 运行 `build` task。

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

### 包含的 Taskfile 的变量

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

在被调用 task 中覆盖变量就像通知 `vars` 属性一样简单：

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

默认情况，task 在本地项目的 `.task` 目录保存 checksums 值。 一般都会在 `.gitignore`（或类似配置）中忽略掉这个目录，这样它就不会被提交。 (If you have a task for code generation that is committed it may make sense to commit the checksum of that task as well, though).

如果你想要将这些文件存储在另一个目录中，你可以在你的机器中设置一个 `TASK_TEMP_DIR` 环境变量。 可以使用相对路径，比如 `tmp/task`，相对项目根目录，也可以用绝对路径、用户目录路径，比如 `/tmp/.task` 或 `~/.task`（每个项目单独创建子目录）。

```bash
export TASK_TEMP_DIR='~/.task'
```

:::

:::info

Each task has only one checksum stored for its `sources`. If you want to distinguish a task by any of its input variables, you can add those variables as part of the task's label, and it will be considered a different task.

This is useful if you want to run a task once for each distinct set of inputs until the sources actually change. For example, if the sources depend on the value of a variable, or you if you want the task to rerun if some arguments change even if the source has not.

:::

:::tip

将 method 设置为 `none` 会跳过任何验证并始终运行任务。

:::

:::info

For the `checksum` (default) or `timestamp` method to work, it is only necessary to inform the source files. When the `timestamp` method is used, the last time of the running the task is considered as a generate.

:::

### 使用程序检查来表示任务是最新的

Alternatively, you can inform a sequence of tests as `status`. If no error is returned (exit status 0), the task is considered up-to-date:

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

Normally, you would use `sources` in combination with `generates` - but for tasks that generate remote artifacts (Docker images, deploys, CD releases) the checksum source and timestamps require either access to the artifact or for an out-of-band refresh of the `.checksum` fingerprint file.

Two special variables `{{.CHECKSUM}}` and `{{.TIMESTAMP}}` are available for interpolation within `status` commands, depending on the method assigned to fingerprint the sources. Only `source` globs are fingerprinted.

Note that the `{{.TIMESTAMP}}` variable is a "live" Go `time.Time` struct, and can be formatted using any of the methods that `time.Time` responds to.

See [the Go Time documentation](https://golang.org/pkg/time/) for more information.

如果你想强制任务运行，即使是最新的，你也可以使用 `--force` 或 `-f`。

Also, `task --status [tasks]...` will exit with a non-zero exit code if any of the tasks are not up-to-date.

`status` can be combined with the [fingerprinting](#by-fingerprinting-locally-generated-files-and-their-sources) to have a task run if either the the source/generated artifacts changes, or the programmatic check fails:

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

In addition to `status` checks, `preconditions` checks are the logical inverse of `status` checks. That is, if you need a certain set of conditions to be _true_ you can use the `preconditions` stanza. `preconditions` are similar to `status` lines, except they support `sh` expansion, and they SHOULD all return 0.

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

Preconditions can set specific failure messages that can tell a user what steps to take using the `msg` field.

If a task has a dependency on a sub-task with a precondition, and that precondition is not met - the calling task will fail. Note that a task executed with a failing precondition will not run unless `--force` is given.

Unlike `status`, which will skip a task if it is up to date and continue executing tasks that depend on it, a `precondition` will fail a task, along with any other tasks that depend on it.

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

If a task executed by multiple `cmds` or multiple `deps` you can control when it is executed using `run`. `run` can also be set at the root of the Taskfile to change the behavior of all the tasks unless explicitly overridden.

`run` 支持的值：

- `always` (default) always attempt to invoke the task regardless of the number of previous executions
- `once` only invoke this task once regardless of the number of references
- `when_changed` only invokes the task once for each unique set of variables passed into the task

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

## 变量

在进行变量插值时，Task 将查找以下内容。 They are listed below in order of importance (i.e. most important first):

- Variables declared in the task definition
- Variables given while calling a task from another (See [Calling another task](#calling-another-task) above)
- Variables of the [included Taskfile](#including-other-taskfiles) (when the task is included)
- Variables of the [inclusion of the Taskfile](#vars-of-included-taskfiles) (when the task is included)
- Global variables (those declared in the `vars:` option in the Taskfile)
- Environment variables

使用环境变量传输参数的示例：

```bash
$ TASK_VARIABLE=a-value task do-something
```

:::tip

A special variable `.TASK` is always available containing the task name.

:::

Since some shells do not support the above syntax to set environment variables (Windows) tasks also accept a similar style when not at the beginning of the command.

```bash
$ task write-file FILE=file.txt "CONTENT=Hello, World!" print "MESSAGE=All done!"
```

Example of locally declared vars:

```yaml
version: '3'

tasks:
  print-var:
    cmds:
      - echo "{{.VAR}}"
    vars:
      VAR: Hello!
```

Example of global vars in a `Taskfile.yml`:

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

The below syntax (`sh:` prop in a variable) is considered a dynamic variable. The value will be treated as a command and the output assigned. If there are one or more trailing newlines, the last newline will be trimmed.

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

This works for all types of variables.

## 将 CLI 参数转发到 cmds

If `--` is given in the CLI, all following parameters are added to a special `.CLI_ARGS` variable. This is useful to forward arguments to another command.

The below example will run `yarn install`.

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

With the `defer` keyword, it's possible to schedule cleanup to be run once the task finishes. The difference with just putting it as the last command is that this command will run even when the task fails.

In the example below, `rm -rf tmpdir/` will run even if the third command fails:

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

Due to the nature of how the [Go's own `defer` work](https://go.dev/tour/flowcontrol/13), the deferred commands are executed in the reverse order if you schedule multiple of them.

:::

## Go 的模板引擎

Task 在执行命令之前将命令解析为 [Go 的模板引擎][gotemplate]。 可以通过点语法 (`.VARNAME`) 访问变量。

All functions by the Go's [slim-sprig lib](https://go-task.github.io/slim-sprig/) are available. The following example gets the current date in a given format:

```yaml
version: '3'

tasks:
  print-date:
    cmds:
      - echo {{now | date "2006-01-02"}}
```

Task 还增加了以下功能：

- `OS`: Returns the operating system. Possible values are "windows", "linux", "darwin" (macOS) and "freebsd".
- `ARCH`: return the architecture Task was compiled to: "386", "amd64", "arm" or "s390x".
- `splitLines`: Splits Unix (\n) and Windows (\r\n) styled newlines.
- `catLines`: Replaces Unix (\n) and Windows (\r\n) styled newlines with a space.
- `toSlash`: Does nothing on Unix, but on Windows converts a string from `\` path format to `/`.
- `fromSlash`: Opposite of `toSlash`. Does nothing on Unix, but on Windows converts a string from `/` path format to `\`.
- `exeExt`: Returns the right executable extension for the current OS (`".exe"` for Windows, `""` for others).
- `shellQuote`: Quotes a string to make it safe for use in shell scripts. Task uses [this Go function](https://pkg.go.dev/mvdan.cc/sh/v3@v3.4.0/syntax#Quote) for this. The Bash dialect is assumed.
- `splitArgs`: Splits a string as if it were a command's arguments. Task uses [this Go function](https://pkg.go.dev/mvdan.cc/sh/v3@v3.4.0/shell#Fields)

Example:

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

运行 `task --list`（或 `task -l`）列出所有带有描述的任务。 The following Taskfile:

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

运行 `task --summary task-name` 将显示任务的摘要。 The following Taskfile:

```yaml
version: '3'

tasks:
  release:
    deps: [build]
    summary: |
      发布你的项目到 github

      它将在开始发布之前构建您的项目。
      请确保您在开始之前已经设置了 GITHUB_TOKEN。
    cmds:
      - your-release-tool

  build:
    cmds:
      - your-build-tool
```

运行 `task --summary release` 将打印以下输出：

```
task: release

发布你的项目到 github

它将在开始发布之前构建您的项目。
请确保您在开始之前已经设置了 GITHUB_TOKEN。

dependencies:
 - build

commands:
 - your-release-tool
```

如果缺少摘要，将打印描述。 If the task does not have a summary or a description, a warning is printed.

Please note: _showing the summary will not execute the command_.

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

Sometimes you may want to override the task name printed on the summary, up-to-date messages to STDOUT, etc. In this case, you can just set `label:`, which can also be interpolated with variables:

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

## 静默模式

静默模式在 Task 运行命令之前禁用命令回显。 For the following Taskfile:

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

Dry run mode (`--dry`) compiles and steps through each task, printing the commands that would be run without executing them. This is useful for debugging your Taskfiles.

## 忽略错误

您可以选择在命令执行期间忽略错误。 Given the following Taskfile:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - exit 1
      - echo "Hello World"
```

Task will abort the execution after running `exit 1` because the status code `1` stands for `EXIT_FAILURE`. However, it is possible to continue with execution using `ignore_error`:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - cmd: exit 1
        ignore_error: true
      - echo "Hello World"
```

`ignore_error` can also be set for a task, which means errors will be suppressed for all commands. Nevertheless, keep in mind that this option will not propagate to other tasks called either by `deps` or `cmds`!

## 输出语法

By default, Task just redirects the STDOUT and STDERR of the running commands to the shell in real-time. This is good for having live feedback for logging printed by commands, but the output can become messy if you have multiple commands running simultaneously and printing lots of stuff.

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

When using the `group` output, you can optionally provide a templated message to print at the start and end of the group. This can be useful for instructing CI systems to group all of the output for a given task, such as with [GitHub Actions' `::group::` command](https://docs.github.com/en/actions/learn-github-actions/workflow-commands-for-github-actions#grouping-log-lines) or [Azure Pipelines](https://docs.microsoft.com/en-us/azure/devops/pipelines/scripts/logging-commands?expand=1&view=azure-devops&tabs=bash#formatting-commands).

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

When using the `group` output, you may swallow the output of the executed command on standard output and standard error if it does not fail (zero exit code).

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

The `prefix` output will prefix every line printed by a command with `[task-name]` as the prefix, but you can customize the prefix for a command with the `prefix:` attribute:

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

The `output` option can also be specified by the `--output` or `-o` flags.

:::

## 交互式 CLI 应用

When running interactive CLI applications inside Task they can sometimes behave weirdly, especially when the [output mode](#output-syntax) is set to something other than `interleaved` (the default), or when interactive apps are run in parallel with other tasks.

The `interactive: true` tells Task this is an interactive application and Task will try to optimize for it:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - vim my-file.txt
    interactive: true
```

If you still have problems running an interactive app through Task, please open an issue about it.

## 短 Task 语法

Starting on Task v3, you can now write tasks with a shorter syntax if they have the default settings (e.g. no custom `env:`, `vars:`, `desc:`, `silent:` , etc):

```yaml
version: '3'

tasks:
  build: go build -v -o ./app{{exeExt}} .

  run:
    - task: build
    - ./app{{exeExt}} -h localhost -p 8080
```

## `set` 和 `shopt`

It's possible to specify options to the [`set`](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html) and [`shopt`](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html) builtins. This can be added at global, task or command level.

```yaml
version: '3'

set: [pipefail]
shopt: [globstar]

tasks:
  # `globstar` required for double star globs to work
  default: echo **/*.go
```

:::info

Keep in mind that not all options are available in the [shell interpreter library](https://github.com/mvdan/sh) that Task uses.

:::

## 观察任务

With the flags `--watch` or `-w` task will watch for file changes and run the task again. This requires the `sources` attribute to be given, so task knows which files to watch.

The default watch interval is 5 seconds, but it's possible to change it by either setting `interval: '500ms'` in the root of the Taskfile passing it as an argument like `--interval=500ms`.

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[gotemplate]: https://golang.org/pkg/text/template/
