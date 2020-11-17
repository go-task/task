# Installation

Task offers many installation methods. Check out the available methods below.

## Package Managers

<!-- tabs:start -->

#### **Homebrew**

If you're on macOS or Linux and have [Homebrew][homebrew] installed, getting
Task is as simple as running:

```bash
brew install go-task/tap/go-task
```

#### **Snap**

Task is available in [Snapcraft][snapcraft], but keep in mind that your
Linux distribution should allow classic confinement for Snaps to Task work
right:

```bash
sudo snap install task --classic
```

#### **Scoop**

If you're on Windows and have [Scoop][scoop] installed, use `extras` bucket
to install Task like:

```cmd
scoop bucket add extras
scoop install task
```

This installation method is community owned. After a new release of Task, it
may take some time until it's available on Scoop.

#### **AUR**

If you're on Arch Linux you can install Task from
[AUR](https://aur.archlinux.org/packages/taskfile-git) using your favorite
package manager such as `yay`, `pacaur` or `yaourt`:

```cmd
yay -S taskfile-git
```

This installation method is community owned, but since it's `-git` version of
the package, it's always latest available version based on the Git repository.

<!-- tabs:end -->

## Get The Binary

<!-- tabs:start -->

#### **Binary**

You can download the binary from the [releases page on GitHub][releases] and
add to your `$PATH`.

DEB and RPM packages are also available.

The `task_checksums.txt` file contains the SHA-256 checksum for each file.

#### **Install Script**

We also have a [install script][installscript], which is very useful on
scenarios like CIs. Many thanks to [GoDownloader][godownloader] for allowing
easily generating this script.

```bash
# For Default Installion to ./bin with debug logging
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d

# For Installation To /usr/local/bin for userwide access with debug logging
# May require sudo sh
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin

```

> This method will download the binary on the local `./bin` directory by default.

#### **GitHub Actions**

If you want to install Task in GitHub Actions you can try using
[this action](https://github.com/arduino/actions/tree/master/setup-taskfile)
by the Arduino team:

```yaml
- name: Install Task
  uses: Arduino/actions/setup-taskfile@master
```

This installation method is community owned.

<!-- tabs:end -->

## Build From Source

<!-- tabs:start -->

#### **Go Modules**

First, make sure you have [Go][go] properly installed and setup.

Task requires [Go Modules](https://github.com/golang/go/wiki/Modules) and
doesn't officially support installing via `go get` anymore.

Installing in another directory:

```bash
git clone https://github.com/go-task/task
cd task

# Compiling binary to $GOPATH/bin
go install -v ./cmd/task

# Compiling it to another location.
# Use -o ./task.exe on Windows.
go build -v -o ./task ./cmd/task
```

> For CI environments we recommend using the [Install Script](#get-the-binary)
> instead, which is faster and more stable, since it'll just download the latest
> released binary, instead of compiling the edge (master branch) version.

<!-- tabs:end -->

[go]: https://golang.org/
[snapcraft]: https://snapcraft.io/task
[homebrew]: https://brew.sh/
[installscript]: https://github.com/go-task/task/blob/master/install-task.sh
[releases]: https://github.com/go-task/task/releases
[godownloader]: https://github.com/goreleaser/godownloader
[scoop]: https://scoop.sh/
