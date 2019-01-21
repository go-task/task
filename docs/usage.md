# Usage

## Getting started

Create a file called `Taskfile.yml` in the root of your project.
The `cmds` attribute should contain the commands of a task.
The example below allows compiling a Go app and uses [Minify][minify] to concat
and minify multiple CSS files into a single one.

```yaml
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
Windows, where `sh` or `bash` are usually not available. Just remember any
executable called must be available by the OS or in PATH.

If you omit a task name, "default" will be assumed.

## Environment

You can use `env` to set custom environment variables for a specific task:

```yaml
version: '2'

tasks:
  greet:
    cmds:
      - echo $GREETING
    env:
      GREETING: Hey, there!
```

Additionally, you can set globally environment variables, that'll be available
to all tasks:

```yaml
version: '2'

env:
  GREETING: Hey, there!

tasks:
  greet:
    cmds:
      - echo $GREETING
```

> NOTE: `env` supports expansion and and retrieving output from a shell command
> just like variables, as you can see on the [Variables](#variables) section.

## Operating System specific tasks

If you add a `Taskfile_{{GOOS}}.yml` you can override or amend your Taskfile
based on the operating system.

Example:

Taskfile.yml:

```yaml
version: '2'

tasks:
  build:
    cmds:
      - echo "default"
```

Taskfile_linux.yml:

```yaml
version: '2'

tasks:
  build:
    cmds:
      - echo "linux"
```

Will print out `linux` and not `default`.

Keep in mind that the version of the files should match. Also, when redefining
a task the whole task is replaced, properties of the task are not merged.

It's also possible to have an OS specific `Taskvars.yml` file, like
`Taskvars_windows.yml`, `Taskvars_linux.yml`, or `Taskvars_darwin.yml`. See the
[variables section](#variables) below.

## Including other Taskfiles

> This feature is still experimental and may have bugs.

If you want to share tasks between different projects (Taskfiles), you can use
the importing mechanism to include other Taskfiles using the `includes` keyword:

```yaml
version: '2'

includes:
  docs: ./documentation # will look for ./documentation/Taskfile.yml
  docker: ./DockerTasks.yml
```

The tasks described in the given Taskfiles will be available with the informed
namespace. So, you'd call `task docs:serve` to run the `serve` task from
`documentation/Taskfile.yml` or `task docker:build` to run the `build` task
from the `DockerTasks.yml` file.

> The included Taskfiles must be using the same schema version the main
> Taskfile uses.

> Also, for now included Taskfiles can't include other Taskfiles.
> This was a deliberate decision to keep use and implementation simple.
> If you disagree, open an GitHub issue and explain your use case. =)

## Task directory

By default, tasks will be executed in the directory where the Taskfile is
located. But you can easily make the task run in another folder informing
`dir`:

```yaml
version: '2'

tasks:
  serve:
    dir: public/www
    cmds:
      # run http server
      - caddy
```

## Task dependencies

You may have tasks that depend on others. Just pointing them on `deps` will
make them run automatically before running the parent task:

```yaml
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

```yaml
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

If there is more than one dependency, they always run in parallel for better
performance.

If you want to pass information to dependencies, you can do that the same
manner as you would to [call another task](#calling-another-task):

```yaml
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

## Calling another task

When a task has many dependencies, they are executed concurrently. This will
often result in a faster build pipeline. But in some situations you may need
to call other tasks serially. In this case, just use the following syntax:

```yaml
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

```yaml
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

## Prevent unnecessary work

If a task generates something, you can inform Task the source and generated
files, so Task will prevent to run them if not necessary.

```yaml
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

```yaml
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

```yaml
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

Also, `task --status [tasks]...` will exit with a non-zero exit code if any of
the tasks are not up-to-date.

## Variables

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

```yaml
version: '2'

tasks:
  print-var:
    cmds:
      echo "{{.VAR}}"
    vars:
      VAR: Hello!
```

Example of global vars in a `Taskfile.yml`:

```yaml
version: '2'

vars:
  GREETING: Hello from Taskfile!

tasks:
  greet:
    cmds:
      - echo "{{.GREETING}}"
