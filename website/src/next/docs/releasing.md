---
title: Releasing
description:
  Task release process including GoReleaser, Homebrew, npm, Snapcraft, winget,
  and other package managers
section: Contributing
docType: contributing
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

Raw executables can also be reproduced and verified locally by checking out a
specific tag and calling `goreleaser build`, using the Go version defined in the
above GitHub Actions.

## Website

`task release:<version>` promotes the documentation before tagging: the docs in
`website/src/next/docs`, their sidebar and the `next-*` JSON schemas are copied
over their published counterparts, so the released tag carries the docs of the
version it ships. The release workflow then runs `task website:deploy:prod`.

A new blog post is a Markdown file in `website/src/next/blog` with no file at
the same relative path in `website/src/latest/blog`, excluding `index.md`.
Renaming a post therefore makes it new. On the `next` site, new posts display
the build date in UTC (`YYYY-MM-DD`) and are sorted using that date, without
changing the source files. The `latest` site uses only the stored dates.

During release preparation, `cmd/release` replaces or adds the `date` in each
new post's frontmatter with the same local date used for the changelog. It
validates all new posts' frontmatter before writing; an error identifies the
file to fix. It then copies the blog to `latest`, so both copies carry the same
date in the release commit. This is the preparation date, even if publication on
GitHub happens later. Already published posts keep their dates, including when
their content changes. The `--version` and `--notes` flags do not write files.

Because taskfile.dev is built from the latest copy, it can be redeployed at any
time between releases - to publish a blog post or a documentation fix - without
exposing the docs of unreleased features:

```shell
git checkout main && git pull
task website:deploy:prod
```

## Package managers

GoReleaser will automatically publish the release to most package managers:

- Cloudsmith (DEB and RPM repositories)
- Homebrew
- npm
- Snapcraft
- winget

Once the release is published, GoReleaser announces it on the
[Discord server](https://discord.gg/6TY36E39UK). Nightly releases are not
announced.

These package managers are updated automatically by the community:

- [Scoop](https://github.com/ScoopInstaller/Main/blob/master/bucket/task.json)
- [Nix](https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/go/go-task/package.nix)

[goreleaser]: https://goreleaser.com/
