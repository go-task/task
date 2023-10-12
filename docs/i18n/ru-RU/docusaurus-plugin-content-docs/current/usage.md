---
slug: /usage/
sidebar_position: 3
---

# Использование

## Начало работы

Создайте файл с именем `Taskfile.yml` в корне вашего проекта. Атрибут `cmds` должен содержать команды задачи. Пример ниже позволяет скомпилировать приложение Go и использовать [esbuild](https://esbuild.github.io/) чтобы собрать и минимизировать несколько CSS файлов в один.

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

Запуск задач настолько прост, что достаточно выполнить команду:

```bash
task assets build
```

Task использует [mvdan.cc/sh](https://mvdan.cc/sh/) - нативный интерпретатор sh на Go. Таким образом, вы можете писать команды sh / bash, и они будут работать даже в Windows, где обычно не доступны `sh` или `bash`. Просто помните, что любой исполняемый файл, который вызывается, должен быть доступен ОС или находиться в переменной PATH.

Если вы опустите имя задачи, то будет использоваться имя "default".

## Поддерживаемые названия файлов

Task будет искать следующие файлы, в порядке приоритета:

- Taskfile.yml
- taskfile.yml
- Taskfile.yaml
- taskfile.yaml
- Taskfile.dist.yml
- taskfile.dist.yml
- Taskfile.dist.yaml
- taskfile.dist.yaml

Идея создания вариантов `.dist` заключается в том, чтобы позволить проектам иметь одну фиксированную версию (`.dist`), при этом позволяя отдельным пользователям переопределить Taskfile, добавив дополнительный `Taskfile.yml` (который будет находится в `.gitignore`).

### Запуск Taskfile из поддиректории

Если Taskfile не найден в текущем рабочем каталоге, он будет искать его вверх по дереву файлов, пока не найдет его (похоже на то, как работает `git`). При запуске Task из подкаталога, он будет работать так, как будто вы запустили его из каталога, содержащего Taskfile.

Вы можете использовать эту функцию вместе со специальной переменной `{{.USER_WORKING_DIR}}`, чтобы создавать переиспользуемые задачи. Например, если у вас есть монорепозиторий с каталогами для каждого микросервиса, вы можете `cd` в директорию микросервиса и запустить команду задачи, без создания нескольких задач или Taskfile с идентичным содержимым. Например:

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

В этом примере мы можем выполнить `cd <service>` и `task up`, и при условии, что каталог `<service>` содержит файл `docker-compose.yml`, Docker composition будет запущен.

### Запуск глобального Taskfile

Если вы вызовите Task с помощью флага `--global` (псевдоним `-g`), будет искать ваш домашний каталог вместо рабочего каталога. In short, Task will look for a Taskfile that matches `$HOME/{T,t}askfile.{yml,yaml}` .

Это полезно, чтобы иметь автоматизацию, которую можно запустить из любого места вашей системы!

:::info

Когда вы запускаете ваш глобальный Taskfile с помощью `-g`, task будут выполняться по умолчанию в директории `$HOME`, а не в вашей рабочей директории!

Как упоминалось в предыдущем разделе, специальная переменная `{{.USER_WORKING_DIR}}` может быть очень полезной для запуска команд в директории, из которой вы вызываете `task -g`.

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

## Переменные среды

### Task

Вы можете использовать `env` для создания своих переменных среды для конкретной task:

```yaml
version: '3'

tasks:
  greet:
    cmds:
      - echo $GREETING
    env:
      GREETING: Hey, there!
```

Также, вы можете создавать глобальные переменные окружения, которые будут доступны всем task:

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

`env` поддерживает дополнение и извлечение вывода из команды shell или переменной, вы можете посмотреть в разделе [Переменные](#variables).

:::

### .env файлы

Вы также можете попросить Task включать файлы, подобные `.env` используя настройку `dotenv:`:

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

Dotenv файлы также могут быть указаны на уровне task:

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

Переменные окружения, определенные на уровне task, заменят переменные объявленные в dotenv файлах:

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

Обратите внимание, в данный момент вы не можете использовать ключ `dotenv` во вложенных Taskfile.

:::

## Включение других Taskfile

Если вы хотите использовать task в других проектах (Taskfile), вы можете использовать механизм импорта для включения других Taskfile используя ключевое слово `includes`:

```yaml
version: '3'

includes:
  docs: ./documentation # will look for ./documentation/Taskfile.yml
  docker: ./DockerTasks.yml
```

Task, описанные в указанных Taskfile, будут доступны с указанным пространством имен. Таким образом, вы можете запустить `serve` task из файла `documentation/Taskfile.yml` с помощью команды `task docs:serve`, или запустить `build` task из файла `DockerTasks.yml` с помощью команды `task docker:build`.

Относительные пути разрешаются относительно каталога, содержащего включающий Taskfile.

### Специфичные для ОС Taskfile

С помощью `version: '2'` task автоматически включает любой `Taskfile_{{OS}}.yml`, если такой файл существует (например: `Taskfile_windows.yml`, `Taskfile_linux.yml` или `Taskfile_darwin.yml`). Так как такое поведение было несколько неявным, оно было удалено в версии 3. Тем не менее можно получить схожее поведение, явно импортировав соответствующие файлы:

```yaml
version: '3'

includes:
  build: ./Taskfile_{{OS}}.yml
```

### Директория включенного Taskfile

По умолчанию task включенного Taskfile выполняются в текущем каталоге, даже если Taskfile находится в другом каталоге, но вы можете заставить задачи выполняться в другом каталоге, используя альтернативный синтаксис:

```yaml
version: '3'

includes:
  docs:
    taskfile: ./docs/Taskfile.yml
    dir: ./docs
```

:::info

Включенные Taskfile должны использовать ту же версию схемы, что и основной Taskfile.

:::

### Опциональные включения

Включения, отмеченные как необязательные, позволяют Task продолжать выполнение в нормальном режиме, если включенный файл отсутствует.

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

### Внутренние включения

Включения, отмеченные как internal, устанавливают также все задачи включенного файла как internal (см. секцию [Внутренние task](#внутренние-task) ниже). Это полезно, когда включаются утилитарные task, которые не предназначены для прямого использования пользователем.

```yaml
version: '3'

includes:
  tests:
    taskfile: ./taskfiles/Utils.yml
    internal: true
```

### Переменные включенных Taskfile

Вы также можете указывать переменные при включении Taskfile. Это может быть полезно для создания переиспользуемого Taskfile, который можно настроить или даже включать более одного раза:

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

### Псевдонимы пространств имен

При включении Taskfile, вы можете дать пространству имен список `aliases`. Это работает так же, как и [псевдонимы task](#псевдонимы-task) и может использоваться вместе, чтобы создавать более короткие и легко набираемые команды.

```yaml
version: '3'

includes:
  generate:
    taskfile: ./taskfiles/Generate.yml
    aliases: [gen]
```

:::info

Переменные, объявленные во включенном Taskfile, имеют приоритет над переменными включающего Taskfile! Если вы хотите, чтобы переменную включенного Taskfile можно было переопределить, используйте [функцию default](https://go-task.github.io/slim-sprig/defaults.html): `MY_VAR: '{{.MY_VAR | default "my-default-value"}}'`.

:::

## Внутренние task

Внутренние task - это task, которые не могут быть вызваны напрямую пользователем. Они не будут отображаться в выводе при вызове команды `task --list|--list-all`. Другие task могут вызывать внутренние задачи обычным способом. Это полезно для создания переиспользуемых task, похожих на функции, которые не имеют практического значения при выполнении в командной строке.

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

## Директория task

По умолчанию, task выполняются в директории, где находится Taskfile. Но вы можете легко заставить task выполниться в другом каталоге, указав `dir`:

```yaml
version: '3'

tasks:
  serve:
    dir: public/www
    cmds:
      # run http server
      - caddy
```

Если директории не существует, `task` создаст ее.

## Зависимости task

> Зависимости выполняются параллельно, поэтому зависимости task не должны зависеть друг от друга. Если вы хотите принудительно запустить задачи последовательно, обратите внимание на раздел [Вызов другой task](#вызов-другой-task), описанный ниже.

У вас могут быть task, которые зависят от других. Просто перечислите их в списке `deps`, и они будут автоматически запущены перед запуском родительской task:

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

В примере выше, `assets` будет всегда запускаться перед `build`, если вы запустите `task build`.

Task может иметь зависимости без команд, чтобы группировать задачи вместе:

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

Если есть более одной зависимости, то они всегда выполняются параллельно для лучшей производительности.

:::tip

Вы также можете запустить несколько task, указанные в командной строке, параллельно, используя флаг `--parallel` (псевдоним `-p`). Например: `task --parallel js css`.

:::

Если вы хотите передать информацию зависимостям, вы можете сделать это таким же образом, как и при [вызове другой task](#вызов-другой-task):

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

## Платформно-зависимые tasks и команды

Если вы хотите ограничить запуск tasks определенными платформами, вы можете сделать это используя ключ `platforms:`. Tasks могут быть ограничены определенной ОС, архитектурой или комбинацией этих элементов. В случае несоответствия, task или команда будут пропущены, и ошибка не вернется.

Разрешенные значения ОС и архитектур или `GOOS` и `GOARCH` определены языком Go [здесь](https://github.com/golang/go/blob/master/src/go/build/syslist.go).

Task ниже `build-windows` будет выполняться только на Windows любой архитектуры:

```yaml
version: '3'

tasks:
  build-windows:
    platforms: [windows]
    cmds:
      - echo 'Running command on Windows'
```

Это можно ограничить определенной архитектурой следующим образом:

```yaml
version: '3'

tasks:
  build-windows-amd64:
    platforms: [windows/amd64]
    cmds:
      - echo 'Running command on Windows (amd64)'
```

Также можно ограничить task определенными архитектурами:

```yaml
version: '3'

tasks:
  build-amd64:
    platforms: [amd64]
    cmds:
      - echo 'Running command on amd64'
```

Несколько платформ можно указать так:

```yaml
version: '3'

tasks:
  build:
    platforms: [windows/amd64, darwin]
    cmds:
      - echo 'Running command on Windows (amd64) and macOS'
```

Отдельные команды также могут быть ограничены определенными платформами:

```yaml
version: '3'

tasks:
  build:
    cmds:
      - cmd: echo 'Running command on Windows (amd64) and macOS'
        platforms: [windows/amd64, darwin]
      - cmd: echo 'Running on all platforms'
```

## Вызов другой task

Когда task имеет много зависимостей, они выполняются параллельно. Это частно приводит к более быстрому построению пайплайна. Однако, в некоторых ситуациях вам может понадобиться вызвать другие task последовательно. В этом случае используйте следующий синтаксис:

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

Используя атрибуты `vars` и `silent`, вы можете выбирать, передавать ли переменные и включать или выключать [silent mode](#silent-mode) для вызова каждый раз отдельно:

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

Указанный выше синтаксис также поддерживается в `deps`.

:::tip

NOTE: If you want to call a task declared in the root Taskfile from within an [included Taskfile](#including-other-taskfiles), add a leading `:` like this: `task: :task-name`.

:::

## Prevent unnecessary work

### By fingerprinting locally generated files and their sources

If a task generates something, you can inform Task the source and generated files, so Task will prevent running them if not necessary.

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

`sources` and `generates` can be files or file patterns. When given, Task will compare the checksum of the source files to determine if it's necessary to run the task. If not, it will just print a message like `Task "js" is up to date`.

If you prefer this check to be made by the modification timestamp of the files, instead of its checksum (content), just set the `method` property to `timestamp`.

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

In situations where you need more flexibility the `status` keyword can be used. You can even combine the two. See the documentation for [status](#using-programmatic-checks-to-indicate-a-task-is-up-to-date) for an example.

:::info

By default, task stores checksums on a local `.task` directory in the project's directory. Most of the time, you'll want to have this directory on `.gitignore` (or equivalent) so it isn't committed. (If you have a task for code generation that is committed it may make sense to commit the checksum of that task as well, though).

If you want these files to be stored in another directory, you can set a `TASK_TEMP_DIR` environment variable in your machine. It can contain a relative path like `tmp/task` that will be interpreted as relative to the project directory, or an absolute or home path like `/tmp/.task` or `~/.task` (subdirectories will be created for each project).

```bash
export TASK_TEMP_DIR='~/.task'
```

:::

:::info

Each task has only one checksum stored for its `sources`. If you want to distinguish a task by any of its input variables, you can add those variables as part of the task's label, and it will be considered a different task.

This is useful if you want to run a task once for each distinct set of inputs until the sources actually change. For example, if the sources depend on the value of a variable, or you if you want the task to rerun if some arguments change even if the source has not.

:::

:::tip

The method `none` skips any validation and always run the task.

:::

:::info

For the `checksum` (default) or `timestamp` method to work, it is only necessary to inform the source files. When the `timestamp` method is used, the last time of the running the task is considered as a generate.

:::

### Using programmatic checks to indicate a task is up to date

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

You can use `--force` or `-f` if you want to force a task to run even when up-to-date.

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

### Using programmatic checks to cancel the execution of a task and its dependencies

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

### Limiting when tasks run

If a task executed by multiple `cmds` or multiple `deps` you can control when it is executed using `run`. `run` can also be set at the root of the Taskfile to change the behavior of all the tasks unless explicitly overridden.

Supported values for `run`:

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

### Ensuring required variables are set

If you want to check that certain variables are set before running a task then you can use `requires`. This is useful when might not be clear to users which variables are needed, or if you want clear message about what is required. Also some tasks could have dangerous side effects if run with un-set variables.

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

## Variables

When doing interpolation of variables, Task will look for the below. They are listed below in order of importance (i.e. most important first):

- Variables declared in the task definition
- Variables given while calling a task from another (See [Calling another task](#calling-another-task) above)
- Variables of the [included Taskfile](#including-other-taskfiles) (when the task is included)
- Variables of the [inclusion of the Taskfile](#vars-of-included-taskfiles) (when the task is included)
- Global variables (those declared in the `vars:` option in the Taskfile)
- Environment variables

Example of sending parameters with environment variables:

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

### Dynamic variables

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

## Forwarding CLI arguments to commands

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

## Doing task cleanup with `defer`

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

If you want to move the cleanup command into another task, that is possible as well:

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

## Go's template engine

Task parse commands as [Go's template engine][gotemplate] before executing them. Variables are accessible through dot syntax (`.VARNAME`).

All functions by the Go's [slim-sprig lib](https://go-task.github.io/slim-sprig/) are available. The following example gets the current date in a given format:

```yaml
version: '3'

tasks:
  print-date:
    cmds:
      - echo {{now | date "2006-01-02"}}
```

Task also adds the following functions:

- `OS`: Returns the operating system. Possible values are `windows`, `linux`, `darwin` (macOS) and `freebsd`.
- `ARCH`: return the architecture Task was compiled to: `386`, `amd64`, `arm` or `s390x`.
- `splitLines`: Splits Unix (`\n`) and Windows (`\r\n`) styled newlines.
- `catLines`: Replaces Unix (`\n`) and Windows (`\r\n`) styled newlines with a space.
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

## Help

Running `task --list` (or `task -l`) lists all tasks with a description. The following Taskfile:

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

would print the following output:

```bash
* build:   Build the go binary.
* test:    Run all the go tests.
```

If you want to see all tasks, there's a `--list-all` (alias `-a`) flag as well.

## Display summary of task

Running `task --summary task-name` will show a summary of a task. The following Taskfile:

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

with running `task --summary release` would print the following output:

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

If a summary is missing, the description will be printed. If the task does not have a summary or a description, a warning is printed.

Please note: _showing the summary will not execute the command_.

## Псевдонимы task

Aliases are alternative names for tasks. They can be used to make it easier and quicker to run tasks with long or hard-to-type names. You can use them on the command line, when [calling sub-tasks](#calling-another-task) in your Taskfile and when [including tasks](#including-other-taskfiles) with aliases from another Taskfile. They can also be used together with [namespace aliases](#namespace-aliases).

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

## Overriding task name

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

## Warning Prompts

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

## Silent mode

Silent mode disables the echoing of commands before Task runs it. For the following Taskfile:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "Print something"
```

Normally this will be printed:

```sh
echo "Print something"
Print something
```

With silent mode on, the below will be printed instead:

```sh
Print something
```

There are four ways to enable silent mode:

- At command level:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - cmd: echo "Print something"
        silent: true
```

- At task level:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "Print something"
    silent: true
```

- Globally at Taskfile level:

```yaml
version: '3'

silent: true

tasks:
  echo:
    cmds:
      - echo "Print something"
```

- Or globally with `--silent` or `-s` flag

If you want to suppress STDOUT instead, just redirect a command to `/dev/null`:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "This will print nothing" > /dev/null
```

## Dry run mode

Dry run mode (`--dry`) compiles and steps through each task, printing the commands that would be run without executing them. This is useful for debugging your Taskfiles.

## Ignore errors

You have the option to ignore errors during command execution. Given the following Taskfile:

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

## Output syntax

By default, Task just redirects the STDOUT and STDERR of the running commands to the shell in real-time. This is good for having live feedback for logging printed by commands, but the output can become messy if you have multiple commands running simultaneously and printing lots of stuff.

To make this more customizable, there are currently three different output options you can choose:

- `interleaved` (default)
- `group`
- `prefixed`

To choose another one, just set it to root in the Taskfile:

```yaml
version: '3'

output: 'group'

tasks:
  # ...
```

The `group` output will print the entire output of a command once after it finishes, so you will not have live feedback for commands that take a long time to run.

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

## Interactive CLI application

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

## Short task syntax

Starting on Task v3, you can now write tasks with a shorter syntax if they have the default settings (e.g. no custom `env:`, `vars:`, `desc:`, `silent:` , etc):

```yaml
version: '3'

tasks:
  build: go build -v -o ./app{{exeExt}} .

  run:
    - task: build
    - ./app{{exeExt}} -h localhost -p 8080
```

## `set` and `shopt`

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

## Watch tasks

With the flags `--watch` or `-w` task will watch for file changes and run the task again. This requires the `sources` attribute to be given, so task knows which files to watch.

The default watch interval is 5 seconds, but it's possible to change it by either setting `interval: '500ms'` in the root of the Taskfile passing it as an argument like `--interval=500ms`.

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
