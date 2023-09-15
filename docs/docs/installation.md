---
slug: /installation/
sidebar_position: 2
---

# Installation

Task offers many installation methods. Check out the available methods below.

## Package Managers

### Homebrew

If you're on macOS or Linux and have [Homebrew][homebrew] installed, getting
Task is as simple as running:

```bash
brew install go-task/tap/go-task
```

The above Formula is
[maintained by ourselves](https://github.com/go-task/homebrew-tap/blob/main/Formula/go-task.rb).

Recently, Task was also made available
[on the official Homebrew repository](https://formulae.brew.sh/formula/go-task),
so you also have that option if you prefer:

```bash
brew install go-task
```

### Tea

If you're on macOS or Linux and have [tea][tea] installed, getting
Task is as simple as running:

```bash
tea task
```

or, if you have tea’s magic enabled:

```bash
task
```
This installation method is community owned. After a new release of Task, they
are automatically released by tea in a minimum of time.

### Snap

Task is available in [Snapcraft][snapcraft], but keep in mind that your Linux
distribution should allow classic confinement for Snaps to Task work right:

```bash
sudo snap install task --classic
```

### Chocolatey

If you're on Windows and have [Chocolatey][choco] installed, getting Task is as
simple as running:

```bash
choco install go-task
```

This installation method is community owned.

### Scoop

If you're on Windows and have [Scoop][scoop] installed, getting Task is as
simple as running:

```cmd
scoop install task
```

This installation method is community owned. After a new release of Task, it may
take some time until it's available on Scoop.

### AUR

If you're on Arch Linux you can install Task from
[AUR](https://aur.archlinux.org/packages/go-task-bin) using your favorite
package manager such as `yay`, `pacaur` or `yaourt`:

```cmd
yay -S go-task-bin
```

Alternatively, there's
[this package](https://aur.archlinux.org/packages/go-task) which installs from
the source code instead of downloading the binary from the
[releases page](https://github.com/go-task/task/releases):

```cmd
yay -S go-task
```

This installation method is community owned.

### Fedora

If you're on Fedora Linux you can install Task from the official
[Fedora](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/)
repository using `dnf`:

```cmd
sudo dnf install go-task
```

This installation method is community owned. After a new release of Task, it may
take some time until it's available in
[Fedora](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/).

### Nix

If you're on NixOS or have Nix installed you can install Task from
[nixpkgs](https://github.com/NixOS/nixpkgs):

```cmd
nix-env -iA nixpkgs.go-task
```

This installation method is community owned. After a new release of Task, it may
take some time until it's available in
[nixpkgs](https://github.com/NixOS/nixpkgs).

### npm

You can also use Node and npm to install Task by installing
[this package](https://www.npmjs.com/package/@go-task/cli).

```bash
npm install -g @go-task/cli
```

### Winget

If you are using Windows and installed the
[winget](https://github.com/microsoft/winget-cli) package management tool, you
can install Task from [winget-pkgs](https://github.com/microsoft/winget-pkgs).

```bash
winget install Task.Task
```

## Get The Binary

### Binary

You can download the binary from the [releases page on GitHub][releases] and add
to your `$PATH`.

DEB and RPM packages are also available.

The `task_checksums.txt` file contains the SHA-256 checksum for each file.

### Install Script

We also have an [install script][installscript] which is very useful in
scenarios like CI. Many thanks to [GoDownloader][godownloader] for enabling the
easy generation of this script.

By default, it installs on the `./bin` directory relative to the working
directory:

```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d
```

It is possible to override the installation directory with the `-b` parameter.
On Linux, common choices are `~/.local/bin` and `~/bin` to install for the
current user or `/usr/local/bin` to install for all users:

```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
```

:::caution

On macOS and Windows, `~/.local/bin` and `~/bin` are not added to `$PATH` by
default.

:::

### GitHub Actions

If you want to install Task in GitHub Actions you can try using
[this action](https://github.com/arduino/setup-task) by the Arduino team:

```yaml
- name: Install Task
  uses: arduino/setup-task@v1
  with:
    version: 3.x
    repo-token: ${{ secrets.GITHUB_TOKEN }}
```

This installation method is community owned.

## Build From Source

### Go Modules

Ensure that you have a supported version of [Go][go] properly installed and
setup. You can find the minimum required version of Go in the
[go.mod](https://github.com/go-task/task/blob/main/go.mod#L3) file.

You can then install the latest release globally by running:

```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

Or you can install into another directory:

```bash
env GOBIN=/bin go install github.com/go-task/task/v3/cmd/task@latest
```

:::tip

For CI environments we recommend using the [install script](#install-script)
instead, which is faster and more stable, since it'll just download the latest
released binary.

:::

## Setup completions

Download the autocompletion file corresponding to your shell.

[All completions are available on the Task repository](https://github.com/go-task/task/tree/main/completion).

### Bash

First, ensure that you installed bash-completion using your package manager.

Make the completion file executable:

```
chmod +x path/to/task.bash
```

After, add this to your `~/.bash_profile`:

```shell
source path/to/task.bash
```

### ZSH

Put the `_task` file somewhere in your `$FPATH`:

```shell
mv path/to/_task /usr/local/share/zsh/site-functions/_task
```

Ensure that the following is present in your `~/.zshrc`:

```shell
autoload -U compinit
compinit -i
```

ZSH version 5.7 or later is recommended.

### Fish

Move the `task.fish` completion script:

```shell
mv path/to/task.fish ~/.config/fish/completions/task.fish
```

### PowerShell

Open your profile script with:

```
mkdir -Path (Split-Path -Parent $profile) -ErrorAction SilentlyContinue
notepad $profile
```

Add the line and save the file:

```shell
Invoke-Expression -Command path/to/task.ps1
```

<!-- prettier-ignore-start -->
[go]: https://golang.org/
[snapcraft]: https://snapcraft.io/task
[homebrew]: https://brew.sh/
[installscript]: https://github.com/go-task/task/blob/main/install-task.sh
[releases]: https://github.com/go-task/task/releases
[godownloader]: https://github.com/goreleaser/godownloader
[choco]: https://chocolatey.org/
[scoop]: https://scoop.sh/
[tea]: https://tea.xyz/
<!-- prettier-ignore-end -->