```

Example of `Taskvars.yml` file:

```yaml
PROJECT_NAME: My Project
DEV_MODE: production
GIT_COMMIT: {sh: git log -n 1 --format=%h}
```

### Variables expansion

Variables are expanded 2 times by default. You can change that by setting the
`expansions:` option. Change that will be necessary if you compose many
variables together:

```yaml
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

### Dynamic variables

The below syntax (`sh:` prop in a variable) is considered a dynamic variable.
The value will be treated as a command and the output assigned. If there is one
or more trailing newlines, the last newline will be trimmed.

```yaml
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

## Go's template engine

Task parse commands as [Go's template engine][gotemplate] before executing
them. Variables are accessible through dot syntax (`.VARNAME`).

All functions by the Go's [sprig lib](http://masterminds.github.io/sprig/)
are available. The following example gets the current date in a given format:

```yaml
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
- `fromSlash`: Opposite of `toSlash`. Does nothing on Unix, but on Windows
  converts a string from `\` path format to `/`.
- `exeExt`: Returns the right executable extension for the current OS
  (`".exe"` for Windows, `""` for others).

Example:

```yaml
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

## Help

Running `task --list` (or `task -l`) lists all tasks with a description.
The following Taskfile:

```yaml
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

```yaml
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

```yaml
version: '2'

tasks:
  echo:
    cmds:
      - cmd: echo "Print something"
        silent: true
```

* At task level:

```yaml
version: '2'

tasks:
  echo:
    cmds:
      - echo "Print something"
    silent: true
```

* Or globally with `--silent` or `-s` flag

If you want to suppress STDOUT instead, just redirect a command to `/dev/null`:

```yaml
version: '2'

tasks:
  echo:
    cmds:
      - echo "This will print nothing" > /dev/null
```

## Dry run mode

Dry run mode (`--dry`) compiles and steps through each task, printing the commands
that would be run without executing them. This is useful for debugging your Taskfiles.

## Ignore errors

You have the option to ignore errors during command execution.
Given the following Taskfile:

```yaml
version: '2'

tasks:
  echo:
    cmds:
      - exit 1
      - echo "Hello World"
```

Task will abort the execution after running `exit 1` because the status code `1` stands for `EXIT_FAILURE`.
However it is possible to continue with execution using `ignore_error`:

```yaml
version: '2'

tasks:
  echo:
    cmds:
      - cmd: exit 1
        ignore_error: true
      - echo "Hello World"
```

`ignore_error` can also be set for a task, which mean errors will be suppressed
for all commands. But keep in mind this option won't propagate to other tasks
called either by `deps` or `cmds`!

## Output syntax

By default, Task just redirect the STDOUT and STDERR of the running commands
to the shell in real time. This is good for having live feedback for log
printed by commands, but the output can become messy if you have multiple
commands running at the same time and printing lots of stuff.

To make this more customizable, there are currently three different output
options you can choose:

- `interleaved` (default)
- `group`
- `prefixed`

To choose another one, just set it to root in the Taskfile:

```yaml
version: '2'

output: 'group'

tasks:
  # ...
```

 The `group` output will print the entire output of a command once, after it
 finishes, so you won't have live feedback for commands that take a long time
 to run.

 The `prefix` output will prefix every line printed by a command with
 `[task-name] ` as the prefix, but you can customize the prefix for a command
 with the `prefix:` attribute:

 ```yaml
version: '2'

output: prefixed

tasks:
  default:
    deps:
      - task: print
        vars: {TEXT: foo}
      - task: print
        vars: {TEXT: bar}
      - task: print
        vars: {TEXT: baz}

  print:
    cmds:
      - echo "{{.TEXT}}"
    prefix: "print-{{.TEXT}}"
    silent: true
```

```bash
$ task default
[print-foo] foo
[print-bar] bar
[print-baz] baz
```

## Watch tasks

If you give a `--watch` or `-w` argument, task will watch for file changes
and run the task again. This requires the `sources` attribute to be given,
so task know which files to watch.

[gotemplate]: https://golang.org/pkg/text/template/
[minify]: https://github.com/tdewolff/minify/tree/master/cmd/minify
