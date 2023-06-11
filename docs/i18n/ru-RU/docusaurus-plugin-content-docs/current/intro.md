---
slug: /
sidebar_position: 1
title: Home
---

# Task

<div align="center">
  <img id="logo" src="img/logo.svg" height="250px" width="250px" />
</div>

Task - это инструмент для запуска / сборки, который стремится быть простым и удобным в использовании, чем, например, [GNU Make][make].

Так как Task написан на [Go][go], он представляет собой единственный исполняемый файл и не имеет других зависимостей, что означает, что вам не нужно заниматься сложной настройкой установки просто для использования инструмента сборки.

После [установки](installation.md) вам просто нужно описать вашу задачу сборки, используя простую схему [YAML][yaml] в файле с именем `Taskfile.yml`:

```yaml title="Taskfile.yml"
version: '3'

tasks:
  hello:
    cmds:
      - echo 'Hello World from Task!'
    silent: true
```

И вызвать ее, запустив `task hello` в вашем терминале.

Приведенный выше пример - это только начало, вы можете посмотреть на [руководство](/usage) по использованию, чтобы посмотреть полную документацию схемы и функций Task.

## Особенности

- [Простая установка](installation.md): просто загрузите единственный исполняемый файл, добавьте его в `$PATH` и вы готовы! Или вы можете установить с помощью [Homebrew][homebrew], [Snapcraft][snapcraft] или [Scoop][scoop], если хотите.
- Доступен на CI-серверах: просто добавьте [эту простую команду](installation.md#install-script) для установки в CI-скрипт и готово! Теперь можно использовать Task в качестве части вашего CI-пайплайна.
- Полностью кроссплатформенный: в то время как большинство инструментов сборки хорошо работают только в Linux или macOS, Task также поддерживает Windows, благодаря [интерпретатору командной оболочки для Go][sh].
- Отлично подходит для кодогенерации: вы можете легко [предотвратить запуск задачи](/usage#prevent-unnecessary-work), если необходимый набор файлов не изменился с прошлого запуска (основываясь на времени изменения или содержимом).

## Золотые спонсоры

<div class="gold-sponsors">

| [Appwrite](https://appwrite.io/?utm_source=taskfile.dev&utm_medium=website&utm_campaign=task_oss_fund)                       |
| ---------------------------------------------------------------------------------------------------------------------------- |
| [![Appwrite](/img/appwrite.svg)](https://appwrite.io/?utm_source=taskfile.dev&utm_medium=website&utm_campaign=task_oss_fund) |

</div>

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[make]: https://www.gnu.org/software/make/
[go]: https://go.dev/
[yaml]: http://yaml.org/
[homebrew]: https://brew.sh/
[snapcraft]: https://snapcraft.io/
[scoop]: https://scoop.sh/
[sh]: https://github.com/mvdan/sh
