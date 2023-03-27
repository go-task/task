---
slug: /community/
sidebar_position: 8
---

# 社区

一些改善 Task 生态的工作是由社区完成，包括安装方法或代码编辑器集成。 我（指作者）非常感谢所有帮助提升整体体验的人们。

## 翻译

[@DeronW](https://github.com/DeronW) 在 [此存储库](https://github.com/DeronW/task) 中维护网站的 [中文翻译](https://task-zh.readthedocs.io/zh_CN/latest/)。

## 编辑器集成

### JSON Schema

Schema 的初步工作是由 [@KROSF](https://github.com/KROSF) 在此 [Gist](https://gist.github.com/KROSF/c5435acf590acd632f71bb720f685895) 上完成的。 这个 Schema 目前在 https://taskfile.dev/schema.json 上可用，并在 https://json.schemastore.org/taskfile.json 上添加了链接，因此它可以自动在许多代码编辑器使用，例如 VSCode。 可以通过编辑 [此文件](https://github.com/go-task/task/blob/master/docs/static/schema.json) 来完成贡献。

### Visual Studio Code 扩展

另外，在开发 Visual Studio Code 扩展过程中， 还有一些工作由 [@paulvarache](https://github.com/paulvarache) 完成， 代码在 [这里](https://github.com/paulvarache/vscode-taskfile) 并发布到了 [这里](https://marketplace.visualstudio.com/items?itemName=paulvarache.vscode-taskfile)。

### Sublime Text 4 包

通过 Sublime Text 的命令面板有一个简便的安装运行方法。 这个包是由 [@biozz](https://github.com/biozz) 开发的， 源代码在 [这里](https://github.com/biozz/sublime-taskfile) 并且发布到了包管理 [这里](https://packagecontrol.io/packages/Taskfile)。

### IntelliJ 插件

JetBrains IntelliJ 插件由 [@lechuckroh](https://github.com/lechuckroh) 完成， 代码在 [这里](https://github.com/lechuckroh/task-intellij-plugin) 并且发布到了 [这里](https://plugins.jetbrains.com/plugin/17058-taskfile)。

## 其他集成

- [mk](https://github.com/pycontribs/mk) 命令行工具可以原生识别任务文件。

## 安装方法

有些安装方式是通过第三方维护的：

- [GitHub Actions](https://github.com/arduino/setup-task) 由 [@arduino](https://github.com/arduino) 维护
- [AUR](https://aur.archlinux.org/packages/go-task-bin) 由 [@carlsmedstad](https://github.com/carlsmedstad) 维护
- [Scoop](https://github.com/ScoopInstaller/Main/blob/master/bucket/task.json)
- [Fedora](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/)
- [NixOS](https://github.com/NixOS/nixpkgs/blob/master/pkgs/development/tools/go-task/default.nix)

## 更多

同时，感谢所有 [代码贡献者](https://github.com/go-task/task/graphs/contributors)， [资金赞助](https://opencollective.com/task)，以及 [提交问题](https://github.com/go-task/task/issues?q=is%3Aissue) 和 [解答问题](https://github.com/go-task/task/discussions) 的人。

如果你发现文档有哪些遗漏信息，欢迎提交 pull request。
