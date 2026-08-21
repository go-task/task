---
title: Releasing
description:
  Task release process including GoReleaser, Homebrew, npm, Snapcraft, winget,
  and other package managers
outline: deep
---

# Releasing

The release process of Task is done with the help of [GoReleaser][goreleaser].
You can test the release process locally by calling the `goreleaser:test` task
of the Taskfile.

[GitHub Actions](https://github.com/go-task/task/actions) should release
artifacts automatically when a new Git tag is pushed to `main` branch (raw
executables and DEB and RPM packages).

The body of the GitHub release is the section of the `CHANGELOG.md` matching the
version being released, extracted by `go run ./cmd/release --notes`. A version
without changelog entries is released with an empty body.

Raw executables can also be reproduced and verified locally by
checking out a specific tag and calling `goreleaser build`, using the Go version
defined in the above GitHub Actions.

## Website

`task release:<version>` promotes the documentation before tagging: the docs in
`website/src/next/docs`, their sidebar and the `next-*` JSON schemas are copied over
their published counterparts, so the released tag carries the docs of the
version it ships. The release workflow then runs `task website:deploy:prod`.

Because taskfile.dev is built from the latest copy, it can be redeployed at any
time between releases - to publish a blog post or a documentation fix -
without exposing the docs of unreleased features:

```shell
git checkout main && git pull
task website:deploy:prod
```

## Package managers

GoReleaser will automatically publish the release to most package managers:

* Cloudsmith (DEB and RPM repositories)
* Homebrew
* npm
* winget

A single package manager still require manual steps:

* Snapcraft:
  * Update the `version:` field on [snapcraft.yaml][snapcraftyaml]
  * Trigger a new build on [Snapcraft -> Builds][snapcraftbuilds]
  * Once finished, move the new build to "stable" on [Snapcraft -> Releases][snapcraftreleases]

These package managers are updated automatically by the community:

* [Scoop](https://github.com/ScoopInstaller/Main/blob/master/bucket/task.json)
* [Nix](https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/go/go-task/package.nix)

[goreleaser]: https://goreleaser.com/
[snapcraftyaml]: https://github.com/go-task/snap/blob/main/snap/snapcraft.yaml#L2
[snapcraftbuilds]: https://snapcraft.io/task/builds
[snapcraftreleases]: https://snapcraft.io/task/releases
