---
slug: /installation/
sidebar_position: 2
---

# 安装

Task 提供以下多种安装方式。 查看以下可用方法。

## 包管理工具

### Homebrew

如果您使用的是 macOS 或 Linux 并安装了 [Homebrew](https://brew.sh/)，获取 Task 就像运行以下命令一样简单：

```bash
brew install go-task/tap/go-task
```

The above Formula is [maintained by ourselves](https://github.com/go-task/homebrew-tap/blob/main/Formula/go-task.rb).

最近，[官方 Homebrew 存储库](https://formulae.brew.sh/formula/go-task) 中也提供了 Task，因此如果您愿意，也可以使用该选项：

```bash
brew install go-task
```

### Tea

If you're on macOS or Linux and have [tea][tea] installed, getting Task is as simple as running:

```bash
tea task
```

or, if you have tea’s magic enabled:

```bash
task
```
这种安装方式是社区维护的。 After a new release of Task, they are automatically released by tea in a minimum of time.

### Snap

Task 在 [Snapcraft][snapcraft] 中可用，但请记住，您的 Linux 发行版应该符合 Snaps 的基本要求才能正确安装：

```bash
sudo snap install task --classic
```

### Chocolatey

如果 Windows 上安装了 [Chocolatey][choco]，再安装 Task 只要这样：

```bash
choco install go-task
```

这种安装方式是社区维护的。

### Scoop

如果 Windows 上安装了 [Scoop][scoop]，再安装 Task 只要这样：

```cmd
scoop install task
```

This installation method is community owned. 新版 Task 发布后，需要过一段时间才能通过 Scoop 安装。

### AUR

如果你使用的是 Arch Linux，你可以使用你最喜欢的包管理器（例如 `yay`、`pacaur` 或 `yaourt`）从 [AUR](https://aur.archlinux.org/packages/go-task-bin) 安装 Task：

```cmd
yay -S go-task-bin
```

或者，有一个从源代码安装的 [软件包](https://aur.archlinux.org/packages/go-task)，而不是从 [发布页面](https://github.com/go-task/task/releases) 下载二进制文件：

```cmd
yay -S go-task
```

这种安装方式是社区维护的。

### Fedora

如果您使用的是 Fedora Linux，则可以使用 `dnf` 从 [官方 Fedora 存储库](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/) 安装 Task：

```cmd
sudo dnf install go-task
```

这种安装方式是社区维护的。 新版 Task 发布后，需要一段时间才能通过 [Fedora](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/) 安装。

### Nix

如果您使用的是 NixOS 或安装了 Nix，则可以从 [nixpkgs](https://github.com/NixOS/nixpkgs) 安装 Task：

```cmd
nix-env -iA nixpkgs.go-task
```

这种安装方式是社区维护的。 新版本的 Task 发布后，可能需要一些时间才能在 [nixpkgs](https://github.com/NixOS/nixpkgs) 中可用。

### npm

您也可以通过使用 Node 和 npm 安装 [此包](https://www.npmjs.com/package/@go-task/cli) 来安装 Task。

```bash
npm install -g @go-task/cli
```

### Winget

如果您正在使用 Windows 并且安装了 [winget](https://github.com/microsoft/winget-cli) 软件包管理工具，您可以从 [winget-pkgs](https://github.com/microsoft/winget-pkgs) 安装 Task。

```bash
winget install Task.Task
```

## 获取二进制文件

### 二进制文件

您可以从 [GitHub 上的发布页面][releases] 下载二进制文件并添加到您的 `$PATH` 中。

还支持 DEB 和 RPM 包。

`task_checksums.txt` 文件包含每个文件的 SHA-256 checksum。

### 安装脚本

我们还有一个 [安装脚本][installscript]，它在 CI 等场景中非常有用。 非常感谢 [GoDownloader](https://github.com/goreleaser/godownloader) 使这个脚本的生成变得容易。

默认情况下，它安装在相对于工作目录的 `./bin` 目录中：

```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d
```

可以使用 `-b` 参数覆盖安装目录。 通过 `-b` 参数可以自定义安装目录，在 Linux 中当前用户安装一般会选择 `~/.local/bin` 或 `~/bin`， 全局用户安装会选择 `/usr/local/bin`：

```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
```

:::caution

在 macOS 和 Windows 上，`~/.local/bin` 和 `~/bin` 默认情况下不会添加到 `$PATH`。

:::

### GitHub Actions

如果你想在 GitHub Actions 中安装 Task，你可以尝试使用 Arduino 团队的 [这个 action](https://github.com/arduino/setup-task)：

```yaml
- name: Install Task
  uses: arduino/setup-task@v1
  with:
    version: 3.x
    repo-token: ${{ secrets.GITHUB_TOKEN }}
```

This installation method is community owned.

## 从源码构建

### Go Modules

确保您已正确安装和设置受支持的 [Go][go] 版本。 您可以在 [go.mod](https://github.com/go-task/task/blob/main/go.mod#L3) 文件中找到最低要求的 Go 版本。

然后，您可以通过运行以下命令全局安装最新版本：

```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

或者你可以安装到另一个目录：

```bash
env GOBIN=/bin go install github.com/go-task/task/v3/cmd/task@latest
```

:::tip

对于 CI 环境，我们建议改用 [安装脚本](#安装脚本)，它更快更稳定，因为它只会下载最新发布的二进制文件。

:::

## 自动完成

下载与您的 shell 对应的自动完成文件。

[所有自动完成都在 Task 存储库中可用](https://github.com/go-task/task/tree/main/completion)。

### Bash

首先，确认你通过包管理安装了 bash-completion。

使完成文件可执行：

```
chmod +x path/to/task.bash
```

然后在 `~/.bash_profile` 文件中添加：

```shell
source path/to/task.bash
```

### ZSH

把 `_task` 文件放到你的 `$FPATH` 中：

```shell
mv path/to/_task /usr/local/share/zsh/site-functions/_task
```

在 `~/.zshrc` 文件中添加：

```shell
autoload -U compinit
compinit -i
```

建议使用 ZSH 5.7 或更高版本。

### Fish

移动 `task.fish` 完成脚本：

```shell
mv path/to/task.fish ~/.config/fish/completions/task.fish
```

### PowerShell

使用以下命令打开您的配置文件脚本：

```
mkdir -Path (Split-Path -Parent $profile) -ErrorAction SilentlyContinue
notepad $profile
```

添加这行并保存文件：

```shell
Invoke-Expression -Command path/to/task.ps1
```

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[go]: https://golang.org/
[snapcraft]: https://snapcraft.io/task
[installscript]: https://github.com/go-task/task/blob/main/install-task.sh
[releases]: https://github.com/go-task/task/releases
[choco]: https://chocolatey.org/
[scoop]: https://scoop.sh/
[tea]: https://tea.xyz/
