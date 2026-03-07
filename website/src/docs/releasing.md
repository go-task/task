---
title: Releasing
description:
  Task release process including GoReleaser, Homebrew, npm, Snapcraft, winget,
  and other package managers
outline: deep
---

# Releasing

The release process of Task is done with the help of [GoReleaser][goreleaser].
You can test the release process locally by calling the `test-release` task of
the Taskfile.

[GitHub Actions](https://github.com/go-task/task/actions) should release
artifacts automatically when a new Git tag is pushed to `main` branch (raw
executables and DEB and RPM packages).

Raw executables can also be reproduced and verified locally by
checking out a specific tag and calling `goreleaser build`, using the Go version
defined in the above GitHub Actions.

## Package managers

GoReleaser will automatically publish the release to most package managers:

* Cloudsmith (DEB and RPM repositories)
* Homebrew
* npm
* winget

A single package manager still require manual steps:

* Snapcraft:
  * Update the `version:` field on [snapcraft.yaml][snapcraftyaml]
  <!-- * Trigger a new build on [Snapcraft -> Builds][snapcraftbuilds] -->
  * Once finished, move the new build to "stable" on [Snapcraft -> Releases][snapcraftreleases]

These package managers are updated automatically by the community:

* [Scoop](https://github.com/ScoopInstaller/Main/blob/master/bucket/task.json)
* [Nix](https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/go/go-task/package.nix)

[goreleaser]: https://goreleaser.com/
[snapcraftyaml]: https://github.com/go-task/snap/blob/main/snap/snapcraft.yaml#L2
[snapcraftbuilds]: https://snapcraft.io/task/builds
[snapcraftreleases]: https://snapcraft.io/task/releases
