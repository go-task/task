---
slug: /releasing/
sidebar_position: 10
---

# Releasing

The release process of Task is done with the help of [GoReleaser](https://goreleaser.com/). You can test the release process locally by calling the `test-release` task of the Taskfile.

[GitHub Actions](https://github.com/go-task/task/actions) should release artifacts automatically when a new Git tag is pushed to master (raw executables and DEB and RPM packages).

Since v3.15.0, raw executables can also be reproduced and verified locally by checking out a specific tag and calling `goreleaser build`, using the Go version defined in the above GitHub Actions.

# Homebrew

Goreleaser will automatically push a new commit to the [Formula/go-task.rb](https://github.com/go-task/homebrew-tap/blob/master/Formula/go-task.rb) file in the [Homebrew tap](https://github.com/go-task/homebrew-tap) repository to release the new version.

# npm

To release to npm update the version in the [`package.json`](https://github.com/go-task/task/blob/master/package.json#L3) file and then run `task npm:publish` to push it.

# Snapcraft

The [snap package](https://github.com/go-task/snap) requires to manual steps to release a new version:

- Updating the current version on [snapcraft.yaml](https://github.com/go-task/snap/blob/master/snap/snapcraft.yaml#L2).
- Moving both `amd64`, `armhf` and `arm64` new artifacts to the stable channel on the [Snapcraft dashboard](https://snapcraft.io/task/releases).

# Scoop

Scoop is a command-line package manager for the Windows operating system. Scoop package manifests are maintained by the community. Scoop owners usually take care of updating versions there by editing [this file](https://github.com/ScoopInstaller/Main/blob/master/bucket/task.json). If you think its Task version is outdated, open an issue to let us know.

# Nix

Nix is a community owned installation method. Nix package maintainers usually take care of updating versions there by editing [this file](https://github.com/NixOS/nixpkgs/blob/nixos-unstable/pkgs/development/tools/go-task/default.nix). If you think its Task version is outdated, open an issue to let us know.

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
