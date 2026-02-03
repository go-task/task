---
title: Installation
description: Installation methods for Task
outline: deep
---

# Installation

Task offers many installation methods. Check out the available methods below.

## Official Package Managers

These installation methods are maintained by the Task team and are always
up-to-date.

### [dnf](https://docs.fedoraproject.org/en-US/quick-docs/dnf) ![Fedora](https://img.shields.io/badge/Fedora-51A2DA?logo=fedora&logoColor=fff) ![CentOS](https://img.shields.io/badge/CentOS-002260?logo=centos&logoColor=F0F0F0) ![Fedora](https://img.shields.io/badge/Red_Hat-EE0000?logo=redhat&logoColor=white) {#dnf}

[[package](https://cloudsmith.io/~task/repos/task/packages/?sort=-format&q=format%3Arpm)]

If you Set up the repository by running :

```shell
curl -1sLf 'https://dl.cloudsmith.io/public/task/task/setup.rpm.sh' | sudo -E bash
```

Then you can install Task with:

```shell
dnf install task
```

### [apt](https://doc.ubuntu-fr.org/apt) ![Ubuntu](https://img.shields.io/badge/Ubuntu-E95420?logo=Ubuntu&logoColor=white) ![Debian](https://img.shields.io/badge/debian-red?logo=debian&logoColor=orange&color=darkred) ![Linux Mint](https://img.shields.io/badge/Linux%20Mint-87CF3E?logo=linuxmint&logoColor=fff) {#apt}

[[package](https://cloudsmith.io/~task/repos/task/packages/?sort=-format&q=format%3Adeb)]

If you Set up the repository by running:

```shell
curl -1sLf 'https://dl.cloudsmith.io/public/task/task/setup.deb.sh' | sudo -E bash
```

Then you can install Task with:

```shell
apt install task
```

### [Homebrew](https://brew.sh) ![macOS](https://img.shields.io/badge/MacOS-000000?logo=apple&logoColor=F0F0F0) ![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black) {#homebrew}

Task is available via our official Homebrew tap
[[source](https://github.com/go-task/homebrew-tap/blob/main/Formula/go-task.rb)]:

```shell
brew install go-task/tap/go-task
```

Alternatively it can be installed from the official Homebrew repository
[[package](https://formulae.brew.sh/formula/go-task)]
[[source](https://github.com/Homebrew/homebrew-core/blob/master/Formula/g/go-task.rb)]
by running:

```shell
brew install go-task
```

### [Snap](https://snapcraft.io/task) ![macOS](https://img.shields.io/badge/MacOS-000000?logo=apple&logoColor=F0F0F0) ![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black) {#snap}

Task is available on [Snapcraft](https://snapcraft.io/task)
[[source](https://github.com/go-task/snap/blob/main/snap/snapcraft.yaml)], but
keep in mind that your Linux distribution should allow classic confinement for
Snaps to Task work correctly:

```shell
sudo snap install task --classic
```

### [npm](https://www.npmjs.com) ![macOS](https://img.shields.io/badge/MacOS-000000?logo=apple&logoColor=F0F0F0) ![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black) ![Windows](https://custom-icon-badges.demolab.com/badge/Windows-0078D6?logo=windows11&logoColor=white) {#npm}

Npm can be used as cross-platform way to install Task globally or as a
dependency of your project
[[package](https://www.npmjs.com/package/@go-task/cli)]
[[source](https://github.com/go-task/task/blob/main/package.json)]:

```shell
npm install -g @go-task/cli
```

### [WinGet](https://github.com/microsoft/winget-cli) ![Windows](https://custom-icon-badges.demolab.com/badge/Windows-0078D6?logo=windows11&logoColor=white) {#winget}

Task is available via the
[community repository](https://github.com/microsoft/winget-pkgs)
[[source](https://github.com/microsoft/winget-pkgs/tree/master/manifests/t/Task/Task)]:

```shell
winget install Task.Task
```

## Community-Maintained Package Managers

::: warning Community Maintained

These installation methods are maintained by the community and may not always be
up-to-date with the latest Task version. The Task team does not directly control
these packages.

:::

### [Mise](https://mise.jdx.dev/) ![macOS](https://img.shields.io/badge/MacOS-000000?logo=apple&logoColor=F0F0F0) ![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black) ![Windows](https://custom-icon-badges.demolab.com/badge/Windows-0078D6?logo=windows11&logoColor=white) {#mise}

Mise is a cross-platform package manager that acts as a "frontend" to a variety
of other package managers "backends" such as `asdf`, `aqua` and `ubi`.

If using Mise, we recommend using the `aqua` or `ubi` backends to install Task
as these install directly from our GitHub releases.

::: code-group

```shell [aqua]
mise use -g aqua:go-task/task@latest
mise install
```

```shell [ubi]
mise use -g ubi:go-task/task
mise install
```

:::

### [Macports](https://macports.org) ![macOS](https://img.shields.io/badge/MacOS-000000?logo=apple&logoColor=F0F0F0) {#macports}

Task repository is tracked by Macports
[[package](https://ports.macports.org/port/go-task/details/)]
[[source](https://github.com/macports/macports-ports/blob/master/devel/go-task/Portfile)]:

```shell
port install go-task
```

### [pip](https://pip.pypa.io) ![macOS](https://img.shields.io/badge/MacOS-000000?logo=apple&logoColor=F0F0F0) ![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black) ![Windows](https://custom-icon-badges.demolab.com/badge/Windows-0078D6?logo=windows11&logoColor=white) {#pip}

Like npm, pip can be used as a cross-platform way to install Task
[[package](https://pypi.org/project/go-task-bin)]
[[source](https://github.com/Bing-su/pip-binary-factory/tree/main/task)]:

```shell
pip install go-task-bin
```

### [Chocolatey](https://chocolatey.org) ![Windows](https://custom-icon-badges.demolab.com/badge/Windows-0078D6?logo=windows11&logoColor=white) {#chocolatey}

[[package](https://community.chocolatey.org/packages/go-task)]
[[source](https://github.com/Starz0r/ChocolateyPackagingScripts/blob/master/src/go-task_gh_build.py)]

```shell
choco install go-task
```

### [Scoop](https://scoop.sh) ![Windows](https://custom-icon-badges.demolab.com/badge/Windows-0078D6?logo=windows11&logoColor=white) {#scoop}

[[source](https://github.com/ScoopInstaller/Main/blob/master/bucket/task.json)]

```shell
scoop install task
```

### Arch ([pacman](https://wiki.archlinux.org/title/Pacman)) ![Arch Linux](https://img.shields.io/badge/Arch%20Linux-1793D1?logo=arch-linux&logoColor=fff) {#arch}

[[package](https://archlinux.org/packages/extra/x86_64/go-task/)]
[[source](https://gitlab.archlinux.org/archlinux/packaging/packages/go-task)]

```shell
pacman -S go-task
```

### Fedora ([dnf](https://docs.fedoraproject.org/en-US/quick-docs/dnf)) ![Fedora](https://img.shields.io/badge/Fedora-51A2DA?logo=fedora&logoColor=fff) {#fedora-community}

[[package](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/)]
[[source](https://src.fedoraproject.org/rpms/golang-github-task)]

```shell
dnf install go-task
```

### FreeBSD ([Ports](https://ports.freebsd.org/cgi/ports.cgi)) ![FreeBSD](https://img.shields.io/badge/FreeBSD-990000?logo=freebsd&logoColor=fff) {#freebsd}

[[package](https://cgit.freebsd.org/ports/tree/devel/task)]
[[source](https://cgit.freebsd.org/ports/tree/devel/task/Makefile)]

```shell
pkg install task
```

### [Nix](https://nixos.org) ![Nix](https://img.shields.io/badge/Nix-5277C3?logo=nixos&logoColor=fff) ![NixOS](https://img.shields.io/badge/NixOS-5277C3?logo=nixos&logoColor=fff) ![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black) ![macOS](https://img.shields.io/badge/MacOS-000000?logo=apple&logoColor=F0F0F0) {#nix}

[[source](https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/go/go-task/package.nix)]

```shell
nix-env -iA nixpkgs.go-task
```

### [pacstall](https://github.com/pacstall/pacstall) ![Debian](https://img.shields.io/badge/Debian-A81D33?logo=debian&logoColor=fff) ![Ubuntu](https://img.shields.io/badge/Ubuntu-E95420?logo=ubuntu&logoColor=fff) {#pacstall}

[[package](https://pacstall.dev/packages/go-task-deb)]
[[source](https://github.com/pacstall/pacstall-programs/blob/master/packages/go-task-deb/go-task-deb.pacscript)]

```shell
pacstall -I go-task-deb
```

### [pkgx](https://pkgx.sh) ![macOS](https://img.shields.io/badge/MacOS-000000?logo=apple&logoColor=F0F0F0) ![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black) {#pkgx}

[[package](https://pkgx.dev/pkgs/taskfile.dev)]
[[source](https://github.com/pkgxdev/pantry/blob/main/projects/taskfile.dev/package.yml)]

```shell
pkgx task
```

or, if you have pkgx integration enabled:

```shell
task
```

## Get The Binary

### Binary

You can download the binary from the
[releases page on GitHub](https://github.com/go-task/task/releases) and add to
your `$PATH`.

DEB and RPM packages are also available.

The `task_checksums.txt` file contains the SHA-256 checksum for each file.

### Install Script

We also have an
[install script](https://github.com/go-task/task/blob/main/install-task.sh)
which is very useful in scenarios like CI. Many thanks to
[GoDownloader](https://github.com/goreleaser/godownloader) for enabling the easy
generation of this script.

By default, it installs on the `./bin` directory relative to the working
directory:

```shell
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d
```

It is possible to override the installation directory with the `-b` parameter.
On Linux, common choices are `~/.local/bin` and `~/bin` to install for the
current user or `/usr/local/bin` to install for all users:

```shell
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
```

::: warning

On macOS and Windows, `~/.local/bin` and `~/bin` are not added to `$PATH` by
default.

:::

By default, it installs the latest version available. You can also specify a tag
(available in [releases](https://github.com/go-task/task/releases)) to install a
specific version:

```shell
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d v3.36.0
```

Parameters are order specific, to set both installation directory and version:

```shell
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin v3.42.1
```

### GitHub Actions

We have an [official GitHub Action](https://github.com/go-task/setup-task) to
install Task in your GitHub workflows. This repository is forked from the
fantastic project by the Arduino team. Check out the repository for more
examples and configuration.

```yaml
- name: Install Task
  uses: go-task/setup-task@v1
```

## Build From Source

### Go Modules

Ensure that you have a supported version of [Go](https://golang.org) properly
installed and setup. You can find the minimum required version of Go in the
[go.mod](https://github.com/go-task/task/blob/main/go.mod#L3) file.

You can then install the latest release globally by running:

```shell
go install github.com/go-task/task/v3/cmd/task@latest
```

Or you can install into another directory:

```shell
env GOBIN=/bin go install github.com/go-task/task/v3/cmd/task@latest
```

::: tip

For CI environments we recommend using the [install script](#install-script)
instead, which is faster and more stable, since it'll just download the latest
released binary.

:::

## Setup completions

Some installation methods will automatically install completions too, but if
this isn't working for you or your chosen method doesn't include them, you can
run `task --completion <shell>` to output a completion script for any supported
shell. There are a couple of ways these completions can be added to your shell
config:

### Option 1. Load the completions in your shell's startup config (Recommended)

This method loads the completion script from the currently installed version of
task every time you create a new shell. This ensures that your completions are
always up-to-date.
If your executable isn’t named task, set the `TASK_EXE` environment variable before running eval.

::: code-group

```shell [bash]
# ~/.bashrc

# export TASK_EXE='go-task' if needed
eval "$(task --completion bash)"
```

```shell [zsh]
# ~/.zshrc

# export TASK_EXE='go-task' if needed
eval "$(task --completion zsh)"
```

```shell [fish]
# ~/.config/fish/config.fish

# export TASK_EXE='go-task' if needed
task --completion fish | source
```

```powershell [powershell]
# $PROFILE\Microsoft.PowerShell_profile.ps1
Invoke-Expression  (&task --completion powershell | Out-String)
```

:::

### Option 2. Copy the script to your shell's completions directory

This method requires you to manually update the completions whenever Task is
updated. However, it is useful if you want to modify the completions yourself.

::: code-group

```shell [bash]
task --completion bash > /etc/bash_completion.d/task
```

```shell [zsh]
task --completion zsh  > /usr/local/share/zsh/site-functions/_task
```

```shell [fish]
task --completion fish > ~/.config/fish/completions/task.fish
```

:::

### Zsh customization

The Zsh completion supports the standard `verbose` zstyle to control whether task
descriptions are shown. By default, descriptions are displayed. To show only task
names without descriptions, add this to your `~/.zshrc` (after the completion is loaded):

```shell
zstyle ':completion:*:*:task:*' verbose false
```
