---
title: Package API Reference
description: A reference for Task's Golang package API
---

# Package API Reference

::: warning

**_Task's package API is still experimental and subject to breaking changes._**

This means that unlike our CLI, we may make breaking changes to the package API
in minor (or even patch) releases. We try to avoid this when possible, but it
may be necessary in order to improve the overall design of the package API.

In the future we may stabilize the package API. However, this is not currently
planned. For now, if you need to use Task as a Go package, we recommend pinning
the version in your `go.mod` file. Where possible we will try to include a
changelog entry for breaking changes to the package API.

:::

Task is primarily a CLI tool that is agnostic of any programming language.
However, it is written in Go and therefore can also be used as a Go package too.
This can be useful if you are already using Go in your project and you need to
extend Task's functionality in some way. In this document, we describe the
public API surface of Task and how to use it. This may also be useful if you
want to contribute to Task or understand how it works in more detail.

## Key packages

The following packages make up the most important parts of Task's package API.
Below we have listed what they are for and some of the key types available:

### [`github.com/go-task/task/v3`]

The core task package provides most of the main functionality for Task including
fetching and executing tasks from a Taskfile. At this time, the vast majority of
the this package's functionality is exposed via the [`task.Executor`] which
allows the user to fetch and execute tasks from a Taskfile.

::: info

This is the package which is most likely to be the subject of breaking changes
as we refine the API.

:::

### [`github.com/go-task/task/v3/taskfile`]

The `taskfile` package provides utilities for _reading_ Taskfiles from various
sources. These sources can be local files, remote files, or even in-memory
strings (via stdin).

- [`taskfile.Node`] - A reference to the location of a Taskfile. A `Node` is an
  interface that has several implementations:
  - [`taskfile.FileNode`] - Local files
  - [`taskfile.HTTPNode`] - Remote files via HTTP/HTTPS
  - [`taskfile.GitNode`] - Remote files via Git
  - [`taskfile.StdinNode`] - In-memory strings (via stdin)
- [`taskfile.Reader`] - Accepts a `Node` and reads the Taskfile from it.
- [`taskfile.Snippet`] - Mostly used for rendering Taskfile errors. A snippet
  stores a small part of a taskfile around a given line number and column. The
  output can be syntax highlighted for CLIs and include line/column indicators.

### [`github.com/go-task/task/v3/taskfile/ast`]

AST stands for ["Abstract Syntax Tree"][ast]. An AST allows us to easily
represent the Taskfile syntax in Go. This package provides a way to parse
Taskfile YAML into an AST and store them in memory.

- [`ast.TaskfileGraph`] - Represents a set of Taskfiles and their dependencies
  between one another.
- [`ast.Taskfile`] - Represents a single Taskfile or a set of merged Taskfiles.
  The `Taskfile` type contains all of the subtypes for the Taskfile syntax, such
  as `tasks`, `includes`, `vars`, etc. These are not listed here for brevity.

### [`github.com/go-task/task/v3/errors`]

Contains all of the error types used in Task. All of these types implement the
[`errors.TaskError`] interface which wraps Go's standard [`error`] interface.
This allows you to call the `Code` method on the error to obtain the unique exit
code for any error.

## Reading Taskfiles

Start by importing the `github.com/go-task/task/v3/taskfile` package. This
provides all of the functions you need to read a Taskfile into memory:

```go
import (
    "github.com/go-task/task/v3/taskfile"
)
```

Reading Taskfiles is done by using a [`taskfile.Reader`] and an implementation
of [`taskfile.Node`]. In this example we will read a local file by using the
[`taskfile.FileNode`] type. You can create this by calling the
[`taskfile.NewFileNode`] function:

```go
node := taskfile.NewFileNode("Taskfile.yml", "./path/to/dir")
```

and then create a your reader by calling the [`taskfile.NewReader`] function and
passing any functional options you want to use. For example, you could pass a
debug function to the reader which will be called with debug messages:

```go
reader := taskfile.NewReader(
  taskfile.WithDebugFunc(func(s string) {
    slog.Debug(s)
  }),
)
```

Now that everything is set up, you can read the Taskfile (and any included
Taskfiles) by calling the `Read` method on the reader and pass the `Node` as an
argument:

```go
ctx := context.Background()
tfg, err := reader.Read(ctx, node)
// handle error
```

This returns an instance of [`ast.TaskfileGraph`] which is a "Directed Acyclic
Graph" (DAG) of all the parsed Taskfiles. We use this graph to store and resolve
the `includes` directives in Taskfiles. However most of the time, you will want
a merged Taskfile. To do this, simply call the `Merge` method on the Taskfile
graph:

```go
tf, err := tfg.Merge()
// handle error
```

This compiles the DAG into a single [`ast.Taskfile`] containing all the
namespaces and tasks from all the Taskfiles we read.

::: info

We plan to remove AST merging in the future as it is unnecessarily complex and
causes lots of issues with scoping.

:::

[`github.com/go-task/task/v3`]: https://pkg.go.dev/github.com/go-task/task/v3
[`github.com/go-task/task/v3/taskfile`]:
  https://pkg.go.dev/github.com/go-task/task/v3/taskfile
[`github.com/go-task/task/v3/taskfile/ast`]:
  https://pkg.go.dev/github.com/go-task/task/v3/taskfile/ast
[`github.com/go-task/task/v3/errors`]:
  https://pkg.go.dev/github.com/go-task/task/v3/errors
[`ast.TaskfileGraph`]:
  https://pkg.go.dev/github.com/go-task/task/v3/taskfile/ast#TaskfileGraph
[`ast.Taskfile`]:
  https://pkg.go.dev/github.com/go-task/task/v3/taskfile/ast#Taskfile
[`taskfile.Node`]: https://pkg.go.dev/github.com/go-task/task/v3/taskfile#Node
[`taskfile.FileNode`]:
  https://pkg.go.dev/github.com/go-task/task/v3/taskfile#FileNode
[`taskfile.HTTPNode`]:
  https://pkg.go.dev/github.com/go-task/task/v3/taskfile#HTTPNode
[`taskfile.GitNode`]:
  https://pkg.go.dev/github.com/go-task/task/v3/taskfile#GitNode
[`taskfile.StdinNode`]:
  https://pkg.go.dev/github.com/go-task/task/v3/taskfile#StdinNode
[`taskfile.NewFileNode`]:
  https://pkg.go.dev/github.com/go-task/task/v3/taskfile#NewFileNode
[`taskfile.Reader`]:
  https://pkg.go.dev/github.com/go-task/task/v3/taskfile#Reader
[`taskfile.NewReader`]:
  https://pkg.go.dev/github.com/go-task/task/v3/taskfile#NewReader
[`taskfile.Snippet`]:
  https://pkg.go.dev/github.com/go-task/task/v3/taskfile#Snippet
[`task.Executor`]: https://pkg.go.dev/github.com/go-task/task/v3#Executor
[`task.Formatter`]: https://pkg.go.dev/github.com/go-task/task/v3#Formatter
[`errors.TaskError`]:
  https://pkg.go.dev/github.com/go-task/task/v3/errors#TaskError
[`error`]: https://pkg.go.dev/builtin#error
[ast]: https://en.wikipedia.org/wiki/Abstract_syntax_tree
