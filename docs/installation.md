# Installation

## Binary

Or you can download the binary from the [releases][releases] page and add to
your $PATH. DEB and RPM packages are also available.
The `task_checksums.txt` file contains the sha256 checksum for each file.

## Homebrew

If you're on macOS or Linux and have [Homebrew][homebrew] installed, getting
Task is as simple as running:

```bash
brew install go-task/tap/go-task
```

> This installation method is only currently supported on amd64 architectures.

## Snap

Task is available for [Snapcraft][snapcraft], but keep in mind that your
Linux distribution should allow classic confinement for Snaps to Task work
right:

```bash
sudo snap install task
```

## Scoop

If you're on Windows and have [Scoop][scoop] installed, use `extras` bucket
to install Task like:

```cmd
scoop bucket add extras
scoop install task
```

This installation method is community owned. After a new release of Task, it
may take some time until it's available on Scoop.

## Arch Linux

If you're on Arch Linux you can install Task from
[AUR](https://aur.archlinux.org/packages/taskfile-git) using your favorite
package manager such as `yay`, `pacaur` or `yaourt`:

```cmd
yay -S taskfile-git
```

This installation method is community owned, but since it's `-git` version of
the package, it's always latest available version based on the Git repository.

## Go

Task now uses [Go Modules](https://github.com/golang/go/wiki/Modules), which
means you may have trouble compiling it on older Go versions or using
`$GOPATH`.

For CI environments we recommend using the [Install Script](#install-script)
instead, which is faster and more stable, since it'll just download the latest
released binary, instead of compiling the edge (master branch) version.

Installing in another directory:

```bash
git clone https://github.com/go-task/task
cd task

# compiling binary to $GOPATH/bin
go install -v ./cmd/task

# compiling it to another location
# use -o ./task.exe on Windows
go build -v -o ./task ./cmd/task
```

Both methods requires having the [Go][go] environment properly setup locally.

## Install script

We also have a [install script][installscript], which is very useful on
scenarios like CIs. Many thanks to [godownloader][godownloader] for allowing
easily generating this script.

```bash
curl -sL https://taskfile.dev/install.sh | sh
```

> This method will download the binary on the local `./bin` directory by default.

## GitHub Actions

If you want to install Task in GitHub Actions you can try using
[this action](https://github.com/arduino/actions/tree/master/setup-taskfile)
by the Arduino team:

```yaml
- name: Install Task
  uses: Arduino/actions/setup-taskfile@master
```

This installation method is community owned.

[go]: https://golang.org/
[snapcraft]: https://snapcraft.io/
[homebrew]: https://brew.sh/
[installscript]: https://github.com/go-task/task/blob/master/install-task.sh
[releases]: https://github.com/go-task/task/releases
[godownloader]: https://github.com/goreleaser/godownloader
[scoop]: https://scoop.sh/
