---
slug: /usage/
sidebar_position: 3
---

# Usage

## Getting started

Create a file called `Taskfile.yml` in the root of your project.
The `cmds` attribute should contain the commands of a task.
The example below allows compiling a Go app and uses [Minify][minify] to concat
and minify multiple CSS files into a single one.

```yaml
version: '3'

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

Task uses [mvdan.cc/sh](https://mvdan.cc/sh/), a native Go sh
interpreter. So you can write sh/bash commands, and it will work even on
Windows, where `sh` or `bash` are usually not available. Just remember any
executable called must be available by the OS or in PATH.

If you omit a task name, "default" will be assumed.

## Supported file names

Task will look for the following file names, in order of priority:

- Taskfile.yml
- Taskfile.yaml
- Taskfile.dist.yml
- Taskfile.dist.yaml

The intention of having the `.dist` variants is to allow projects to have one
committed version (`.dist`) while still allowing individual users to override
the Taskfile by adding an additional `Taskfile.yml` (which would be on
`.gitignore`).

## Environment variables

### Task

You can use `env` to set custom environment variables for a specific task:

```yaml
version: '3'

tasks:
  greet:
    cmds:
      - echo $GREETING
    env:
      GREETING: Hey, there!
```

Additionally, you can set global environment variables that will be available
to all tasks:

```yaml
version: '3'

env:
  GREETING: Hey, there!

tasks:
  greet:
    cmds:
      - echo $GREETING
```

:::info

`env` supports expansion and retrieving output from a shell command
just like variables, as you can see in the [Variables](#variables) section.

:::

### .env files

You can also ask Task to include `.env` like files by using the `dotenv:`
setting:

```bash title=".env"
KEYNAME=VALUE
```

```bash title="testing/.env"
ENDPOINT=testing.com
```

```yaml title="Taskfile.yml"
version: '3'

env:
  ENV: testing

dotenv: ['.env', '{{.ENV}}/.env.', '{{.HOME}}/.env']

tasks:
  greet:
    cmds:
      - echo "Using $KEYNAME and endpoint $ENDPOINT"
```

## Including other Taskfiles

If you want to share tasks between different projects (Taskfiles), you can use
the importing mechanism to include other Taskfiles using the `includes` keyword:

```yaml
version: '3'

includes:
  docs: ./documentation # will look for ./documentation/Taskfile.yml
  docker: ./DockerTasks.yml
```

The tasks described in the given Taskfiles will be available with the informed
namespace. So, you'd call `task docs:serve` to run the `serve` task from
`documentation/Taskfile.yml` or `task docker:build` to run the `build` task
from the `DockerTasks.yml` file.

Relative paths are resolved relative to the directory containing the including Taskfile.

### OS-specific Taskfiles

With `version: '2'`, task automatically includes any `Taskfile_{{OS}}.yml`
if it exists (for example: `Taskfile_windows.yml`, `Taskfile_linux.yml` or
`Taskfile_darwin.yml`). Since this behavior was a bit too implicit, it
was removed on version 3, but you still can have a similar behavior by
explicitly importing these files:

```yaml
version: '3'

includes:
  build: ./Taskfile_{{OS}}.yml
```

### Directory of included Taskfile

By default, included Taskfile's tasks are run in the current directory, even
if the Taskfile is in another directory, but you can force its tasks to run
in another directory by using this alternative syntax:

```yaml
version: '3'

includes:
  docs:
    taskfile: ./docs/Taskfile.yml
    dir: ./docs
```

:::info

The included Taskfiles must be using the same schema version as the main
Taskfile uses.

:::

### Optional includes

Includes marked as optional will allow Task to continue execution as normal if
the included file is missing.

```yaml
version: '3'

includes:
  tests:
    taskfile: ./tests/Taskfile.yml
    optional: true

tasks:
  greet:
    cmds:
      - echo "This command can still be successfully executed if ./tests/Taskfile.yml does not exist"
