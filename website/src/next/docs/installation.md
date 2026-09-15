---
title: Installation
description: Installation methods for Task
outline: deep
pageClass: installation-page
---

<script setup>
import InstallationMethod from '../../../.vitepress/components/InstallationMethod.vue';
import InstallationMethods from '../../../.vitepress/components/InstallationMethods.vue';
</script>

# Installation

Choose a package manager for your system, or install Task from a binary.

<InstallationMethods>

<nav class="installation-shortcuts" aria-label="Installation shortcuts">
  <a href="#get-the-binary">Binaries &amp; CI</a>
  <a href="#verify-installation">Verify installation</a>
  <a href="#setup-completions">Shell completions</a>
</nav>

## Official packages {#official-package-managers}

Maintained by the Task team and kept up-to-date.

<InstallationMethod platforms="macos" platform-label="macOS">

<template v-slot:heading>

### [Homebrew](https://brew.sh) {#homebrew}

</template>

<template v-slot:links>

[Tap source](https://github.com/go-task/homebrew-tap/blob/main/Casks/go-task.rb)
[Homebrew package](https://formulae.brew.sh/formula/go-task)
[Formula source](https://github.com/Homebrew/homebrew-core/blob/master/Formula/g/go-task.rb)

</template>

```shell
brew install go-task/tap/go-task
```

Or use the Homebrew core formula:

```shell
brew install go-task
```

</InstallationMethod>

<InstallationMethod platforms="linux" platform-label="Fedora · CentOS · Red Hat">

<template v-slot:heading>

### [dnf](https://docs.fedoraproject.org/en-US/quick-docs/dnf) {#dnf}

</template>

<template v-slot:links>

[View package](https://cloudsmith.io/~task/repos/task/packages/?sort=-format&q=format%3Arpm)

</template>

Set up the repository:

```shell
curl -1sLf 'https://dl.cloudsmith.io/public/task/task/setup.rpm.sh' | sudo -E bash
```

Install Task:

```shell
dnf install task
```

</InstallationMethod>

<InstallationMethod platforms="linux" platform-label="Ubuntu · Debian · Linux Mint">

<template v-slot:heading>

### [apt](https://doc.ubuntu-fr.org/apt) {#apt}

</template>

<template v-slot:links>

[View package](https://cloudsmith.io/~task/repos/task/packages/?sort=-format&q=format%3Adeb)

</template>

Set up the repository:

```shell
curl -1sLf 'https://dl.cloudsmith.io/public/task/task/setup.deb.sh' | sudo -E bash
```

Install Task:

```shell
apt install task
```

</InstallationMethod>

<InstallationMethod platforms="linux" platform-label="Alpine Linux">

<template v-slot:heading>

### [apk](https://wiki.alpinelinux.org/wiki/Alpine_Package_Keeper) {#apk}

</template>

<template v-slot:links>

[View package](https://cloudsmith.io/~task/repos/task/packages/?sort=-format&q=format%3Aalpine)

</template>

Set up the repository:

```shell
curl -1sLf 'https://dl.cloudsmith.io/public/task/task/setup.alpine.sh' | sudo -E bash
```

Install Task:

```shell
apk add task
```

</InstallationMethod>

<InstallationMethod platforms="linux" platform-label="Linux">

<template v-slot:heading>

### [Snap](https://snapcraft.io/task) {#snap}

</template>

<template v-slot:links>

[Source](https://github.com/go-task/snap/blob/main/snap/snapcraft.yaml)

</template>

```shell
sudo snap install task --classic
```

Requires a Linux distribution with classic confinement support.

</InstallationMethod>

<InstallationMethod platforms="macos linux windows" platform-label="macOS · Linux · Windows">

<template v-slot:heading>

### [npm](https://www.npmjs.com) {#npm}

</template>

<template v-slot:links>

[View package](https://www.npmjs.com/package/@go-task/cli)
[Source](https://github.com/go-task/task/blob/main/package.json)

</template>

```shell
npm install -g @go-task/cli
```

Task is also available as a project dependency.

</InstallationMethod>

<InstallationMethod platforms="windows" platform-label="Windows">

<template v-slot:heading>

### [WinGet](https://github.com/microsoft/winget-cli) {#winget}

</template>

<template v-slot:links>

[Source](https://github.com/microsoft/winget-pkgs/tree/master/manifests/t/Task/Task)

</template>

```shell
winget install Task.Task
```

Available through the
[WinGet community repository](https://github.com/microsoft/winget-pkgs).

</InstallationMethod>

<p class="install-hosting">Package repository hosting for deb/rpm/apk is graciously provided by <a href="https://cloudsmith.com">Cloudsmith</a>.</p>

## Community packages {#community-maintained-package-managers}

Maintained by the community, outside the Task team's control. These packages may
lag behind the latest release.

<InstallationMethod platforms="macos linux windows" platform-label="macOS · Linux · Windows">

<template v-slot:heading>

### [Mise](https://mise.jdx.dev/) {#mise}

</template>

<template v-slot:links>

[View package](https://mise-tools.jdx.dev/tools/task)

</template>

```shell
mise use -g task
```

For a project-local installation, use `mise use task` instead. This adds Task to
your project's `mise.toml`.

</InstallationMethod>

<InstallationMethod platforms="macos" platform-label="macOS">

<template v-slot:heading>

### [Macports](https://macports.org) {#macports}

</template>

<template v-slot:links>

[View package](https://ports.macports.org/port/go-task/details/)
[Source](https://github.com/macports/macports-ports/blob/master/devel/go-task/Portfile)

</template>

```shell
port install go-task
```

</InstallationMethod>

<InstallationMethod platforms="macos linux windows" platform-label="macOS · Linux · Windows">

<template v-slot:heading>

### [pip](https://pip.pypa.io) {#pip}

</template>

<template v-slot:links>

[View package](https://pypi.org/project/go-task-bin)
[Source](https://github.com/Bing-su/pip-binary-factory/tree/main/task)

</template>

```shell
pip install go-task-bin
```

</InstallationMethod>

<InstallationMethod platforms="windows" platform-label="Windows">

<template v-slot:heading>

### [Chocolatey](https://chocolatey.org) {#chocolatey}

</template>

<template v-slot:links>

[View package](https://community.chocolatey.org/packages/go-task)
[Source](https://github.com/Starz0r/ChocolateyPackagingScripts/blob/master/src/go-task_gh_build.py)

</template>

```shell
choco install go-task
```

</InstallationMethod>

<InstallationMethod platforms="windows" platform-label="Windows">

<template v-slot:heading>

### [Scoop](https://scoop.sh) {#scoop}

</template>

<template v-slot:links>

[Source](https://github.com/ScoopInstaller/Main/blob/master/bucket/task.json)

</template>

```shell
scoop install task
```

</InstallationMethod>

<InstallationMethod platforms="linux" platform-label="Arch Linux">

<template v-slot:heading>

### Arch ([pacman](https://wiki.archlinux.org/title/Pacman)) {#arch}

</template>

<template v-slot:links>

[View package](https://archlinux.org/packages/extra/x86_64/go-task/)
[Source](https://gitlab.archlinux.org/archlinux/packaging/packages/go-task)

</template>

```shell
pacman -S go-task
```

</InstallationMethod>

<InstallationMethod platforms="linux" platform-label="Fedora">

<template v-slot:heading>

### Fedora ([dnf](https://docs.fedoraproject.org/en-US/quick-docs/dnf)) {#fedora-community}

</template>

<template v-slot:links>

[View package](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/)
[Source](https://src.fedoraproject.org/rpms/golang-github-task)

</template>

```shell
dnf install go-task
```

</InstallationMethod>

<InstallationMethod platforms="freebsd" platform-label="FreeBSD">

<template v-slot:heading>

### FreeBSD ([Ports](https://ports.freebsd.org/cgi/ports.cgi)) {#freebsd}

</template>

<template v-slot:links>

[View package](https://cgit.freebsd.org/ports/tree/devel/task)
[Source](https://cgit.freebsd.org/ports/tree/devel/task/Makefile)

</template>

```shell
pkg install task
```

</InstallationMethod>

<InstallationMethod platforms="linux macos" platform-label="Nix · NixOS · Linux · macOS">

<template v-slot:heading>

### [Nix](https://nixos.org) {#nix}

</template>

<template v-slot:links>

[Source](https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/go/go-task/package.nix)

</template>

```shell
nix-env -iA nixpkgs.go-task
```

</InstallationMethod>

<InstallationMethod platforms="linux" platform-label="Debian · Ubuntu">

<template v-slot:heading>

### [pacstall](https://github.com/pacstall/pacstall) {#pacstall}

</template>

<template v-slot:links>

[View package](https://pacstall.dev/packages/go-task-deb)
[Source](https://github.com/pacstall/pacstall-programs/blob/master/packages/go-task-deb/go-task-deb.pacscript)

</template>

```shell
pacstall -I go-task-deb
```

</InstallationMethod>

<InstallationMethod platforms="macos linux" platform-label="macOS · Linux">

<template v-slot:heading>

### [pkgx](https://pkgx.sh) {#pkgx}

</template>

<template v-slot:links>

[View package](https://pkgx.dev/pkgs/taskfile.dev)
[Source](https://github.com/pkgxdev/pantry/blob/main/projects/taskfile.dev/package.yml)

</template>

```shell
pkgx task
```

or, if you have pkgx integration enabled:

```shell
task
```

</InstallationMethod>

</InstallationMethods>

## Binaries & CI {#get-the-binary}

Install Task without a package manager: download a binary, use the install
script, or add the setup action to your GitHub workflow.

### Download a binary {#binary}

1. Open the [GitHub releases](https://github.com/go-task/task/releases) and
   download the archive for your operating system and architecture.
2. Extract `task` (`task.exe` on Windows).
3. Move the executable to a directory on your `PATH`.

Each release also includes DEB, RPM and APK packages, plus `task_checksums.txt`
with SHA-256 checksums for the release files.

### Install with a script {#install-script}

Use the
[install script](https://github.com/go-task/task/blob/main/install-task.sh) for
a shell-based installation, including CI environments. It downloads a prebuilt
binary and verifies its checksum; no Go installation is needed.

By default, the script installs the latest release into `./bin`, relative to
your current directory. Choose a different directory or pin a release:

::: code-group

```shell [Latest release]
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d
```

```shell [Custom directory]
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
```

```shell [Pinned version]
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d v3.42.1
```

```shell [Directory + version]
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin v3.42.1
```

:::

When combining options, keep the
[release tag](https://github.com/go-task/task/releases) last.

::: tip Make Task available in your shell

Add the installation directory to your `PATH` to run `task` from anywhere. With
the default location, you can also run `./bin/task` directly.

On Linux, `~/.local/bin` and `~/bin` are common per-user locations;
`/usr/local/bin` is a system-wide location and may require elevated permissions.
Do not assume these directories are already on your `PATH`, especially on macOS
and Windows.

:::

### GitHub Actions

Add the [official setup action](https://github.com/go-task/setup-task) to your
job's `steps` before running Task:

```yaml
- name: Install Task
  uses: go-task/setup-task@v2

- name: Verify Task
  run: task --version
```

Use the action's `version` input to pin a Task release. See the
[action documentation](https://github.com/go-task/setup-task#usage) for examples
and configuration.

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

## Verify installation

After installing, open a terminal and check that Task is available:

```shell
task --version
```

If you installed Task with `go tool`, run `go tool task --version` instead.

You're ready to [create your first Taskfile](./getting-started.md). You can also
enable shell completions below.

## Shell completions {#setup-completions}

Some installation methods will automatically install completions too, but if
this isn't working for you or your chosen method doesn't include them, you can
run `task --completion <shell>` to output a completion script for any supported
shell.

Every shell shares a single source of truth: the script is a thin wrapper that
asks the `task` binary itself what to suggest, so Bash, Zsh, Fish, Nushell and
PowerShell all offer the same task names, aliases, flags, flag values and
`requires` vars.

There are a couple of ways these completions can be added to your shell config:

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

Nushell shares a single external completer between every command, so the script
chains to the one already configured — carapace and friends keep working. Load it
from an autoload directory as shown above rather than from `config.nu`, so that
your own completer is the one being chained to. If you would rather wire it
yourself, the script also exposes a `task-external-completer` command:

```nu
$env.config.completions.external.completer = {|spans|
    match ($spans | first) {
        task => (task-external-completer $spans)
        _ => (do $my_other_completer $spans)
    }
}
```

Two engine directives behave differently under Nushell by design: it never
appends a space after an external completion (so `NoSpace` is a no-op) and never
re-sorts the results (so `KeepOrder` is always honoured).

### Legacy completion scripts

Before the engine, every shell carried its own hand-written completion script,
each with its own idea of what to suggest. Those scripts are still shipped and
available through `task --legacy-completion <shell>`, as an escape hatch should
the engine misbehave in your setup. They are deprecated, will not receive further
fixes, and will be removed in a future release.
