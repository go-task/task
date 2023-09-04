---
slug: /styleguide/
sidebar_position: 10
---

# 风格指南

这是对 `Taskfile.yml` 文件的官方风格指南。 本指南包含一些基本说明，可让您的任务文件保持简洁易用。

这包含一般准则，但不一定需要严格遵守。 如果你需要或想要，请随时提出不同意见，并在某些时候以不同方式进行。 此外，请随时打开 Issue 或 PR，对本指南进行改进。

## 使用 `Taskfile.yml` 而不是 `taskfile.yml`

```yaml
# bad
taskfile.yml


# good
Taskfile.yml
```

这对于 Linux 用户尤其重要。 Windows 和 macOS 的文件系统不区分大小写，因此 `taskfile.yml` 最终会正常工作，即使它不受官方支持。 不过，在 Linux 上，只有 `Taskfile.yml` 可以工作。

## 使用正确的关键字顺序

- `version:`
- `includes:`
- 可选配置命令，比如 `output:`、`silent:`、`method:` 和 `run:`
- `vars:`
- `env:`、`dotenv:`
- `tasks:`

## 使用 2 个空格缩进

这是 YAML 文件最常见的约定，Task 同样也遵循它。

```yaml
# bad
tasks:
    foo:
        cmds:
            - echo 'foo'


# good
tasks:
  foo:
    cmds:
      - echo 'foo'
```

## 用空行分隔主要部分

```yaml
# bad
version: '3'
includes:
  docker: ./docker/Taskfile.yml
output: prefixed
vars:
  FOO: bar
env:
  BAR: baz
tasks:
  # ...


# good
version: '3'

includes:
  docker: ./docker/Taskfile.yml

output: prefixed

vars:
  FOO: bar

env:
  BAR: baz

tasks:
  # ...
```

## 用空行分隔 task

```yaml
# bad
version: '3'

tasks:
  foo:
    cmds:
      - echo 'foo'
  bar:
    cmds:
      - echo 'bar'
  baz:
    cmds:
      - echo 'baz'


# good
version: '3'

tasks:
  foo:
    cmds:
      - echo 'foo'

  bar:
    cmds:
      - echo 'bar'

  baz:
    cmds:
      - echo 'baz'
```

## 使用大写变量名称

```yaml
# bad
version: '3'

vars:
  binary_name: myapp

tasks:
  build:
    cmds:
      - go build -o {{.binary_name}} .


# good
version: '3'

vars:
  BINARY_NAME: myapp

tasks:
  build:
    cmds:
      - go build -o {{.BINARY_NAME}} .
```

## 模板中不要用空格包住变量

```yaml
# bad
version: '3'

tasks:
  greet:
    cmds:
      - echo '{{ .MESSAGE }}'


# good
version: '3'

tasks:
  greet:
    cmds:
      - echo '{{.MESSAGE}}'
```

这个约定也被大多数人用于 Go 模板。

## 用破折号分隔任务名称单词

```yaml
# bad
version: '3'

tasks:
  do_something_fancy:
    cmds:
      - echo 'Do something'


# good
version: '3'

tasks:
  do-something-fancy:
    cmds:
      - echo 'Do something'
```

## 使用冒号作为任务命名空间

```yaml
# good
version: '3'

tasks:
  docker:build:
    cmds:
      - docker ...

  docker:run:
    cmds:
      - docker-compose ...
```

这在使用包含的任务文件时也会自动完成。

## 优先使用额外的脚本，避免使用复杂的多行命令。

```yaml
# bad
version: '3'

tasks:
  build:
    cmds:
      - |
        for i in $(seq 1 10); do
          echo $i
          echo "some other complex logic"
        done'

# good
version: '3'

tasks:
  build:
    cmds:
      - ./scripts/my_complex_script.sh
```