```

### Internal includes

Includes marked as internal will set all the tasks of the included file to be
internal as well (see the [Internal tasks](#internal-tasks) section below).
This is useful when including utility tasks that are not intended to be used
directly by the user.

```yaml
version: '3'

includes:
  tests:
    taskfile: ./taskfiles/Utils.yml
    internal: true
```

### Vars of included Taskfiles

You can also specify variables when including a Taskfile. This may be useful
for having reusable Taskfile that can be tweaked or even included more than once:

```yaml
version: '3'

includes:
  backend:
    taskfile: ./taskfiles/Docker.yml
    vars:
      DOCKER_IMAGE: backend_image

  frontend:
    taskfile: ./taskfiles/Docker.yml
    vars:
      DOCKER_IMAGE: frontend_image
```

### Namespace aliases

When including a Taskfile, you can give the namespace a list of `aliases`.
This works in the same way as [task aliases](#task-aliases) and can be used
together to create shorter and easier-to-type commands.

```yaml
version: '3'

includes:
  generate:
    taskfile: ./taskfiles/Generate.yml
    aliases: [gen]
```

:::info

Vars declared in the included Taskfile have preference over the
variables in the including Taskfile! If you want a variable in an included Taskfile to be overridable,
use the [default function](https://go-task.github.io/slim-sprig/defaults.html):
`MY_VAR: '{{.MY_VAR | default "my-default-value"}}'`.

:::

## Internal tasks

Internal tasks are tasks that cannot be called directly by the user. They will
not appear in the output when running `task --list|--list-all`. Other tasks may
call internal tasks in the usual way. This is useful for creating reusable,
function-like tasks that have no useful purpose on the command line.

```yaml
version: '3'

tasks:
  build-image-1:
    cmds:
      - task: build-image
        vars:
          DOCKER_IMAGE: image-1

  build-image:
    internal: true
    cmds:
      - docker build -t {{.DOCKER_IMAGE}} .
```

## Task directory

By default, tasks will be executed in the directory where the Taskfile is
located. But you can easily make the task run in another folder, informing
`dir`:

```yaml
version: '3'

tasks:
  serve:
    dir: public/www
    cmds:
      # run http server
      - caddy
```

If the directory does not exist, `task` creates it.

## Task dependencies

> Dependencies run in parallel, so dependencies of a task should not depend one
> another. If you want to force tasks to run serially, take a look at the
> [Calling Another Task](#calling-another-task) section below.

You may have tasks that depend on others. Just pointing them on `deps` will
make them run automatically before running the parent task:

```yaml
version: '3'

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
version: '3'

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

:::tip

You can also make the tasks given by the command line run in parallel by
using the `--parallel` flag (alias `-p`). Example: `task --parallel js css`.

:::

