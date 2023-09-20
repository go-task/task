---
slug: /releasing/
sidebar_position: 13
---

# 发布

Task 的发布流程是在 [GoReleaser][goreleaser] 的帮助下完成的。 本地调用 Taskfile 的 `test-release` task 可以测试发布流程。

[GitHub Actions](https://github.com/go-task/task/actions) 会在新 tag 推送到 `main` 分支的时候，自动发布产出物（原生的可执行文件、DEB 和 RPM 包）。

从 v3.15.0 开始，原始可执行文件也可以通过检查特定的标签并调用 `goreleaser build`，使用上述 GitHub Actions 中定义的 Go 版本，在本地进行复制和验证。

# Homebrew

Goreleaser will automatically push a new commit to the [Formula/go-task.rb][gotaskrb] file in the [Homebrew tap][homebrewtap] repository to release the new version.

# npm

要发布到 npm ，请更新 [`package.json`][packagejson] 文件中的版本，然后运行 `task npm:publish` 来推送它。

# Snapcraft

[snap package](https://github.com/go-task/snap) 发布新版本需要手动执行下面步骤：

- Updating the current version on [snapcraft.yaml][snapcraftyaml].
- 把新的 `amd64`, `armhf` 和 `arm64` 移动到 [Snapcraft dashboard][snapcraftdashboard] 的稳定通道。

# winget

winget also requires manual steps to be completed. By running `task test-release` locally, manifest files will be generated on `dist/winget/manifests/t/Task/Task/v{version}`. [Upload the manifest directory into this fork](https://github.com/go-task/winget-pkgs/tree/master/manifests/t/Task/Task) and open a pull request into [this repository](https://github.com/microsoft/winget-pkgs).

# Scoop

Scoop 是一个 Windows 系统的命令行包管理工具。 Scoop 的包清单由社区维护。 Scoop 的维护人通常会在 [这个文件](https://github.com/lukesampson/scoop-extras/blob/master/bucket/task.json) 里维护版本。 If you think its Task version is outdated, open an issue to let us know.

# Nix

Nix 安装由社区维护。 Nix 包的维护人员通常会在 [这个文件](https://github.com/NixOS/nixpkgs/blob/nixos-unstable/pkgs/development/tools/go-task/default.nix) 里维护版本。 如果发现 Task 版本是旧的，请提交一个 Issue 通知我们。

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[goreleaser]: https://goreleaser.com/
[homebrewtap]: https://github.com/go-task/homebrew-tap
[gotaskrb]: https://github.com/go-task/homebrew-tap/blob/main/Formula/go-task.rb
[packagejson]: https://github.com/go-task/task/blob/main/package.json#L3
[snapcraftyaml]: https://github.com/go-task/snap/blob/main/snap/snapcraft.yaml#L2
[snapcraftdashboard]: https://snapcraft.io/task/releases
