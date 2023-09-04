---
slug: /integrations/
sidebar_position: 8
---

# Интеграции

## Расширение для Visual Studio Code

У Task есть официальное расширение для [Visual Studio Code](https://marketplace.visualstudio.com/items?itemName=task.vscode-task). Код для этого проекта можно найти [здесь](https://github.com/go-task/vscode-task). Чтобы использовать это расширение, на вашей системе должна быть установлена версия Task 3.23.0+.

Это расширение предоставляет следующие функции:

- Просмотр задач в боковой панели.
- Запуск задач из боковой панели и командной строки.
- Перейти к определению из боковой панели и командной строки.
- Выполнить последнюю "task" команду.
- Поддержка нескольких рабочих пространств.
- Инициализировать Taskfile в текущем рабочем пространстве.

Чтобы включить автозаполнение и проверку вашего Taskfile, см. раздел [Схема](#schema) ниже.

![Task for Visual Studio Code](https://github.com/go-task/vscode-task/blob/main/res/preview.png?raw=true)

## Схема

Изначально была создана [@KROSF](https://github.com/KROSF) вот тут [this Gist](https://gist.github.com/KROSF/c5435acf590acd632f71bb720f685895) и теперь официально поддерживается в этом [файле](https://github.com/go-task/task/blob/main/docs/static/schema.json) и доступна по ссылке https://taskfile.dev/schema.json. Эта схема может быть использована для проверки Task файлов и автодополнения во многих редакторах кода:

### Visual Studio Code

Чтобы интегрировать схему в VS Code, вам нужно установить [YAML расширение](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml) от Red Hat. Любой `Taskfile.yml` в вашем проекте должен автоматически быть обнаружен и валидирован/автодополнение должен работать. Если это не работает или вы хотите настроить его вручную для файлов с другим именем, вы можете добавить следующие в `settings.json`:

```json
// settings.json
{
  "yaml.schemas": {
    "https://taskfile.dev/schema.json": [
      "**/Taskfile.yml",
      "./path/to/any/other/taskfile.yml"
    ]
  }
}
```

Вы также можете настроить схему непосредственно внутри Taskfile, добавив следующий комментарий в начале файла:

```yaml
# yaml-language-server: $schema=https://taskfile.dev/schema.json
version: '3'
```

Вы можете найти дополнительную информацию об этом в [YAML language server project](https://github.com/redhat-developer/yaml-language-server).

## Интеграции сообщества

В дополнение к нашей официальной интеграции, сообщество разработчиков разработало свои собственные интеграции для Task:

- Расширение для [Sublime Text](https://packagecontrol.io/packages/Taskfile) [[источник](https://github.com/biozz/sublime-taskfile)] [@biozz](https://github.com/biozz)
- Расширение для [IntelliJ](https://plugins.jetbrains.com/plugin/17058-taskfile) [[источник](https://github.com/lechuckroh/task-intellij-plugin)] [@lechuckroh](https://github.com/lechuckroh)
- [mk](https://github.com/pycontribs/mk) - инструмент командной строки распознает Taskfile'ы.

Если вы сделали что-то, что интегрируется с Task, пожалуйста, не стесняйтесь открыть PR, чтобы добавить его в этот список.
