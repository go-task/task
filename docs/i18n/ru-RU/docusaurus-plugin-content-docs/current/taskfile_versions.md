---
slug: /taskfile-versions/
sidebar_position: 5
---

# Версии Taskfile

Синтаксис Taskfile и функции со временем изменяются. Этот документ объясняет, что изменилось в каждой версии и как обновить свой Taskfile.

## Что обозначает версия Taskfile

Версия Taskfile соответствует версии Task. Например: изменение на Taskfile версии `2` означает, что Task `v2.0.0` должна быть выпущена для ее поддержки.

`version:` ключ Taskfile принимает [semVer](https://semver.org/lang/ru/) строку. Пример: `2`, `2.0` или `2.0.0`. Если вы решите использовать Task версии `2.0`, то у вас не будет доступа к функциям версии `2.1`, но если вы решите использовать версию `2`, то любые функции версий `2.x.x` будут доступны, но не `3.0.0+`.

## Version 3 ![latest](https://img.shields.io/badge/latest-brightgreen)

Основные изменения, сделанные в `v3`:

- Output задачи теперь цветной
- Добавлена поддержка `.env` файлов
- Добавлен параметр `label:`. Появилась возможность переопределить имя задачи в логах
- Глобальный параметр `method:` был добавлен для установки метода по умолчанию, а задача по умолчанию изменена на `checksum`
- Two magic variables were added when using `status:`: `CHECKSUM` and `TIMESTAMP` which contains, respectively, the XXH3 checksum and greatest modification timestamp of the files listed on `sources:`
- Кроме того, переменная `TASK` всегда доступна по имени текущей задачи
- Переменные CLI всегда считаются глобальными переменными
- Добавлена опция `dir:` в `includes` для того, чтобы выбрать, в каком каталоге Taskfile будет запущен:

```yaml
includes:
  docs:
    taskfile: ./docs
    dir: ./docs
```

- Реализован короткий синтаксис задачи. Все синтаксисы ниже эквивалентны:

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

- Был произведён большой рефакторинг обработки переменных. Теперь всё стало более прозрачно. The `expansions:` setting was removed as it became unnecessary. Это порядок, в котором Task будет обрабатывать переменные, каждый уровень может видеть переменные, объявленные на предыдущем и переопределять их.
  - Переменные окружения
  - Глобальные + CLI переменные
  - Вызов переменных
  - Переменные Task

## Версия 2.6

:::caution

v2 schema support is [deprecated][deprecate-version-2-schema] and will be removed in a future release.

:::

Версия 2.6 поставляется с `preconditions` опцией в задачах.

```yaml
version: '2'

tasks:
  upload_environment:
    preconditions:
      - test -f .env
    cmds:
      - aws s3 cp .env s3://myenvironment
```

Пожалуйста, проверьте [документацию][includes]

## Версия 2.2

:::caution

v2 schema support is [deprecated][deprecate-version-2-schema] and will be removed in a future release.

:::

В Версии 2.2 появилась новая глобальная опция `includes`, которая позволяет импортировать другие Taskfile'ы:

```yaml
version: '2'

includes:
  docs: ./documentation # will look for ./documentation/Taskfile.yml
  docker: ./DockerTasks.yml
```

## Версия 2.1

:::caution

v2 schema support is [deprecated][deprecate-version-2-schema] and will be removed in a future release.

:::

В версии 2.1 появилась глобальная опция `output`, которая позволяет иметь больше контроля над тем, как вывод команд печатается на консоли (см. [документацию][output]):

```yaml
version: '2'

output: prefixed

tasks:
  server:
    cmds:
      - go run main.go
  prefix: server
```

Начиная с этой версии можно игнорировать ошибки команды или задачи (смотрите документацию [здесь][ignore_errors]):

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

## Версия 2.0

:::caution

v2 schema support is [deprecated][deprecate-version-2-schema] and will be removed in a future release.

:::

В версии 2 был добавлен ключ `version: `. Он позволяет выпускать обновления сохраняя обратную совместимость. Пример использования:

```yaml
version: '2'

tasks:
  echo:
    cmds:
      - echo "Hello, World!"
```

Версия 2 позволяет создавать глобальные переменные непосредственно в Taskfile, если вы не хотите создавать `Taskvars.yml`:

```yaml
version: '2'

vars:
  GREETING: Hello, World!

tasks:
  greet:
    cmds:
      - echo "{{.GREETING}}"
```

Порядок приоритетов переменных также отличается:

1. Переменные Task
2. Call variables
3. Переменные Taskfile
4. Переменные `Taskvars.yml`
5. Environment variables

Добавлена новая глобальная опция для настройки количества расширений переменных (по умолчанию 2):

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

## Версия 1

:::caution

v1 schema support was removed in Task >= v3.0.0.

:::

В первой версии `Taskfile` поле `version:` не доступно, потому что задачи были в корне документа YAML. Пример:

```yaml
echo:
  cmds:
    - echo "Hello, World!"
```

Порядок приоритетов переменных также отличается:

1. Call variables
2. Переменные среды
3. Переменные Task
4. Переменные `Taskvars.yml`

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[output]: usage.md#output-syntax
[ignore_errors]: usage.md#ignore-errors
[includes]: usage.md#including-other-taskfiles
[deprecate-version-2-schema]: https://github.com/go-task/task/issues/1197
