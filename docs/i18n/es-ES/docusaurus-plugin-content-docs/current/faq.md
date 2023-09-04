---
slug: /faq/
sidebar_position: 15
---

# FAQ

This page contains a list of frequently asked questions about Task.

## Why won't my task update my shell environment?

This is a limitation of how shells work. Task runs as a subprocess of your current shell, so it can't change the environment of the shell that started it. This limitation is shared by other task runners and build tools too.

A common way to work around this is to create a task that will generate output that can be parsed by your shell. For example, to set an environment variable on your shell you can write a task like this:

```yaml
my-shell-env:
  cmds:
    - echo "export FOO=foo"
    - echo "export BAR=bar"
```

Now run `eval $(task my-shell-env)` and the variables `$FOO` and `$BAR` will be available in your shell.

## I can't reuse my shell in a task's commands

Task runs each command as a separate shell process, so something you do in one command won't effect any future commands. For example, this won't work:

```yaml
version: '3'

tasks:
  foo:
    cmds:
      - a=foo
      - echo $a
      # outputs ""
```

To work around this you can either use a multiline command:

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

Or for more complex multi-line commands it is recommended to move your code into a separate file and call that instead:

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
