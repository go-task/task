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

Task now uses [Go Modules](https://github.com/golang/go/wiki/Modules), which
means you may have trouble compiling it on older Go versions.

For CI environments we recommend using the [Install Script](#install-script)
instead, which is faster and more stable, since it'll just download the latest
released binary, instead of compiling the edge (master branch) version.

Installing in your `$GOPATH`:

```bash
go get -u -v github.com/go-task/task/cmd/task
```

Installing in another directory:

```bash
git clone https://github.com/go-task/task
cd task

# compiling binary to $GOPATH/bin
go install -v

# compiling it to another location
# use -o ./task.exe on Windows
go build -v -o ./task ./cmd/task
```

Both methods requires having the [Go][go] environment properly setup locally.

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
