---
slug: /faq/
sidebar_position: 15
---

# ЧАВО

Эта страница содержит список часто задаваемых вопросов о Task.

## Почему task не обновляет мои переменные среды оболочки?

Это ограничение работы оболочек. Task запускается как подпроцесс вашей текущей оболочки, поэтому он не может сменить переменные среды оболочки, которая запустила его. Это ограничение есть и в других task runners и инструментах сборки тоже.

Самый простой способ обойти это - создать задачу, генерирующую вывод, который может быть проанализирован вашей оболочкой. Например, чтобы установить переменные среды в вашей оболочке, вы можете написать task, похожую на:

```yaml
my-shell-env:
  cmds:
    - echo "export FOO=foo"
    - echo "export BAR=bar"
```

Теперь запустите `eval $(task my-shell-env)`, после этого переменные `$FOO` и `$BAR` будут доступны в вашей оболочке.

## Я не могу переиспользовать свою оболочку в командах task's

Task запускает каждую команду в качестве отдельного процесса оболочки, поэтому действия в одной команде не повлияют на другие команды. Например, это не сработает:

```yaml
version: '3'

tasks:
  foo:
    cmds:
      - a=foo
      - echo $a
      # outputs ""
```

Чтобы обойти это, вы можете использовать многострочную команду:

```yaml
version: '3'

tasks:
  foo:
    cmds:
      - |
        a=foo
        echo $a
      # outputs "foo"
```

Или для более сложных многострочных команд рекомендуется перенести ваш код в отдельный файл, и вызвать его вместо команды:

```yaml
version: '3'

tasks:
  foo:
    cmds:
      - ./foo-printer.bash
```

```bash
#!/bin/bash
a=foo
echo $a
```

## 'x' builtin command doesn't work on Windows

The default shell on Windows (`cmd` and `powershell`) do not have commands like `rm` and `cp` available as builtins. This means that these commands won't work. If you want to make your Taskfile fully cross-platform, you'll need to work around this limitation using one of the following methods:

- Use the `{{OS}}` function to run an OS-specific script.
- Use something like `{{if eq OS "windows"}}powershell {{end}}<my_cmd>` to detect windows and run the command in Powershell directly.
- Use a shell on Windows that supports these commands as builtins, such as [Git Bash][git-bash] or [WSL][wsl].

We want to make improvements to this part of Task and the issues below track this work. Constructive comments and contributions are very welcome!

- [#197](https://github.com/go-task/task/issues/197)
- [mvdan/sh#93](https://github.com/mvdan/sh/issues/93)
- [mvdan/sh#97](https://github.com/mvdan/sh/issues/97)

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[git-bash]: https://gitforwindows.org/
[wsl]: https://learn.microsoft.com/en-us/windows/wsl/install
