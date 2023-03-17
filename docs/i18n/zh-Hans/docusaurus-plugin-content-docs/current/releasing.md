---
slug: /releasing/
sidebar_position: 10
---

# 发布

Task 的发布流程是在 [GoReleaser](https://goreleaser.com/) 的帮助下完成的。 本地调用 Taskfile 的 `test-release` 任务可以测试发布流程。

[GitHub Actions](https://github.com/go-task/task/actions) 会在新 tag 推送到 master 分支的时候，自动发布产出物（原生的可执行文件、DEB 和 RPM 包）。

从 v3.15.0 开始，原始可执行文件也可以通过检查特定的标签并调用 `goreleaser build`，使用上述 GitHub Actions 中定义的 Go 版本，在本地进行复制和验证。

# Homebrew

Goreleaser 会自动向 [Homebrew tap](https://github.com/go-task/homebrew-tap) 仓库中的 [Formula/go-task.rb](https://github.com/go-task/homebrew-tap/blob/master/Formula/go-task.rb) 文件推送一个新的提交，以发布新的版本。

# npm

要发布到 npm ，请更新 [`package.json`](https://github.com/go-task/task/blob/master/package.json#L3) 文件中的版本，然后运行 `task npm:publish` 来推送它。

# Snapcraft

[snap package](https://github.com/go-task/snap) 发布新版本需要手动执行下面步骤：

* 更新 [snapcraft.yaml](https://github.com/go-task/snap/blob/master/snap/snapcraft.yaml#L2) 文件中的版本。
* 把新的 `amd64`, `armhf` 和 `arm64` 移动到 [Snapcraft dashboard](https://snapcraft.io/task/releases) 的稳定通道。

# Scoop

Scoop 是一个 Windows 系统的命令行包管理工具。 Scoop 的包清单由社区维护。 Scoop 的维护人通常会在 [这个文件](https://github.com/lukesampson/scoop-extras/blob/master/bucket/task.json) 里维护版本。 如果发现 Task 版本是旧的，请提交一个 Issue 通知我们。

# Nix

Nix 安装由社区维护。 Nix 包的维护人员通常会在 [这个文件](https://github.com/NixOS/nixpkgs/blob/nixos-unstable/pkgs/development/tools/go-task/default.nix) 里维护版本。 如果发现 Task 版本是旧的，请提交一个 Issue 通知我们。
