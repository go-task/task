# Installation

## Binary

Or you can download the binary from the [releases][releases] page and add to
your $PATH. DEB and RPM packages are also available.
The `task_checksums.txt` file contains the sha256 checksum for each file.

## Homebrew

If you're on macOS and have [Homebrew][homebrew] installed, getting Task is
as simple as running:

```bash
brew install go-task/tap/go-task
```

## Snap

Task is available for [Snapcraft][snapcraft], but keep in mind that your
Linux distribution should allow classic confinement for Snaps to Task work
right:

```bash
sudo snap install task
```

## Go

If you have a [Go][go] environment setup, you can simply run:

```bash
go get -u -v github.com/go-task/task/cmd/task
```

## Install script

We also have a [install script][installscript], which is very useful on
scanarios like CIs. Many thanks to [godownloader][godownloader] for allowing
easily generating this script.

```bash
curl -s https://taskfile.org/install.sh | sh
```

> This method will download the binary on the local `./bin` directory by default.

[go]: https://golang.org/
[snapcraft]: https://snapcraft.io/
[homebrew]: https://brew.sh/
[installscript]: https://github.com/go-task/task/blob/master/install-task.sh
[releases]: https://github.com/go-task/task/releases
[godownloader]: https://github.com/goreleaser/godownloader
