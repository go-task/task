[![Join the chat at https://gitter.im/go-task/task](https://badges.gitter.im/go-task/task.svg)](https://gitter.im/go-task/task?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Build Status](https://travis-ci.org/go-task/task.svg?branch=master)](https://travis-ci.org/go-task/task)

# Task - Simple task runner / "Make" alternative

Task is a simple tool that allows you to easily run development and build
tasks. Task is written in Go, but can be used to develop any language.
It aims to be simpler and easier to use then [GNU Make][make].

## Installation

If you have a [Go][golang] environment setup, you can simply run:

```bash
go get -u -v github.com/go-task/task/cmd/task
```

Or you can download from the [releases][releases] page and add to your `PATH`.

## Usage

Create a file called `Taskfile.yml` in the root of the project.
(`Taskfile.toml` and `Taskfile.json` are also supported, but YAML is used in
the documentation). The `cmds` attribute should contains the commands of a
task:

```yml
build:
  cmds:
    - go build -v -i main.go

assets:
  cmds:
    - gulp
```

Running the tasks is as simple as running:

```bash
task assets build
```

If Bash is available (Linux and Windows while on Git Bash), the commands will
run in Bash, otherwise will fallback to `cmd` (on Windows).

If you ommit a task name, "default" will be assumed.

### Environment

You can specify environment variables that are added when running a command:

```yml
build:
  cmds:
  - echo $hallo
  env:
    hallo: welt
```

### OS specific task support

If you add a `Taskfile_{{GOOS}}` you can override or amend your taskfile based
on the operating system.

Example:

Taskfile.yml:

```yml
build:
  cmds:
  - echo  "default"
```

Taskfile_linux.yml:

```yml
build:
  cmds:
  - echo  "linux"
```

Will print out `linux` and not default

### Running task in another dir

By default, tasks will be executed in the directory where the Taskfile is
located. But you can easily make the task run in another folder informing
`dir`:

```yml
js:
  dir: www/public/js
  cmds:
    - gulp
```

### Task dependencies

You may have tasks that depends on others. Just pointing them on `deps` will
make them run automatically before running the parent task:

```yml
build:
  deps: [assets]
  cmds:
    - go build -v -i main.go

assets:
  cmds:
    - gulp
```

In the above example, `assets` will always run right before `build` if you run
`task build`.

A task can have only dependencies and no commands to group tasks together:

```yml
assets:
  deps: [js, css]

js:
  cmds:
    - npm run buildjs

css:
  cmds:
    - npm run buildcss
```

Each task can only be run once. If it is included from another dependend task causing
a cyclomatic dependency, execution will be stopped.

```yml
task1:
  deps: [task2]

task2:
  deps: [task1]
```

Will stop at the moment the dependencies of `task2` are executed.

### Calling a task from another task

When a task has many dependencies, they are executed concurrently. This will
often result in a faster build pipeline. But in some situations you may need
to call other tasks serially. For this just prefix a command with `^`:

```yml
a-task:
  cmds:
    - ^another-task
    - ^even-another-task
    - echo "Both done"

another-task:
  cmds:
    - ...

even-another-task:
  cmds:
    - ...
```

### Prevent task from running when not necessary

If a task generates something, you can inform Task the source and generated
files, so Task will prevent to run them if not necessary.

```yml
build:
  deps: [js, css]
  cmds:
    - go build -v -i main.go

js:
  cmds:
    - npm run buildjs
  sources:
    - js/src/**/*.js
  generates:
    - public/bundle.js

css:
  cmds:
    - npm run buildcss
  sources:
    - css/src/*.css
  generates:
    - public/bundle.css
```

`sources` and `generates` should be file patterns. When both are given, Task
will compare the modification date/time of the files to determine if it's
necessary to run the task. If not, it will just print
`Task "js" is up to date`.

You can use `--force` or `-f` if you want to force a task to run even when
up-to-date.

### Variables

```yml
build:
  deps: [setvar]
  cmds:
  - echo "{{.PREFIX}} {{.THEVAR}}"
  vars:
    PREFIX: "Path:"

setvar:
  cmds:
  - echo "{{.PATH}}"
  set: THEVAR
```

The above sample saves the path into a new variable which is then again echoed.

You can use environment variables, task level variables and a file called
`Taskvars.yml` (or `Taskvars.toml` or `Taskvars.json`) as source of variables.

They are evaluated in the following order:

Task local variables are overwritten by variables found in `Taskvars` file.
Variables found in `Taskvars` file are overwritten with variables from the
environment. The output of the last command is stored in the environment. So
you can do something like this:

```yml
build:
  deps: [setvar]
  cmds:
  - echo "{{.PREFIX}} '{{.THEVAR}}'"
  vars:
    PREFIX: "Result: "

setvar:
  cmds:
  - echo -n "a"
  - echo -n "{{.THEVAR}}b"
  - echo -n "{{.THEVAR}}c"
  set: THEVAR
```

The result of a run of build would be:

```
a
ab
abc
Result:  'abc'
```

#### Dynamic variables

If you prefix a variable with `$`, then the variable is considered a dynamic
variable. The value after the $-symbol will be treated as a command and the
output assigned.

```yml
build:
  cmds:
    - go build -ldflags="-X main.Version={{.LAST_GIT_COMMIT}}" main.go
  vars:
    LAST_GIT_COMMIT: $git log -n 1 --format=%h
```

### Go's template engine

Task parse commands as [Go's template engine][gotemplate] before executing
them. Variables are acessible through dot syntax (`.VARNAME`).

All functions by the Go's [sprig lib](http://masterminds.github.io/sprig/)
are available. The following example gets the current date in a given format:

```go
print-date:
  cmds:
    - echo {{now | date "2006-01-02"}}
```

Task also adds the following functions:

- `OS`: return operating system. Possible values are "windows", "linux",
  "darwin" (macOS) and "freebsd".
- `ARCH`: return the architecture Task was compiled to: "386", "amd64", "arm"
  or "s390x".
- `IsSH`: on unix systems this should always return `true`. On Windows, will
  return `true` if `sh` command was found (Git Bash). In this case commands
  will run on `sh`. Otherwise, this function will return `false` meaning
  commands will run on `cmd`.
- `ToSlash`: Does nothing on `sh`, but on `cmd` converts a string from `\`
  path format to `/`.
- `FromSlash`: Oposite of `ToSlash`. Does nothing on `sh`, but on `cmd`
  converts a string from `\` path format to `/`.

Example:

```yml
printos:
  cmds:
    - echo '{{OS}} {{ARCH}}'
    - "echo '{{if eq OS \"windows\"}}windows-command{{else}}unix-command{{end}}'"
    - echo 'Is SH? {{IsSH}}'
    # This will be ./bin/executable on sh but .\bin\executable on cmd
    - "{{FromSlash \"./bin/executable\"}}"
```

### Help

Running `task help` lists all tasks with a description. The following taskfile:

```yml
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
    - npm run buildjs

css:
  cmds:
    - npm run buildcss
```

would print the following output:

```bash
build   Build the go binary.
test    Run all the go tests.
```

## Watch tasks (experimental)

If you give a `--watch` or `-w` argument, task will watch for files changes
and run the task again. This requires the `sources` attribute to be given,
so task know which files to watch.

## Alternatives

Similar software:

- [tj/robo][robo]
- [dogtools/dog][dog]

[make]: https://www.gnu.org/software/make/
[releases]: https://github.com/go-task/task/releases
[golang]: https://golang.org/
[gotemplate]: https://golang.org/pkg/text/template/
[robo]: https://github.com/tj/robo
[dog]: https://github.com/dogtools/dog
