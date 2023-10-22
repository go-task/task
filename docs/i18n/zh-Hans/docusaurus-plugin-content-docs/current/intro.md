---
slug: /
sidebar_position: 1
title: 主页
---

# Task

<div align="center">
  <img id="logo" src="img/logo.svg" height="250px" width="250px" />
</div>

Task 是一个任务运行器/构建工具，旨在比 [GNU Make][make] 等更简单易用。

由于它是用 [Go](https://go.dev/) 编写的，Task 只是一个二进制文件，没有其他依赖项，这意味着您不需要为了使用构建工具而烦恼任何复杂的安装设置。

[安装](installation.md) 后，您只需在名为 `Taskfile.yml` 的文件中使用简单的 [YAML][yaml] 规则描述您的构建任务：

```yaml title="Taskfile.yml"
version: '3'

tasks:
  hello:
    cmds:
      - echo 'Hello World from Task!'
    silent: true
```

然后通过从您的终端运行 `task hello` 来调用它。

上面的示例只是一个开始，您可以查看 [使用指南](/usage) 以检查完整的规则文档和 Task 功能。

## Features

- [易于安装](installation.md)：只需要下载一个二进制文件，添加到 `$PATH` 即可！ 或者，您也可以根据需要使用 [Homebrew](https://brew.sh/)、[Snapcraft](https://snapcraft.io/) 或 [Scoop](https://scoop.sh/) 进行安装。
- 可以在 CI 中使用：只要添加 [这个命令](installation.md#安装脚本) 到 CI 安装脚本中，然后就可以把 Task 当做 CI 的一个功能来使用了。
- 真正的跨平台：虽然大多数构建工具只能在 Linux 或 macOS 上运行良好，但由于 [这个用于 Go 的 shell 解释器](https://github.com/mvdan/sh)，Task 也支持 Windows。
- 非常适合代码生成：如果给定的一组文件自上次运行以来没有更改（基于其时间戳或内容），您可以轻松地 [阻止 task 运行](/usage#减少不必要的工作)。

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[make]: https://www.gnu.org/software/make/
[yaml]: http://yaml.org/
