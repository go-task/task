---
title: go tool task
description: How to use Task using go tool.
author: andreynering
date: 2026-04-14
outline: deep
editLink: false
---

# `go tool task`

<AuthorCard :author="$frontmatter.author" />

Do you know that you can use Task without really needing to install it?

If you work with Go, you probably depend on external binaries like linters,
code generators and... Task.

But asking your coworkers or contributors to install dependencies can be messy.
Everyone is on a different operating system, use a different package manager,
etc. In fact, [Task supports several package managers][install], but even having
to choose how you want to install it can lead to some fatigue.

Well, turns out you can just use `go tool`!

Step one: add Task as a tool to your Go project:

```bash
go get -tool github.com/go-task/task/v3/cmd/task@latest
```

The command above will add a line like this to your `go.mod`:

```
tool github.com/go-task/task/v3/cmd/task
```

Step two: prefix `go tool` when calling Task:

```bash
go tool task {arguments...}
```

That's all!

Go will compile the specified Task version on demand when calling `go tool task`.
Don't worry, Go caches the tool, so subsequent calls are faster.

This is useful when running Task on CI, as you don't need to stress about having
to install it. It also means it'll be pinned to a specific Task version (but
Dependabot or Renovate should be able to update it for you).

[install]: https://taskfile.dev/docs/installation
