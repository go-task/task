[![Join Slack room](https://img.shields.io/badge/%23task-chat%20room-blue.svg)](https://gophers.slack.com/messages/task)
[![Build Status](https://travis-ci.org/go-task/task.svg?branch=master)](https://travis-ci.org/go-task/task)

# Task - A task runner / simpler Make alternative written in Go

> We recently released version 2.0.0 of Task. The Taskfile changed a bit.
Please, check the [Taskfile versions](TASKFILE_VERSIONS.md) document to see
what changed and how to upgrade.

Task is a simple tool that allows you to easily run development and build
tasks. Task is written in Golang, but can be used to develop any language.
It aims to be simpler and easier to use then [GNU Make][make].

- [Installation](#installation)
  - [Go](#go)
  - [Homebrew](#homebrew)
  - [Snap](#snap)
  - [Binary](#binary)
- [Usage](#usage)
  - [Environment](#environment)
  - [OS specific task](#os-specific-task)
  - [Task directory](#task-directory)
  - [Task dependencies](#task-dependencies)
  - [Calling another task](#calling-another-task)
  - [Prevent unnecessary work](#prevent-unnecessary-work)
  - [Variables](#variables)
    - [Dynamic variables](#dynamic-variables)
  - [Go's template engine](#gos-template-engine)
  - [Help](#help)
  - [Silent mode](#silent-mode)
  - [Watch tasks](#watch-tasks-experimental)
- [Examples](#examples)
- [Alternative task runners](#alternative-task-runners)

## Installation

### Go

If you have a [Golang][golang] environment setup, you can simply run:

```bash
go get -u -v github.com/go-task/task/cmd/task
```

### Homebrew

If you're on macOS and have [Homebrew][homebrew] installed, getting Task is
as simple as running:

```bash
brew update
brew tap go-task/tap
brew install go-task
```

### Snap

Task is available for [Snapcraft][snapcraft], but keep in mind that your
Linux distribution should allow classic confinement for Snaps to Task work
right:

```bash
sudo snap install task
```

### Binary

Or you can download the binary from the [releases][releases] page and add to
your `PATH`. DEB and RPM packages are also available.
The `task_checksums.txt` file contains the sha256 checksum for each file.

## Usage

Create a file called `Taskfile.yml` in the root of the project.
The `cmds` attribute should contains the commands of a task.
The example below allows compile a Go app and uses [Minify][minify] to concat
and minify multiple CSS files into a single one.

```yml
version: '2'

tasks:
  build:
    cmds:
      - go build -v -i main.go

  assets:
    cmds:
      - minify -o public/style.css src/css
```

Running the tasks is as simple as running:

```bash
task assets build
```

Task uses [github.com/mvdan/sh](https://github.com/mvdan/sh), a native Go sh
interpreter. So you can write sh/bash commands and it will work even on
Windows, where `sh` or `bash` is usually not available. Just remember any
executable called must be available by the OS or in PATH.

If you ommit a task name, "default" will be assumed.

### Environment

You can specify environment variables that are added when running a command:

```yml
version: '2'

tasks:
  build:
    cmds:
      - echo $hallo
    env:
      hallo: welt
```

### OS specific task

If you add a `Taskfile_{{GOOS}}.yml` you can override or amend your taskfile
based on the operating system.

Example:

Taskfile.yml:

```yml
version: '2'

tasks:
  build:
    cmds:
      - echo "default"
```

Taskfile_linux.yml:

```yml
tasks:
  build:
    cmds:
      - echo "linux"
```

Will print out `linux` and not default.

It's also possible to have OS specific `Taskvars.yml` file, like
`Taskvars_windows.yml`, `Taskfile_linux.yml` or `Taskvars_darwin.yml`. See the
[variables section](#variables) below.

### Task directory

By default, tasks will be executed in the directory where the Taskfile is
located. But you can easily make the task run in another folder informing
`dir`:

```yml
version: '2'

tasks:
  serve:
    dir: public/www
    cmds:
      # run http server
      - caddy
```

### Task dependencies

You may have tasks that depends on others. Just pointing them on `deps` will
make them run automatically before running the parent task:

```yml
version: '2'

tasks:
  build:
    deps: [assets]
    cmds:
      - go build -v -i main.go

  assets:
    cmds:
      - minify -o public/style.css src/css
```

In the above example, `assets` will always run right before `build` if you run
`task build`.

A task can have only dependencies and no commands to group tasks together:

```yml
version: '2'

tasks:
  assets:
    deps: [js, css]

  js:
    cmds:
      - minify -o public/script.js src/js

  css:
    cmds:
      - minify -o public/style.css src/css
```

If there are more than one dependency, they always run in parallel for better
performance.

If you want to pass information to dependencies, you can do that the same
manner as you would to [call another task](#calling-another-task):

```yml
version: '2'

tasks:
  default:
    deps:
      - task: echo_sth
        vars: {TEXT: "before 1"}
      - task: echo_sth
        vars: {TEXT: "before 2"}
    cmds:
      - echo "after"

  echo_sth:
    cmds:
      - echo {{.TEXT}}
```

### Calling another task

When a task has many dependencies, they are executed concurrently. This will
often result in a faster build pipeline. But in some situations you may need
to call other tasks serially. In this case, just use the following syntax:

```yml
version: '2'

tasks:
  main-task:
    cmds:
      - task: task-to-be-called
      - task: another-task
      - echo "Both done"

  task-to-be-called:
    cmds:
      - echo "Task to be called"

  another-task:
    cmds:
      - echo "Another task"
```

Overriding variables in the called task is as simple as informing `vars`
attribute:

```yml
version: '2'

tasks:
  main-task:
    cmds:
      - task: write-file
        vars: {FILE: "hello.txt", CONTENT: "Hello!"}
      - task: write-file
        vars: {FILE: "world.txt", CONTENT: "World!"}

  write-file:
    cmds:
      - echo "{{.CONTENT}}" > {{.FILE}}
```

The above syntax is also supported in `deps`.

### Prevent unnecessary work

If a task generates something, you can inform Task the source and generated
files, so Task will prevent to run them if not necessary.

```yml
version: '2'

tasks:
  build:
    deps: [js, css]
    cmds:
      - go build -v -i main.go

  js:
    cmds:
      - minify -o public/script.js src/js
    sources:
      - src/js/**/*.js
    generates:
      - public/script.js

  css:
    cmds:
      - minify -o public/style.css src/css
    sources:
      - src/css/**/*.css
    generates:
      - public/style.css
```

`sources` and `generates` can be files or file patterns. When both are given,
Task will compare the modification date/time of the files to determine if it's
necessary to run the task. If not, it will just print a message like
`Task "js" is up to date`.

If you prefer this check to be made by the content of the files, instead of
its timestamp, just set the `method` property to `checksum`.
You will probably want to ignore the `.task` folder in your `.gitignore` file
(It's there that Task stores the last checksum).
This feature is still experimental and can change until it's stable.

```yml
version: '2'

tasks:
  build:
    cmds:
      - go build .
    sources:
      - ./*.go
    generates:
      - app{{exeExt}}
    method: checksum
```

> TIP: method `none` skips any validation and always run the task.

Alternatively, you can inform a sequence of tests as `status`. If no error
is returned (exit status 0), the task is considered up-to-date:

```yml
version: '2'

tasks:
  generate-files:
    cmds:
      - mkdir directory
      - touch directory/file1.txt
      - touch directory/file2.txt
    # test existence of files
    status:
      - test -d directory
      - test -f directory/file1.txt
      - test -f directory/file2.txt
```

You can use `--force` or `-f` if you want to force a task to run even when
up-to-date.

Also, `task --status [tasks]...` will exit with non-zero exit code if any of
the tasks is not up-to-date.

### Variables

When doing interpolation of variables, Task will look for the below.
They are listed below in order of importance (e.g. most important first):

- Variables declared locally in the task
- Variables given while calling a task from another.
  (See [Calling another task](#calling-another-task) above)
- Variables declared in the `vars:` option in the `Taskfile`
- Variables available in the `Taskvars.yml` file
- Environment variables

Example of sending parameters with environment variables:

```bash
$ TASK_VARIABLE=a-value task do-something
```

Since some shells don't support above syntax to set environment variables
(Windows) tasks also accepts a similar style when not in the beginning of
the command. Variables given in this form are only visible to the task called
right before.

```bash
$ task write-file FILE=file.txt "CONTENT=Hello, World!" print "MESSAGE=All done!"
```

Example of locally declared vars:

```yml
version: '2'

tasks:
  print-var:
    cmds:
      echo "{{.VAR}}"
    vars:
      VAR: Hello!
```

Example of global vars in a `Taskfile.yml`:

```yml
version: '2'

vars:
  GREETING: Hello from Taskfile!

tasks:
  greet:
    cmds:
      - echo "{{.GREETING}}"
```

Example of `Taskvars.yml` file:

```yml
PROJECT_NAME: My Project
DEV_MODE: production
GIT_COMMIT: {sh: git log -n 1 --format=%h}
```

#### Variables expansion

Variables are expanded 2 times by default. You can change that by setting the
`expansions:` option. Change that will be necessary if you compose many
variables together:

```yml
version: '2'

expansions: 3

vars:
  FOO: foo
  BAR: bar
  BAZ: baz
  FOOBAR: "{{.FOO}}{{.BAR}}"
  FOOBARBAZ: "{{.FOOBAR}}{{.BAZ}}"

tasks:
  default:
    cmds:
      - echo "{{.FOOBARBAZ}}"
```

#### Dynamic variables

The below syntax (`sh:` prop in a variable) is considered a dynamic variable.
The value will be treated as a command and the output assigned. If there is one
or more trailing newlines, the last newline will be trimmed.

```yml
version: '2'

tasks:
  build:
    cmds:
      - go build -ldflags="-X main.Version={{.GIT_COMMIT}}" main.go
    vars:
      GIT_COMMIT:
        sh: git log -n 1 --format=%h
```

This works for all types of variables.

### Go's template engine

Task parse commands as [Go's template engine][gotemplate] before executing
them. Variables are accessible through dot syntax (`.VARNAME`).

All functions by the Go's [sprig lib](http://masterminds.github.io/sprig/)
are available. The following example gets the current date in a given format:

```yml
version: '2'

tasks:
  print-date:
    cmds:
      - echo {{now | date "2006-01-02"}}
```

Task also adds the following functions:

- `OS`: Returns operating system. Possible values are "windows", "linux",
  "darwin" (macOS) and "freebsd".
- `ARCH`: return the architecture Task was compiled to: "386", "amd64", "arm"
  or "s390x".
- `splitLines`: Splits Unix (\n) and Windows (\r\n) styled newlines.
- `catLines`: Replaces Unix (\n) and Windows (\r\n) styled newlines with a space.
- `toSlash`: Does nothing on Unix, but on Windows converts a string from `\`
  path format to `/`.
- `fromSlash`: Oposite of `toSlash`. Does nothing on Unix, but on Windows
  converts a string from `\` path format to `/`.
- `exeExt`: Returns the right executable extension for the current OS
  (`".exe"` for Windows, `""` for others).

Example:

```yml
version: '2'

tasks:
  print-os:
    cmds:
      - echo '{{OS}} {{ARCH}}'
      - echo '{{if eq OS "windows"}}windows-command{{else}}unix-command{{end}}'
      # This will be path/to/file on Unix but path\to\file on Windows
      - echo '{{fromSlash "path/to/file"}}'
  enumerated-file:
    vars:
      CONTENT: |
        foo
        bar
    cmds:
      - |
        cat << EOF > output.txt
        {{range $i, $line := .CONTENT | splitLines -}}
        {{printf "%3d" $i}}: {{$line}}
        {{end}}EOF
```

### Help

Running `task --list` (or `task -l`) lists all tasks with a description.
The following taskfile:

```yml
version: '2'

tasks:
  build:
    desc: Build the go binary.
    cmds:
      - go build -v -i main.go

  test:
    desc: Run all the go tests.
    cmds:
      - go test -race ./...

  js:
    cmds:
      - minify -o public/script.js src/js

  css:
    cmds:
      - minify -o public/style.css src/css
```

would print the following output:

```bash
* build:   Build the go binary.
* test:    Run all the go tests.
```

## Silent mode

Silent mode disables echoing of commands before Task runs it.
For the following Taskfile:

```yml
version: '2'

tasks:
  echo:
    cmds:
      - echo "Print something"
```

Normally this will be print:

```sh
echo "Print something"
Print something
```

With silent mode on, the below will be print instead:

```sh
Print something
```

There's three ways to enable silent mode:

* At command level:

```yml
version: '2'

tasks:
  echo:
    cmds:
      - cmd: echo "Print something"
        silent: true
```

* At task level:

```yml
version: '2'

tasks:
  echo:
    cmds:
      - echo "Print something"
    silent: true
```

* Or globally with `--silent` or `-s` flag

If you want to suppress stdout instead, just redirect a command to `/dev/null`:

```yml
version: '2'

tasks:
  echo:
    cmds:
      - echo "This will print nothing" > /dev/null
```

## Watch tasks (experimental)

If you give a `--watch` or `-w` argument, task will watch for files changes
and run the task again. This requires the `sources` attribute to be given,
so task know which files to watch.

## Examples

The [go-task/examples][examples] intends to be a collection of Taskfiles for
various use cases.
(It still lacks many examples, though. Contributions are welcome).

## Alternative task runners

- YAML based:
  - [goeuro/myke][myke]
  - [dreadl0ck/zeus][zeus]
  - [rliebz/tusk][tusk]
- Go based:
  - [markbates/grift][grift]
  - [magefile/mage][mage]
- Make based:
  - [tj/mmake][mmake]

[make]: https://www.gnu.org/software/make/
[releases]: https://github.com/go-task/task/releases
[golang]: https://golang.org/
[gotemplate]: https://golang.org/pkg/text/template/
[myke]: https://github.com/goeuro/myke
[zeus]: https://github.com/dreadl0ck/zeus
[tusk]: https://github.com/rliebz/tusk
[grift]: https://github.com/markbates/grift
[mage]: https://github.com/magefile/mage
[mmake]: https://github.com/tj/mmake
[sh]: https://github.com/mvdan/sh
[minify]: https://github.com/tdewolff/minify/tree/master/cmd/minify
[examples]: https://github.com/go-task/examples
[snapcraft]: https://snapcraft.io/
[homebrew]: https://brew.sh/
