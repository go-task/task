---
title: Installation
description: Installation methods for Task
outline: deep
pageClass: installation-page
---

<script setup>
import InstallationMethods from '../../../.vitepress/components/InstallationMethods.vue';
</script>

# Installation

Install Task with your preferred package manager, download a binary, or set it
up in CI. Then add shell completions and you're ready to go.

<nav class="installation-shortcuts" aria-label="Installation shortcuts">
  <a href="#official-package-managers"><span>01</span><strong>Package managers</strong><small>Find your preferred method</small></a>
  <a href="#get-the-binary"><span>02</span><strong>Binaries &amp; CI</strong><small>Downloads, scripts and actions</small></a>
  <a href="#setup-completions"><span>03</span><strong>Shell completions</strong><small>Finish your setup</small></a>
</nav>

<InstallationMethods>

## Official packages {#official-package-managers}

These installation methods are maintained by the Task team and are always
up-to-date.

<div class="install-method" data-platforms="macos">

### [Homebrew](https://brew.sh) {#homebrew}

<div class="install-meta">

<p class="install-platforms">macOS</p>

<div class="install-links">

[Tap source](https://github.com/go-task/homebrew-tap/blob/main/Casks/go-task.rb)
[Homebrew package](https://formulae.brew.sh/formula/go-task)
[Formula source](https://github.com/Homebrew/homebrew-core/blob/master/Formula/g/go-task.rb)

</div>

</div>

Task is available via our official Homebrew tap:

```shell
brew install go-task/tap/go-task
```

Alternatively it can be installed from the official Homebrew repository by
running:

```shell
brew install go-task
```

</div>

<div class="install-method" data-platforms="linux">

### [dnf](https://docs.fedoraproject.org/en-US/quick-docs/dnf) {#dnf}

<div class="install-meta">

<p class="install-platforms">Fedora · CentOS · Red Hat</p>

<div class="install-links">

[View package](https://cloudsmith.io/~task/repos/task/packages/?sort=-format&q=format%3Arpm)

</div>

</div>

Set up the repository:

```shell
curl -1sLf 'https://dl.cloudsmith.io/public/task/task/setup.rpm.sh' | sudo -E bash
```

Then you can install Task with:

```shell
dnf install task
```

</div>

<div class="install-method" data-platforms="linux">

### [apt](https://doc.ubuntu-fr.org/apt) {#apt}

<div class="install-meta">

<p class="install-platforms">Ubuntu · Debian · Linux Mint</p>

<div class="install-links">

[View package](https://cloudsmith.io/~task/repos/task/packages/?sort=-format&q=format%3Adeb)

</div>

</div>

Set up the repository:

```shell
curl -1sLf 'https://dl.cloudsmith.io/public/task/task/setup.deb.sh' | sudo -E bash
```

Then you can install Task with:

```shell
apt install task
```

</div>

<div class="install-method" data-platforms="linux">

### [apk](https://wiki.alpinelinux.org/wiki/Alpine_Package_Keeper) {#apk}

<div class="install-meta">

<p class="install-platforms">Alpine Linux</p>

<div class="install-links">

[View package](https://cloudsmith.io/~task/repos/task/packages/?sort=-format&q=format%3Aalpine)

</div>

</div>

Set up the repository by running:

```shell
curl -1sLf 'https://dl.cloudsmith.io/public/task/task/setup.alpine.sh' | sudo -E bash
```

Then you can install Task with:

```shell
apk add task
```

</div>

<div class="install-method" data-platforms="linux">

### [Snap](https://snapcraft.io/task) {#snap}

<div class="install-meta">

<p class="install-platforms">Linux</p>

<div class="install-links">

[Source](https://github.com/go-task/snap/blob/main/snap/snapcraft.yaml)

</div>

</div>

Task is available on [Snapcraft](https://snapcraft.io/task), but keep in mind
that your Linux distribution should allow classic confinement for Snaps to Task
work correctly:

```shell
sudo snap install task --classic
```

</div>

<div class="install-method" data-platforms="macos linux windows">

### [npm](https://www.npmjs.com) {#npm}

<div class="install-meta">

<p class="install-platforms">macOS · Linux · Windows</p>

<div class="install-links">

[View package](https://www.npmjs.com/package/@go-task/cli)
[Source](https://github.com/go-task/task/blob/main/package.json)

</div>

</div>

Npm can be used as cross-platform way to install Task globally or as a
dependency of your project:

```shell
npm install -g @go-task/cli
```

</div>

<div class="install-method" data-platforms="windows">

### [WinGet](https://github.com/microsoft/winget-cli) {#winget}

<div class="install-meta">

<p class="install-platforms">Windows</p>

<div class="install-links">

[Source](https://github.com/microsoft/winget-pkgs/tree/master/manifests/t/Task/Task)

</div>

</div>

Task is available via the
[community repository](https://github.com/microsoft/winget-pkgs):

```shell
winget install Task.Task
```

</div>

<p class="install-hosting">Package repository hosting for deb/rpm/apk is graciously provided by <a href="https://cloudsmith.com">Cloudsmith</a>.</p>

## Community packages {#community-maintained-package-managers}

These installation methods are maintained by the community and may not always be
up-to-date with the latest Task version. The Task team does not directly control
these packages.

<div class="install-method" data-platforms="macos linux windows">

### [Mise](https://mise.jdx.dev/) {#mise}

<div class="install-meta">

<p class="install-platforms">macOS · Linux · Windows</p>

<div class="install-links">

[View package](https://mise-tools.jdx.dev/tools/task)

</div>

</div>

Install and activate Task globally with Mise:

```shell
mise use -g task
```

To use Task in the current project instead, run `mise use task`. This installs
Task and adds it to your project's `mise.toml`.

</div>

<div class="install-method" data-platforms="macos">

### [Macports](https://macports.org) {#macports}

<div class="install-meta">

<p class="install-platforms">macOS</p>

<div class="install-links">

[View package](https://ports.macports.org/port/go-task/details/)
[Source](https://github.com/macports/macports-ports/blob/master/devel/go-task/Portfile)

</div>

</div>

Task repository is tracked by Macports:

```shell
port install go-task
```

</div>

<div class="install-method" data-platforms="macos linux windows">

### [pip](https://pip.pypa.io) {#pip}

<div class="install-meta">

<p class="install-platforms">macOS · Linux · Windows</p>

<div class="install-links">

[View package](https://pypi.org/project/go-task-bin)
[Source](https://github.com/Bing-su/pip-binary-factory/tree/main/task)

</div>

</div>

Like npm, pip can be used as a cross-platform way to install Task:

```shell
pip install go-task-bin
```

</div>

<div class="install-method" data-platforms="windows">

### [Chocolatey](https://chocolatey.org) {#chocolatey}

<div class="install-meta">

<p class="install-platforms">Windows</p>

<div class="install-links">

[View package](https://community.chocolatey.org/packages/go-task)
[Source](https://github.com/Starz0r/ChocolateyPackagingScripts/blob/master/src/go-task_gh_build.py)

</div>

</div>

```shell
choco install go-task
```

</div>

<div class="install-method" data-platforms="windows">

### [Scoop](https://scoop.sh) {#scoop}

<div class="install-meta">

<p class="install-platforms">Windows</p>

<div class="install-links">

[Source](https://github.com/ScoopInstaller/Main/blob/master/bucket/task.json)

</div>

</div>

```shell
scoop install task
```

</div>

<div class="install-method" data-platforms="linux">

### Arch ([pacman](https://wiki.archlinux.org/title/Pacman)) {#arch}

<div class="install-meta">

<p class="install-platforms">Arch Linux</p>

<div class="install-links">

[View package](https://archlinux.org/packages/extra/x86_64/go-task/)
[Source](https://gitlab.archlinux.org/archlinux/packaging/packages/go-task)

</div>

</div>

```shell
pacman -S go-task
```

</div>

<div class="install-method" data-platforms="linux">

### Fedora ([dnf](https://docs.fedoraproject.org/en-US/quick-docs/dnf)) {#fedora-community}

<div class="install-meta">

<p class="install-platforms">Fedora</p>

<div class="install-links">

[View package](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/)
[Source](https://src.fedoraproject.org/rpms/golang-github-task)

</div>

</div>

```shell
dnf install go-task
```

</div>

<div class="install-method" data-platforms="freebsd">

### FreeBSD ([Ports](https://ports.freebsd.org/cgi/ports.cgi)) {#freebsd}

<div class="install-meta">

<p class="install-platforms">FreeBSD</p>

<div class="install-links">

[View package](https://cgit.freebsd.org/ports/tree/devel/task)
[Source](https://cgit.freebsd.org/ports/tree/devel/task/Makefile)

</div>

</div>

```shell
pkg install task
```

</div>

<div class="install-method" data-platforms="linux macos">

### [Nix](https://nixos.org) {#nix}

<div class="install-meta">

<p class="install-platforms">Nix · NixOS · Linux · macOS</p>

<div class="install-links">

[Source](https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/go/go-task/package.nix)

</div>

</div>

```shell
nix-env -iA nixpkgs.go-task
```

</div>

<div class="install-method" data-platforms="linux">

### [pacstall](https://github.com/pacstall/pacstall) {#pacstall}

<div class="install-meta">

<p class="install-platforms">Debian · Ubuntu</p>

<div class="install-links">

[View package](https://pacstall.dev/packages/go-task-deb)
[Source](https://github.com/pacstall/pacstall-programs/blob/master/packages/go-task-deb/go-task-deb.pacscript)

</div>

</div>

```shell
pacstall -I go-task-deb
```

</div>

<div class="install-method" data-platforms="macos linux">

### [pkgx](https://pkgx.sh) {#pkgx}

<div class="install-meta">

<p class="install-platforms">macOS · Linux</p>

<div class="install-links">

[View package](https://pkgx.dev/pkgs/taskfile.dev)
[Source](https://github.com/pkgxdev/pantry/blob/main/projects/taskfile.dev/package.yml)

</div>

</div>

```shell
pkgx task
```

or, if you have pkgx integration enabled:

```shell
task
```

</div>

</InstallationMethods>

## Binaries & CI {#get-the-binary}

### Binary

You can download the binary from the
[releases page on GitHub](https://github.com/go-task/task/releases) and add to
your `$PATH`.

DEB, RPM and APK packages are also available.

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

## Build from source

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

## Go Tool

If you're working in a Go project, a nice possibility is using `go tool`.
`go tool` makes it easy to run Task without needing to install the binary
manually. This works well on CI.

To do that, just run the following to add Task as a tool in your Go project.
Task will be added to your `go.mod`.

```bash
go get -tool github.com/go-task/task/v3/cmd/task@latest
```

Then, prefix `go tool` when calling Task like below. Go will compile Task on
demand before calling it.

```bash
go tool task {arguments...}
```

## Shell completions {#setup-completions}

Some installation methods will automatically install completions too, but if
this isn't working for you or your chosen method doesn't include them, you can
run `task --completion <shell>` to output a completion script for any supported
shell. There are a couple of ways these completions can be added to your shell
config:

### Option 1. Load the completions in your shell's startup config (Recommended)

This method loads the completion script from the currently installed version of
task every time you create a new shell. This ensures that your completions are
always up-to-date. If your executable isn’t named task, set the `TASK_EXE`
environment variable before running eval.

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

```nu [nushell]
# ~/.config/nushell/config.nu
#
# Nushell cannot source a script from stdin, so the script is saved where
# Nushell auto-loads it at startup. Autoload directories are read after
# config.nu, so the completions become available in the next shell.
mkdir ($nu.data-dir | path join "vendor/autoload")
task --completion nu | save --force ($nu.data-dir | path join "vendor/autoload/task-completions.nu")
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

```nu [nushell]
task --completion nu | save --force ($nu.data-dir | path join "vendor/autoload/task-completions.nu")
```

:::

### Zsh customization

The Zsh completion supports the standard `verbose` zstyle to control whether
task descriptions are shown. By default, descriptions are displayed. To show
only task names without descriptions, add this to your `~/.zshrc` (after the
completion is loaded):

```shell
zstyle ':completion:*:*:task:*' verbose false
```

By default, task aliases are also offered as completions. To complete only the
canonical task names, add the `show-aliases` zstyle:

```shell
zstyle ':completion:*:*:task:*' show-aliases false
```

### Nushell caveats

Nushell cannot source a script from stdin, so both options above write the
script to an autoload directory. Option 1 rewrites it at every startup, which
keeps it in sync with the installed version of Task — the refreshed completions
are picked up by the next shell. With option 2, re-run the command after
upgrading Task.

The completions are attached to an `extern "task"` declaration, which Nushell
requires to be static. Three consequences are worth knowing:

- The experimental flags (`--force-all`, `--download`, `--offline`, …) are
  always offered, even when the corresponding experiment is disabled. Their
  description is prefixed with the experiment name, and `task --experiments`
  lists the ones that are enabled.
- Passing a value to a boolean flag with `=` does not work: Nushell forwards
  `--color=false` as two arguments, so Task reads `false` as a task name. Use
  `NO_COLOR=1`, or bypass the declaration with `^task --color=false`.
- `TASK_EXE` selects the executable that is run, but not the command name the
  completions are attached to, which is always `task`. For a renamed executable,
  alias it instead:

```nu
use ($nu.data-dir | path join "vendor/autoload/task-completions.nu") *
alias go-task = task
```