If you want to pass information to dependencies, you can do that the same
manner as you would to [call another task](#calling-another-task):

```yaml
version: '3'

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
often result in a faster build pipeline. However, in some situations, you may need
to call other tasks serially. In this case, use the following syntax:

```yaml
version: '3'

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
version: '3'

tasks:
  greet:
    vars:
      RECIPIENT: '{{default "World" .RECIPIENT}}'
    cmds:
      - echo "Hello, {{.RECIPIENT}}!"

  greet-pessimistically:
    cmds:
      - task: greet
        vars: {RECIPIENT: "Cruel World"}
```

The above syntax is also supported in `deps`.

:::tip

NOTE: If you want to call a task declared in the root Taskfile from within an
[included Taskfile](#including-other-taskfiles), add a leading `:` like this:
`task: :task-name`.

:::

## Prevent unnecessary work

### By fingerprinting locally generated files and their sources

If a task generates something, you can inform Task the source and generated
files, so Task will prevent running them if not necessary.

```yaml
version: '3'

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

`sources` and `generates` can be files or file patterns. When given,
Task will compare the checksum of the source files to determine if it's
necessary to run the task. If not, it will just print a message like
`Task "js" is up to date`.

If you prefer this check to be made by the modification timestamp of the files,
instead of its checksum (content), just set the `method` property to `timestamp`.

```yaml
version: '3'

tasks:
  build:
    cmds:
      - go build .
    sources:
      - ./*.go
    generates:
      - app{{exeExt}}
    method: timestamp
```

In situations where you need more flexibility the `status` keyword can be used.
You can even combine the two. See the documentation for
[status](#using-programmatic-checks-to-indicate-a-task-is-up-to-date) for an
example.

:::info

By default, task stores checksums on a local `.task` directory in the project's
directory. Most of the time, you'll want to have this directory on `.gitignore`
(or equivalent) so it isn't committed. (If you have a task for code generation
that is committed it may make sense to commit the checksum of that task as
well, though).

If you want these files to be stored in another directory, you can set a
`TASK_TEMP_DIR` environment variable in your machine. It can contain a relative
path like `tmp/task` that will be interpreted as relative to the project
directory, or an absolute or home path like `/tmp/.task` or `~/.task`
(subdirectories will be created for each project).

```bash
export TASK_TEMP_DIR='~/.task'
```

:::

:::info

Each task has only one checksum stored for its `sources`. If you want
to distinguish a task by any of its input variables, you can add those
variables as part of the task's label, and it will be considered a different
task.

This is useful if you want to run a task once for each distinct set of
inputs until the sources actually change. For example, if the sources depend
on the value of a variable, or you if you want the task to rerun if some arguments
change even if the source has not.

:::

:::tip

The method `none` skips any validation and always run the task.

:::

:::info

For the `checksum` (default) method to work, it is only necessary to
inform the source files, but if you want to use the `timestamp` method, you
also need to inform the generated files with `generates`.

:::

### Using programmatic checks to indicate a task is up to date.

Alternatively, you can inform a sequence of tests as `status`. If no error
is returned (exit status 0), the task is considered up-to-date:

```yaml
version: '3'

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

Normally, you would use `sources` in combination with
`generates` - but for tasks that generate remote artifacts (Docker images,
deploys, CD releases) the checksum source and timestamps require either
access to the artifact or for an out-of-band refresh of the `.checksum`
fingerprint file.

Two special variables `{{.CHECKSUM}}` and `{{.TIMESTAMP}}` are available
for interpolation within `status` commands, depending on the method assigned
to fingerprint the sources. Only `source` globs are fingerprinted.

Note that the `{{.TIMESTAMP}}` variable is a "live" Go `time.Time` struct, and
can be formatted using any of the methods that `time.Time` responds to.

See [the Go Time documentation](https://golang.org/pkg/time/) for more information.

You can use `--force` or `-f` if you want to force a task to run even when
up-to-date.

Also, `task --status [tasks]...` will exit with a non-zero exit code if any of
the tasks are not up-to-date.

`status` can be combined with the [fingerprinting](#by-fingerprinting-locally-generated-files-and-their-sources)
to have a task run if either the the source/generated artifacts changes, or the
programmatic check fails:

```yaml
version: '3'

tasks:
  build:prod:
    desc: Build for production usage.
    cmds:
      - composer install
    # Run this task if source files changes.
    sources:
      - composer.json
      - composer.lock
    generates:
      - ./vendor/composer/installed.json
      - ./vendor/autoload.php
    # But also run the task if the last build was not a production build.
    status:
      - grep -q '"dev": false' ./vendor/composer/installed.json
```

### Using programmatic checks to cancel the execution of a task and its dependencies

In addition to `status` checks, `preconditions` checks are
the logical inverse of `status` checks.  That is, if you need a certain set of
conditions to be _true_ you can use the `preconditions` stanza.
`preconditions` are similar to `status` lines, except they support `sh`
expansion, and they SHOULD all return 0.

```yaml
version: '3'

tasks:
  generate-files:
    cmds:
      - mkdir directory
      - touch directory/file1.txt
      - touch directory/file2.txt
    # test existence of files
    preconditions:
      - test -f .env
      - sh: "[ 1 = 0 ]"
        msg: "One doesn't equal Zero, Halting"
```

Preconditions can set specific failure messages that can tell
a user what steps to take using the `msg` field.

If a task has a dependency on a sub-task with a precondition, and that
precondition is not met - the calling task will fail.  Note that a task
executed with a failing precondition will not run unless `--force` is
given.

Unlike `status`, which will skip a task if it is up to date and continue
executing tasks that depend on it, a `precondition` will fail a task, along
with any other tasks that depend on it.

```yaml
version: '3'

tasks:
  task-will-fail:
    preconditions:
      - sh: "exit 1"

  task-will-also-fail:
    deps:
      - task-will-fail

  task-will-still-fail:
    cmds:
      - task: task-will-fail
      - echo "I will not run"
```

### Limiting when tasks run

If a task executed by multiple `cmds` or multiple `deps` you can control
when it is executed using `run`. `run` can also be set at the root
of the Taskfile to change the behavior of all the tasks unless explicitly
overridden.

Supported values for `run`:

 * `always` (default) always attempt to invoke the task regardless of the
  number of previous executions
 * `once` only invoke this task once regardless of the number of references
 * `when_changed` only invokes the task once for each unique set of variables
  passed into the task

```yaml
version: '3'

tasks:
  default:
    cmds:
      - task: generate-file
        vars: { CONTENT: '1' }
      - task: generate-file
        vars: { CONTENT: '2' }
      - task: generate-file
        vars: { CONTENT: '2' }

  generate-file:
    run: when_changed
    deps:
      - install-deps
    cmds:
      - echo {{.CONTENT}}

  install-deps:
    run: once
    cmds:
      - sleep 5 # long operation like installing packages
```

## Variables

When doing interpolation of variables, Task will look for the below.
They are listed below in order of importance (i.e. most important first):

- Variables declared in the task definition
- Variables given while calling a task from another
  (See [Calling another task](#calling-another-task) above)
- Variables of the [included Taskfile](#including-other-taskfiles) (when the task is included)
- Variables of the [inclusion of the Taskfile](#vars-of-included-taskfiles) (when the task is included)
- Global variables (those declared in the `vars:` option in the Taskfile)
- Environment variables

Example of sending parameters with environment variables:

```bash
$ TASK_VARIABLE=a-value task do-something
```

:::tip

A special variable `.TASK` is always available containing the task name.

:::

Since some shells do not support the above syntax to set environment variables
(Windows) tasks also accept a similar style when not at the beginning of
the command.

```bash
$ task write-file FILE=file.txt "CONTENT=Hello, World!" print "MESSAGE=All done!"
```

Example of locally declared vars:

```yaml
version: '3'

tasks:
  print-var:
    cmds:
      - echo "{{.VAR}}"
    vars:
      VAR: Hello!
```

Example of global vars in a `Taskfile.yml`:

```yaml
version: '3'

vars:
  GREETING: Hello from Taskfile!

tasks:
  greet:
    cmds:
      - echo "{{.GREETING}}"
```

### Dynamic variables

The below syntax (`sh:` prop in a variable) is considered a dynamic variable.
The value will be treated as a command and the output assigned. If there are one
or more trailing newlines, the last newline will be trimmed.

```yaml
version: '3'

tasks:
  build:
    cmds:
      - go build -ldflags="-X main.Version={{.GIT_COMMIT}}" main.go
    vars:
      GIT_COMMIT:
        sh: git log -n 1 --format=%h
```

This works for all types of variables.

## Forwarding CLI arguments to commands

If `--` is given in the CLI, all following parameters are added to a
special `.CLI_ARGS` variable. This is useful to forward arguments to another
command.

The below example will run `yarn install`.

```bash
$ task yarn -- install
```

```yaml
version: '3'

tasks:
  yarn:
    cmds:
      - yarn {{.CLI_ARGS}}
```

## Doing task cleanup with `defer`

With the `defer` keyword, it's possible to schedule cleanup to be run once
the task finishes. The difference with just putting it as the last command is
that this command will run even when the task fails.

In the example below, `rm -rf tmpdir/` will run even if the third command fails:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - mkdir -p tmpdir/
      - defer: rm -rf tmpdir/
      - echo 'Do work on tmpdir/'
```

If you want to move the cleanup command into another task, that is possible as
well:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - mkdir -p tmpdir/
      - defer: { task: cleanup }
      - echo 'Do work on tmpdir/'

  cleanup: rm -rf tmpdir/
```

:::info

Due to the nature of how the
[Go's own `defer` work](https://go.dev/tour/flowcontrol/13), the deferred
commands are executed in the reverse order if you schedule multiple of them.

:::

## Go's template engine

Task parse commands as [Go's template engine][gotemplate] before executing
them. Variables are accessible through dot syntax (`.VARNAME`).

All functions by the Go's [slim-sprig lib](https://go-task.github.io/slim-sprig/)
are available. The following example gets the current date in a given format:

```yaml
version: '3'

tasks:
  print-date:
    cmds:
      - echo {{now | date "2006-01-02"}}
```

Task also adds the following functions:

- `OS`: Returns the operating system. Possible values are "windows", "linux",
  "darwin" (macOS) and "freebsd".
- `ARCH`: return the architecture Task was compiled to: "386", "amd64", "arm"
  or "s390x".
- `splitLines`: Splits Unix (\n) and Windows (\r\n) styled newlines.
- `catLines`: Replaces Unix (\n) and Windows (\r\n) styled newlines with a space.
- `toSlash`: Does nothing on Unix, but on Windows converts a string from `\`
  path format to `/`.
- `fromSlash`: Opposite of `toSlash`. Does nothing on Unix, but on Windows
  converts a string from `/` path format to `\`.
- `exeExt`: Returns the right executable extension for the current OS
  (`".exe"` for Windows, `""` for others).
- `shellQuote`: Quotes a string to make it safe for use in shell scripts.
  Task uses [this Go function](https://pkg.go.dev/mvdan.cc/sh/v3@v3.4.0/syntax#Quote)
  for this. The Bash dialect is assumed.

Example:

```yaml
version: '3'

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
version: '3'

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

If you want to see all tasks, there's a `--list-all` (alias `-a`) flag as well.

## Display summary of task

Running `task --summary task-name` will show a summary of a task.
The following Taskfile:

```yaml
version: '3'

tasks:
  release:
    deps: [build]
    summary: |
      Release your project to github

      It will build your project before starting the release.
      Please make sure that you have set GITHUB_TOKEN before starting.
    cmds:
      - your-release-tool

  build:
    cmds:
      - your-build-tool
```

with running ``task --summary release`` would print the following output:

```
task: release

Release your project to github

It will build your project before starting the release.
Please make sure that you have set GITHUB_TOKEN before starting.

dependencies:
 - build

commands:
 - your-release-tool
```
If a summary is missing, the description will be printed.
If the task does not have a summary or a description, a warning is printed.

Please note: *showing the summary will not execute the command*.

## Task aliases

Aliases are alternative names for tasks. They can be used to make it easier and
quicker to run tasks with long or hard-to-type names. You can use them on the
command line, when [calling sub-tasks](#calling-another-task) in your Taskfile
and when [including tasks](#including-other-taskfiles) with aliases from another
Taskfile. They can also be used together with [namespace
aliases](#namespace-aliases).

```yaml
version: '3'

tasks:
  generate:
    aliases: [gen]
    cmds:
      - task: gen-mocks

  generate-mocks:
    aliases: [gen-mocks]
    cmds:
      - echo "generating..."
```

## Overriding task name

Sometimes you may want to override the task name printed on the summary, up-to-date
messages to STDOUT, etc. In this case, you can just set `label:`, which can also
be interpolated with variables:

```yaml
version: '3'

tasks:
  default:
    - task: print
      vars:
        MESSAGE: hello
    - task: print
      vars:
        MESSAGE: world

  print:
    label: 'print-{{.MESSAGE}}'
    cmds:
      - echo "{{.MESSAGE}}"
```

## Silent mode

Silent mode disables the echoing of commands before Task runs it.
For the following Taskfile:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "Print something"
```

Normally this will be printed:

```sh
echo "Print something"
Print something
```

With silent mode on, the below will be printed instead:

```sh
Print something
```

There are four ways to enable silent mode:

* At command level:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - cmd: echo "Print something"
        silent: true
```

* At task level:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "Print something"
    silent: true
```

* Globally at Taskfile level:

```yaml
version: '3'

silent: true

tasks:
  echo:
    cmds:
      - echo "Print something"
```

* Or globally with `--silent` or `-s` flag

If you want to suppress STDOUT instead, just redirect a command to `/dev/null`:

```yaml
version: '3'

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
version: '3'

tasks:
  echo:
    cmds:
      - exit 1
      - echo "Hello World"
```

Task will abort the execution after running `exit 1` because the status code `1` stands for `EXIT_FAILURE`.
However, it is possible to continue with execution using `ignore_error`:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - cmd: exit 1
        ignore_error: true
      - echo "Hello World"
```

`ignore_error` can also be set for a task, which means errors will be suppressed
for all commands. Nevertheless, keep in mind that this option will not propagate to other tasks
called either by `deps` or `cmds`!

## Output syntax

By default, Task just redirects the STDOUT and STDERR of the running commands
to the shell in real-time. This is good for having live feedback for logging
printed by commands, but the output can become messy if you have multiple
commands running simultaneously and printing lots of stuff.

To make this more customizable, there are currently three different output
options you can choose:

- `interleaved` (default)
- `group`
- `prefixed`

To choose another one, just set it to root in the Taskfile:

```yaml
version: '3'

output: 'group'

tasks:
  # ...
```

The `group` output will print the entire output of a command once after it
finishes, so you will not have live feedback for commands that take a long time
to run.

When using the `group` output, you can optionally provide a templated message
to print at the start and end of the group. This can be useful for instructing
CI systems to group all of the output for a given task, such as with
[GitHub Actions' `::group::` command](https://docs.github.com/en/actions/learn-github-actions/workflow-commands-for-github-actions#grouping-log-lines)
or [Azure Pipelines](https://docs.microsoft.com/en-us/azure/devops/pipelines/scripts/logging-commands?expand=1&view=azure-devops&tabs=bash#formatting-commands).

```yaml
version: '3'

output:
  group:
    begin: '::group::{{.TASK}}'
    end: '::endgroup::'

tasks:
  default:
    cmds:
      - echo 'Hello, World!'
    silent: true
```

```bash
$ task default
::group::default
Hello, World!
::endgroup::
```

The `prefix` output will prefix every line printed by a command with
`[task-name] ` as the prefix, but you can customize the prefix for a command
with the `prefix:` attribute:

 ```yaml
version: '3'

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

:::tip

The `output` option can also be specified by the `--output` or `-o` flags.

:::

## Interactive CLI application

When running interactive CLI applications inside Task they can sometimes behave
weirdly, especially when the [output mode](#output-syntax) is set to something
other than `interleaved` (the default), or when interactive apps are run in
parallel with other tasks.

The `interactive: true` tells Task this is an interactive application and Task
will try to optimize for it:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - vim my-file.txt
    interactive: true
```

If you still have problems running an interactive app through Task, please open
an issue about it.

## Short task syntax

Starting on Task v3, you can now write tasks with a shorter syntax if they
have the default settings (e.g. no custom `env:`, `vars:`, `desc:`, `silent:` , etc):

```yaml
version: '3'

tasks:
  build: go build -v -o ./app{{exeExt}} .

  run:
    - task: build
    - ./app{{exeExt}} -h localhost -p 8080
```

## Watch tasks

With the flags `--watch` or `-w` task will watch for file changes
and run the task again. This requires the `sources` attribute to be given,
so task knows which files to watch.

The default watch interval is 5 seconds, but it's possible to change it by
either setting `interval: '500ms'` in the root of the Taskfile passing it
as an argument like `--interval=500ms`.

[gotemplate]: https://golang.org/pkg/text/template/
[minify]: https://github.com/tdewolff/minify/tree/master/cmd/minify
