---
slug: /faq/
sidebar_position: 15
---

# 常见问题

此页面包含有关 Task 的常见问题列表。

## 为什么我的 task 不会更新我的 shell 环境？

这是 shell 工作方式的限制。 Task 作为当前 shell 的子进程运行，因此它不能更改启动它的 shell 的环境。 其他任务运行器和构建工具也有此限制。

解决此问题的一种常见方法是创建一个 task，该任务将生成可由您的 shell 解析的输出。 例如，要在 shell 上设置环境变量，您可以编写如下任务：

```yaml
my-shell-env:
  cmds:
    - echo "export FOO=foo"
    - echo "export BAR=bar"
```

现在运行 `eval $(task my-shell-env)` 变量 `$FOO` 和 `$BAR` 将在您的 shell 中可用。

## 我不能在多个命令中重用我的 shell

Task runs each command as a separate shell process, so something you do in one command won't effect any future commands. 比如：

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

对于更复杂的多行命令，建议将您的代码移动到个单独的文件，然后调用对应的脚本文件：

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

## 内置的 'x' 命令在 Windows 上不起作用

Windows 上的默认 shell（`cmd` 和 `powershell`）没有像 `rm` 和 `cp` 这样的内置命令。 这意味着这些命令将不起作用。 如果你想让你的 Taskfile 完全跨平台，你需要使用以下方法之一来解决这个限制：

- 使用 `{{OS}}` 函数运行特定于操作系统的脚本。
- 使用 `{{if eq OS "windows"}}powershell {{end}}<my_cmd>` 之类的东西来检测 windows 并直接在 Powershell 中运行命令。
- 在 Windows 上使用支持这些命令的 shell 作为内置命令，例如 [Git Bash][git-bash] 或 [WSL][wsl]。

我们希望对 Task 的这一部分进行改进，下面的 Issue 会跟踪这项工作。 非常欢迎建设性的意见和贡献！

- [#197](https://github.com/go-task/task/issues/197)
- [mvdan/sh#93](https://github.com/mvdan/sh/issues/93)
- [mvdan/sh#97](https://github.com/mvdan/sh/issues/97)

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[git-bash]: https://gitforwindows.org/
[wsl]: https://learn.microsoft.com/en-us/windows/wsl/install
