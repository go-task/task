---
slug: /styleguide/
sidebar_position: 10
---

# Стайлгайд

Это официальный стайлгайд Task для `Taskfile.yml` файлов. Это руководство содержит некоторые базовые инструкции для того, чтобы ваш Taskfile был чистым и понятен другим пользователям.

Этот стайлгайд содержит общие рекомендации по написанию кода, но не обязательно требует их строгого соблюдения. Можете не соглашаться с правилами и использовать другой подход, если вам это нужно или хотите это сделать. Кроме того, не стесняйтесь создавать Issue или PR с улучшениями этого гида.

## Используйте `Taskfile.yml` вместо `taskfile.yml`

```yaml
# bad
taskfile.yml


# good
Taskfile.yml
```

Это особенно важно для пользователей Linux. У Windows и macOS нечувствительные файловые системы, поэтому `taskfile.yml` в конечном итоге будет работать, даже если официально не поддерживается. В Linux только будет работать `Taskfile.yml`.

## Используйте правильный порядок ключевых слов

- `version:`
- `includes:`
- Конфигурационные параметры, такие как `output:`, `silent:`, `method:` и `run:`
- `vars:`
- `env:`, `dotenv:`
- `tasks:`

## Используйте 2 пробела для отступа

Это наиболее распространенное соглашение для YAML-файлов, и Task следует ему.

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

## Разделяйте основные секции пробелами

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

## Добавляйте пробелы между задачами

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

## Используйте имена переменных в верхнем регистре

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

## Не заключайте переменные в пробелы при использовании их в шаблонах

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

Большинство людей использует это соглашение и для любых шаблонов в Go.

## Разделяйте слова в названии задач дефисом

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

## Используйте двоеточие для неймспейсов в названиях задач

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

Это также происходит автоматически при использовании включенных Taskfiles.

## Prefer external scripts over complex multi-line commands

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
