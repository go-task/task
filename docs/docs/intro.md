---
slug: /
sidebar_position: 1
title: Home
---

# Task

<div align="center">
  <img id="logo" src="img/logo.svg" height="250px" width="250px" />
</div>

Task is a task runner / build tool that aims to be simpler and easier to use
than, for example, [GNU Make][make].

Since it's written in [Go][go], Task is just a single binary and has no other
dependencies, which means you don't need to mess with any complicated install
setups just to use a build tool.

Once [installed](installation.md), you just need to describe your build tasks
using a simple [YAML][yaml] schema in a file called `Taskfile.yml`:

```yaml title="Taskfile.yml"
version: '3'

tasks:
  hello:
    cmds:
      - echo 'Hello World from Task!'
    silent: true
```

And call it by running `task hello` from your terminal.

The above example is just the start, you can take a look at the [usage](/usage)
guide to check the full schema documentation and Task features.

## Features

- [Easy installation](installation.md): just download a single binary, add to
  `$PATH` and you're done! Or you can also install using [Homebrew][homebrew],
  [Snapcraft][snapcraft], or [Scoop][scoop] if you want.
- Available on CIs: by adding
  [this simple command](installation.md#install-script) to install on your CI
  script and you're ready to use Task as part of your CI pipeline;
- Truly cross-platform: while most build tools only work well on Linux or macOS,
  Task also supports Windows thanks to [this shell interpreter for Go][sh].
- Great for code generation: you can easily
  [prevent a task from running](/usage#prevent-unnecessary-work) if a given set
  of files haven't changed since last run (based either on its timestamp or
  content).

<!-- prettier-ignore-start -->
[make]: https://www.gnu.org/software/make/
[go]: https://go.dev/
[yaml]: http://yaml.org/
[homebrew]: https://brew.sh/
[snapcraft]: https://snapcraft.io/
[scoop]: https://scoop.sh/
[sh]: https://github.com/mvdan/sh
<!-- prettier-ignore-end -->
