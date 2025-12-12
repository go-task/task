---
outline: deep
---

# Guide

## Running Taskfiles

Specific Taskfiles can be called by specifying the `--taskfile` flag. If you
don't specify a Taskfile, Task will automatically look for a file with one of
the [supported file names](#supported-file-names) in the current directory. If
you want to search in a different directory, you can use the `--dir` flag.

### Supported file names

Task looks for files with the following names, in order of priority:

- `Taskfile.yml`
- `taskfile.yml`
- `Taskfile.yaml`
- `taskfile.yaml`
- `Taskfile.dist.yml`
- `taskfile.dist.yml`
- `Taskfile.dist.yaml`
- `taskfile.dist.yaml`

The `.dist` variants allow projects to have one committed file (`.dist`) while
still allowing individual users to override the Taskfile by adding an additional
`Taskfile.yml` (which would be in your `.gitignore`).

### Running a Taskfile from a subdirectory

If a Taskfile cannot be found in the current working directory, it will walk up
the file tree until it finds one (similar to how `git` works). When running Task
from a subdirectory like this, it will behave as if you ran it from the
directory containing the Taskfile.

You can use this functionality along with the special
<span v-pre>`{{.USER_WORKING_DIR}}`</span> variable to create some very useful
reusable tasks. For example, if you have a monorepo with directories for each
microservice, you can `cd` into a microservice directory and run a task command
to bring it up without having to create multiple tasks or Taskfiles with
identical content. For example:

```yaml
version: '3'

tasks:
  up:
    dir: '{{.USER_WORKING_DIR}}'
    preconditions:
      - test -f docker-compose.yml
    cmds:
      - docker-compose up -d
```

In this example, we can run `cd <service>` and `task up` and as long as the
`<service>` directory contains a `docker-compose.yml`, the Docker composition
will be brought up.

### Running a global Taskfile

If you call Task with the `--global` (alias `-g`) flag, it will look for your
home directory instead of your working directory. In short, Task will look for a
Taskfile that matches `$HOME/{T,t}askfile.{yml,yaml}` .

This is useful to have automation that you can run from anywhere in your system!

::: info

When running your global Taskfile with `-g`, tasks will run on `$HOME` by
default, and not on your working directory!

As mentioned in the previous section, the
<span v-pre>`{{.USER_WORKING_DIR}}`</span> special variable can be very handy
here to run stuff on the directory you're calling `task -g` from.

```yaml
version: '3'

tasks:
  from-home:
    cmds:
      - pwd

  from-working-directory:
    dir: '{{.USER_WORKING_DIR}}'
    cmds:
      - pwd
```

:::

### Reading a Taskfile from stdin

Taskfile also supports reading from stdin. This is useful if you are generating
Taskfiles dynamically and don't want write them to disk. To tell task to read
from stdin, you must specify the `-t/--taskfile` flag with the special `-`
value. You may then pipe into Task as you would any other program:

```shell
task -t - <(cat ./Taskfile.yml)
# OR
cat ./Taskfile.yml | task -t -
```

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

Additionally, you can set global environment variables that will be available to
all tasks:

```yaml
version: '3'

env:
  GREETING: Hey, there!

tasks:
  greet:
    cmds:
      - echo $GREETING
```

::: info

`env` supports expansion and retrieving output from a shell command just like
variables, as you can see in the [Variables](#variables) section.

:::

### .env files

You can also ask Task to include `.env` like files by using the `dotenv:`
setting:

::: code-group

```shell [.env]
KEYNAME=VALUE
```

```shell [testing/.env]
ENDPOINT=testing.com
```

:::

```yaml
version: '3'

env:
  ENV: testing

dotenv: ['.env', '{{.ENV}}/.env', '{{.HOME}}/.env']

tasks:
  greet:
    cmds:
      - echo "Using $KEYNAME and endpoint $ENDPOINT"
```

Dotenv files can also be specified at the task level:

```yaml
version: '3'

env:
  ENV: testing

tasks:
  greet:
    dotenv: ['.env', '{{.ENV}}/.env', '{{.HOME}}/.env']
    cmds:
      - echo "Using $KEYNAME and endpoint $ENDPOINT"
```

Environment variables specified explicitly at the task-level will override
variables defined in dotfiles:

```yaml
version: '3'

env:
  ENV: testing

tasks:
  greet:
    dotenv: ['.env', '{{.ENV}}/.env', '{{.HOME}}/.env']
    env:
      KEYNAME: DIFFERENT_VALUE
    cmds:
      - echo "Using $KEYNAME and endpoint $ENDPOINT"
```

::: info

Please note that you are not currently able to use the `dotenv` key inside
included Taskfiles.

:::

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
`documentation/Taskfile.yml` or `task docker:build` to run the `build` task from
the `DockerTasks.yml` file.

Relative paths are resolved relative to the directory containing the including
Taskfile.

### OS-specific Taskfiles

You can include OS-specific Taskfiles by using a templating function:

```yaml
version: '3'

includes:
  build: ./Taskfile_{{OS}}.yml
```

### Directory of included Taskfile

By default, included Taskfile's tasks are run in the current directory, even if
the Taskfile is in another directory, but you can force its tasks to run in
another directory by using this alternative syntax:

```yaml
version: '3'

includes:
  docs:
    taskfile: ./docs/Taskfile.yml
    dir: ./docs
```

::: info

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
      - echo "This command can still be successfully executed if
        ./tests/Taskfile.yml does not exist"
```

### Internal includes

Includes marked as internal will set all the tasks of the included file to be
internal as well (see the [Internal tasks](#internal-tasks) section below). This
is useful when including utility tasks that are not intended to be used directly
by the user.

```yaml
version: '3'

includes:
  tests:
    taskfile: ./taskfiles/Utils.yml
    internal: true
```

### Flatten includes

You can flatten the included Taskfile tasks into the main Taskfile by using the
`flatten` option. It means that the included Taskfile tasks will be available
without the namespace.

::: code-group

```yaml [Taskfile.yml]
version: '3'

includes:
  lib:
    taskfile: ./Included.yml
    flatten: true

tasks:
  greet:
    cmds:
      - echo "Greet"
      - task: foo
```

```yaml [Included.yml]
version: '3'

tasks:
  foo:
    cmds:
      - echo "Foo"
```

:::

If you run `task -a` it will print :

```sh
task: Available tasks for this project:
* greet:
* foo
```

You can run `task foo` directly without the namespace.

You can also reference the task in other tasks without the namespace. So if you
run `task greet` it will run `greet` and `foo` tasks and the output will be :

```text
Greet
Foo
```

If multiple tasks have the same name, an error will be thrown:

::: code-group

```yaml [Taskfile.yml]
version: '3'
includes:
  lib:
    taskfile: ./Included.yml
    flatten: true

tasks:
  greet:
    cmds:
      - echo "Greet"
      - task: foo
```

```yaml [Included.yml]
version: '3'

tasks:
  greet:
    cmds:
      - echo "Foo"
```

:::

If you run `task -a` it will print:

```text
task: Found multiple tasks (greet) included by "lib"
```

If the included Taskfile has a task with the same name as a task in the main
Taskfile, you may want to exclude it from the flattened tasks.

You can do this by using the
[`excludes` option](#exclude-tasks-from-being-included).

### Exclude tasks from being included

You can exclude tasks from being included by using the `excludes` option. This
option takes the list of tasks to be excluded from this include.

::: code-group

```yaml [Taskfile.yml]
version: '3'

includes:
  included:
    taskfile: ./Included.yml
    excludes: [foo]
```

```yaml [Included.yml]
version: '3'

tasks:
  foo: echo "Foo"
  bar: echo "Bar"
```

:::

`task included:foo` will throw an error because the `foo` task is excluded but
`task included:bar` will work and display `Bar`.

It's compatible with the `flatten` option.

### Vars of included Taskfiles

You can also specify variables when including a Taskfile. This may be useful for
having a reusable Taskfile that can be tweaked or even included more than once:

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

When including a Taskfile, you can give the namespace a list of `aliases`. This
works in the same way as [task aliases](#task-aliases) and can be used together
to create shorter and easier-to-type commands.

```yaml
version: '3'

includes:
  generate:
    taskfile: ./taskfiles/Generate.yml
    aliases: [gen]
```

::: info

Vars declared in the included Taskfile have preference over the variables in the
including Taskfile! If you want a variable in an included Taskfile to be
overridable, use the
[default function](https://sprig.taskfile.dev/defaults.html):
<span v-pre>`MY_VAR: '{{.MY_VAR | default "my-default-value"}}'`</span>.

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

You may have tasks that depend on others. Just pointing them on `deps` will make
them run automatically before running the parent task:

```yaml
version: '3'

tasks:
  build:
    deps: [assets]
    cmds:
      - go build -v -i main.go

  assets:
    cmds:
      - esbuild --bundle --minify css/index.css > public/bundle.css
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
      - esbuild --bundle --minify js/index.js > public/bundle.js

  css:
    cmds:
      - esbuild --bundle --minify css/index.css > public/bundle.css
```

If there is more than one dependency, they always run in parallel for better
performance.

::: tip

You can also make the tasks given by the command line run in parallel by using
the `--parallel` flag (alias `-p`). Example: `task --parallel js css`.

:::

If you want to pass information to dependencies, you can do that the same manner
as you would to [call another task](#calling-another-task):

```yaml
version: '3'

tasks:
  default:
    deps:
      - task: echo_sth
        vars: { TEXT: 'before 1' }
      - task: echo_sth
        vars: { TEXT: 'before 2' }
        silent: true
    cmds:
      - echo "after"

  echo_sth:
    cmds:
      - echo {{.TEXT}}
```

### Fail-fast dependencies

By default, Task waits for all dependencies to finish running before continuing.
If you want Task to stop executing further dependencies as soon as one fails,
you can set `failfast: true` on your [`.taskrc.yml`][config] or for a specific
task:

```yaml
# .taskrc.yml
failfast: true # applies to all tasks
```

```yaml
# Taskfile.yml
version: '3'

tasks:
  default:
    deps: [task1, task2, task3]
    failfast: true # applies only to this task
```

Alternatively, you can use `--failfast`, which also work for `--parallel`.

## Platform specific tasks and commands

If you want to restrict the running of tasks to explicit platforms, this can be
achieved using the `platforms:` key. Tasks can be restricted to a specific OS,
architecture or a combination of both. On a mismatch, the task or command will
be skipped, and no error will be thrown.

The values allowed as OS or Arch are valid `GOOS` and `GOARCH` values, as
defined by the Go language
[here](https://github.com/golang/go/blob/master/src/internal/syslist/syslist.go).

The `build-windows` task below will run only on Windows, and on any
architecture:

```yaml
version: '3'

tasks:
  build-windows:
    platforms: [windows]
    cmds:
      - echo 'Running command on Windows'
```

This can be restricted to a specific architecture as follows:

```yaml
version: '3'

tasks:
  build-windows-amd64:
    platforms: [windows/amd64]
    cmds:
      - echo 'Running command on Windows (amd64)'
```

It is also possible to restrict the task to specific architectures:

```yaml
version: '3'

tasks:
  build-amd64:
    platforms: [amd64]
    cmds:
      - echo 'Running command on amd64'
```

Multiple platforms can be specified as follows:

```yaml
version: '3'

tasks:
  build:
    platforms: [windows/amd64, darwin]
    cmds:
      - echo 'Running command on Windows (amd64) and macOS'
```

Individual commands can also be restricted to specific platforms:

```yaml
version: '3'

tasks:
  build:
    cmds:
      - cmd: echo 'Running command on Windows (amd64) and macOS'
        platforms: [windows/amd64, darwin]
      - cmd: echo 'Running on all platforms'
```

## Calling another task

When a task has many dependencies, they are executed concurrently. This will
often result in a faster build pipeline. However, in some situations, you may
need to call other tasks serially. In this case, use the following syntax:

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

Using the `vars` and `silent` attributes you can choose to pass variables and
toggle [silent mode](#silent-mode) on a call-by-call basis:

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
        vars: { RECIPIENT: 'Cruel World' }
        silent: true
```

The above syntax is also supported in `deps`.

::: tip

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
      - esbuild --bundle --minify js/index.js > public/bundle.js
    sources:
      - src/js/**/*.js
    generates:
      - public/bundle.js

  css:
    cmds:
      - esbuild --bundle --minify css/index.css > public/bundle.css
    sources:
      - src/css/**/*.css
    generates:
      - public/bundle.css
```

`sources` and `generates` can be files or glob patterns. When given, Task will
compare the checksum of the source files to determine if it's necessary to run
the task. If not, it will just print a message like `Task "js" is up to date`.

`exclude:` can also be used to exclude files from fingerprinting. Sources are
evaluated in order, so `exclude:` must come after the positive glob it is
negating.

```yaml
version: '3'

tasks:
  css:
    sources:
      - mysources/**/*.css
      - exclude: mysources/ignoreme.css
    generates:
      - public/bundle.css
```

If you prefer these check to be made by the modification timestamp of the files,
instead of its checksum (content), just set the `method` property to
`timestamp`. This can be done at two levels:

At the task level for a specific task:

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

At the root level of the Taskfile to apply it globally to all tasks:

```yaml
version: '3'

method: timestamp # Will be the default for all tasks

tasks:
  build:
    cmds:
      - go build .
    sources:
      - ./*.go
    generates:
      - app{{exeExt}}
```

In situations where you need more flexibility the `status` keyword can be used.
You can even combine the two. See the documentation for
[status](#using-programmatic-checks-to-indicate-a-task-is-up-to-date) for an
example.

::: info

By default, task stores checksums on a local `.task` directory in the project's
directory. Most of the time, you'll want to have this directory on `.gitignore`
(or equivalent) so it isn't committed. (If you have a task for code generation
that is committed it may make sense to commit the checksum of that task as well,
though).

If you want these files to be stored in another directory, you can set a
`TASK_TEMP_DIR` environment variable in your machine. It can contain a relative
path like `tmp/task` that will be interpreted as relative to the project
directory, or an absolute or home path like `/tmp/.task` or `~/.task`
(subdirectories will be created for each project).

```shell
export TASK_TEMP_DIR='~/.task'
```

:::

::: info

Each task has only one checksum stored for its `sources`. If you want to
distinguish a task by any of its input variables, you can add those variables as
part of the task's label, and it will be considered a different task.

This is useful if you want to run a task once for each distinct set of inputs
until the sources actually change. For example, if the sources depend on the
value of a variable, or you if you want the task to rerun if some arguments
change even if the source has not.

:::

::: tip

The method `none` skips any validation and always runs the task.

:::

::: info

For the `checksum` (default) or `timestamp` method to work, it is only necessary
to inform the source files. When the `timestamp` method is used, the last time
of the running the task is considered as a generate.

:::

### Using programmatic checks to indicate a task is up to date

Alternatively, you can inform a sequence of tests as `status`. If no error is
returned (exit status 0), the task is considered up-to-date:

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

Normally, you would use `sources` in combination with `generates` - but for
tasks that generate remote artifacts (Docker images, deploys, CD releases) the
checksum source and timestamps require either access to the artifact or for an
out-of-band refresh of the `.checksum` fingerprint file.

Two special variables <span v-pre>`{{.CHECKSUM}}`</span> and
<span v-pre>`{{.TIMESTAMP}}`</span> are available for interpolation within
`cmds` and `status` commands, depending on the method assigned to fingerprint
the sources. Only `source` globs are fingerprinted.

Note that the <span v-pre>`{{.TIMESTAMP}}`</span> variable is a "live" Go
`time.Time` struct, and can be formatted using any of the methods that
`time.Time` responds to.

See [the Go Time documentation](https://golang.org/pkg/time/) for more
information.

You can use `--force` or `-f` if you want to force a task to run even when
up-to-date.

Also, `task --status [tasks]...` will exit with a non-zero
[exit code](/docs/reference/cli#exit-codes) if any of the tasks are not up-to-date.

`status` can be combined with the
[fingerprinting](#by-fingerprinting-locally-generated-files-and-their-sources)
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
      - grep -q '"dev"{{:}} false' ./vendor/composer/installed.json
```

### Using programmatic checks to cancel the execution of a task and its dependencies

In addition to `status` checks, `preconditions` checks are the logical inverse
of `status` checks. That is, if you need a certain set of conditions to be
_true_ you can use the `preconditions` stanza. `preconditions` are similar to
`status` lines, except they support `sh` expansion, and they SHOULD all
return 0.

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
      - sh: '[ 1 = 0 ]'
        msg: "One doesn't equal Zero, Halting"
```

Preconditions can set specific failure messages that can tell a user what steps
to take using the `msg` field.

If a task has a dependency on a sub-task with a precondition, and that
precondition is not met - the calling task will fail. Note that a task executed
with a failing precondition will not run unless `--force` is given.

Unlike `status`, which will skip a task if it is up to date and continue
executing tasks that depend on it, a `precondition` will fail a task, along with
any other tasks that depend on it.

```yaml
version: '3'

tasks:
  task-will-fail:
    preconditions:
      - sh: 'exit 1'

  task-will-also-fail:
    deps:
      - task-will-fail

  task-will-still-fail:
    cmds:
      - task: task-will-fail
      - echo "I will not run"
```

### Limiting when tasks run

If a task executed by multiple `cmds` or multiple `deps` you can control when it
is executed using `run`. `run` can also be set at the root of the Taskfile to
change the behavior of all the tasks unless explicitly overridden.

Supported values for `run`:

- `always` (default) always attempt to invoke the task regardless of the number
  of previous executions
- `once` only invoke this task once regardless of the number of references
- `when_changed` only invokes the task once for each unique set of variables
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

### Ensuring required variables are set

If you want to check that certain variables are set before running a task then
you can use `requires`. This is useful when might not be clear to users which
variables are needed, or if you want clear message about what is required. Also
some tasks could have dangerous side effects if run with un-set variables.

Using `requires` you specify an array of strings in the `vars` sub-section under
`requires`, these strings are variable names which are checked prior to running
the task. If any variables are un-set then the task will error and not run.

Environmental variables are also checked.

Syntax:

```yaml
requires:
  vars: [] # Array of strings
```

::: info

Variables set to empty zero length strings, will pass the `requires` check.

:::

Example of using `requires`:

```yaml
version: '3'

tasks:
  docker-build:
    cmds:
      - 'docker build . -t {{.IMAGE_NAME}}:{{.IMAGE_TAG}}'

    # Make sure these variables are set before running
    requires:
      vars: [IMAGE_NAME, IMAGE_TAG]
```

### Ensuring required variables have allowed values

If you want to ensure that a variable is set to one of a predefined set of valid
values before executing a task, you can use requires. This is particularly
useful when there are strict requirements for what values a variable can take,
and you want to provide clear feedback to the user when an invalid value is
detected.

To use `requires`, you specify an array of allowed values in the vars
sub-section under requires. Task will check if the variable is set to one of the
allowed values. If the variable does not match any of these values, the task
will raise an error and stop execution.

This check applies both to user-defined variables and environment variables.

Example of using `requires`:

```yaml
version: '3'

tasks:
  deploy:
    cmds:
      - echo "deploying to {{.ENV}}"

    requires:
      vars:
        - name: ENV
          enum: [dev, beta, prod]
```

If `ENV` is not one of 'dev', 'beta' or 'prod' an error will be raised.

::: info

This is supported only for string variables.

:::

## Variables

Task allows you to set variables using the `vars` keyword. The following
variable types are supported:

- `string`
- `bool`
- `int`
- `float`
- `array`
- `map`

::: info

Defining a map requires that you use a special `map` subkey (see example below).

:::

```yaml
version: 3

tasks:
  foo:
    vars:
      STRING: 'Hello, World!'
      BOOL: true
      INT: 42
      FLOAT: 3.14
      ARRAY: [1, 2, 3]
      MAP:
        map: { A: 1, B: 2, C: 3 }
    cmds:
      - 'echo {{.STRING}}' # Hello, World!
      - 'echo {{.BOOL}}' # true
      - 'echo {{.INT}}' # 42
      - 'echo {{.FLOAT}}' # 3.14
      - 'echo {{.ARRAY}}' # [1 2 3]
      - 'echo {{index .ARRAY 0}}' # 1
      - 'echo {{.MAP}}' # map[A:1 B:2 C:3]
      - 'echo {{.MAP.A}}' # 1
```

Variables can be set in many places in a Taskfile. When executing
[templates][templating-reference], Task will look for variables in the order
listed below (most important first):

- Variables declared in the task definition
- Variables given while calling a task from another (See
  [Calling another task](#calling-another-task) above)
- Variables of the [included Taskfile](#including-other-taskfiles) (when the
  task is included)
- Variables of the [inclusion of the Taskfile](#vars-of-included-taskfiles)
  (when the task is included)
- Global variables (those declared in the `vars:` option in the Taskfile)
- Environment variables

Example of sending parameters with environment variables:

```shell
$ TASK_VARIABLE=a-value task do-something
```

::: tip

A special variable `.TASK` is always available containing the task name.

:::

Since some shells do not support the above syntax to set environment variables
(Windows) tasks also accept a similar style when not at the beginning of the
command.

```shell
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

Example of a `default` value to be overridden from CLI:

```yaml
version: '3'

tasks:
  greet_user:
    desc: 'Greet the user with a name.'
    vars:
      USER_NAME: '{{.USER_NAME| default "DefaultUser"}}'
    cmds:
      - echo "Hello, {{.USER_NAME}}!"
```

```shell
$ task greet_user
task: [greet_user] echo "Hello, DefaultUser!"
Hello, DefaultUser!
$ task greet_user USER_NAME="Bob"
task: [greet_user] echo "Hello, Bob!"
Hello, Bob!
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

### Referencing other variables

Templating is great for referencing string values if you want to pass a value
from one task to another. However, the templating engine is only able to output
strings. If you want to pass something other than a string to another task then
you will need to use a reference (`ref`) instead.

::: code-group

```yaml [Templating Engine]
version: 3

tasks:
  foo:
    vars:
      FOO: [A, B, C] # <-- FOO is defined as an array
    cmds:
      - task: bar
        vars:
          FOO: '{{.FOO}}' # <-- FOO gets converted to a string when passed to bar
  bar:
    cmds:
      - 'echo {{index .FOO 0}}' # <-- FOO is a string so the task outputs '91' which is the ASCII code for '[' instead of the expected 'A'
```

```yaml [Reference]
version: 3

tasks:
  foo:
    vars:
      FOO: [A, B, C] # <-- FOO is defined as an array
    cmds:
      - task: bar
        vars:
          FOO:
            ref: .FOO # <-- FOO gets passed by reference to bar and maintains its type
  bar:
    cmds:
      - 'echo {{index .FOO 0}}' # <-- FOO is still a map so the task outputs 'A' as expected
```

:::

This also works the same way when calling `deps` and when defining a variable
and can be used in any combination:

```yaml
version: 3

tasks:
  foo:
    vars:
      FOO: [A, B, C] # <-- FOO is defined as an array
      BAR:
        ref: .FOO # <-- BAR is defined as a reference to FOO
    deps:
      - task: bar
        vars:
          BAR:
            ref: .BAR # <-- BAR gets passed by reference to bar and maintains its type
  bar:
    cmds:
      - 'echo {{index .BAR 0}}' # <-- BAR still refers to FOO so the task outputs 'A'
```

All references use the same templating syntax as regular templates, so in
addition to calling `.FOO`, you can also pass subkeys (`.FOO.BAR`) or indexes
(`index .FOO 0`) and use functions (`len .FOO`) as described in the
[templating-reference][templating-reference]:

```yaml
version: 3

tasks:
  foo:
    vars:
      FOO: [A, B, C] # <-- FOO is defined as an array
    cmds:
      - task: bar
        vars:
          FOO:
            ref: index .FOO 0 # <-- The element at index 0 is passed by reference to bar
  bar:
    cmds:
      - 'echo {{.FOO}}' # <-- FOO is just the letter 'A'
```

### Parsing JSON/YAML into map variables

If you have a raw JSON or YAML string that you want to process in Task, you can
use a combination of the `ref` keyword and the `fromJson` or `fromYaml`
templating functions to parse the string into a map variable. For example:

```yaml
version: '3'

tasks:
  task-with-map:
    vars:
      JSON: '{"a": 1, "b": 2, "c": 3}'
      FOO:
        ref: 'fromJson .JSON'
    cmds:
      - echo {{.FOO}}
```

```txt
map[a:1 b:2 c:3]
```

## Looping over values

Task allows you to loop over certain values and execute a command for each.
There are a number of ways to do this depending on the type of value you want to
loop over.

### Looping over a static list

The simplest kind of loop is an explicit one. This is useful when you want to
loop over a set of values that are known ahead of time.

```yaml
version: '3'

tasks:
  default:
    cmds:
      - for: ['foo.txt', 'bar.txt']
        cmd: cat {{ .ITEM }}
```

### Looping over a matrix

If you need to loop over all permutations of multiple lists, you can use the
`matrix` property. This should be familiar to anyone who has used a matrix in a
CI/CD pipeline.

```yaml
version: '3'

tasks:
  default:
    silent: true
    cmds:
      - for:
          matrix:
            OS: ['windows', 'linux', 'darwin']
            ARCH: ['amd64', 'arm64']
        cmd:
          echo "{{.ITEM.OS}}/{{.ITEM.ARCH}}"
```

This will output:

```txt
windows/amd64
windows/arm64
linux/amd64
linux/arm64
darwin/amd64
darwin/arm64
```

You can also use references to other variables as long as they are also lists:

```yaml
version: '3'

vars:
  OS_VAR: ['windows', 'linux', 'darwin']
  ARCH_VAR: ['amd64', 'arm64']

tasks:
  default:
    cmds:
      - for:
          matrix:
            OS:
              ref: .OS_VAR
            ARCH:
              ref: .ARCH_VAR
        cmd:
          echo "{{.ITEM.OS}}/{{.ITEM.ARCH}}"
```

### Looping over your task's sources or generated files

You are also able to loop over the sources of your task or the files it
generates:

::: code-group

```yaml [Sources]
version: '3'

tasks:
  default:
    sources:
      - foo.txt
      - bar.txt
    cmds:
      - for: sources
        cmd: cat {{ .ITEM }}
```

```yaml [Generates]
version: '3'

tasks:
  default:
    generates:
      - foo.txt
      - bar.txt
    cmds:
      - for: generates
        cmd: cat {{ .ITEM }}
```

:::

This will also work if you use globbing syntax in `sources` or `generates`. For
example, if you specify a source for `*.txt`, the loop will iterate over all
files that match that glob.

Paths will always be returned as paths relative to the task directory. If you
need to convert this to an absolute path, you can use the built-in `joinPath`
function. There are some
[special variables](/docs/reference/templating#special-variables) that you may find
useful for this.

::: code-group

```yaml [Sources]
version: '3'

tasks:
  default:
    vars:
      MY_DIR: /path/to/dir
    dir: '{{.MY_DIR}}'
    sources:
      - foo.txt
      - bar.txt
    cmds:
      - for: sources
        cmd: cat {{joinPath .MY_DIR .ITEM}}
```

```yaml [Generates]
version: '3'

tasks:
  default:
    vars:
      MY_DIR: /path/to/dir
    dir: '{{.MY_DIR}}'
    generates:
      - foo.txt
      - bar.txt
    cmds:
      - for: generates
        cmd: cat {{joinPath .MY_DIR .ITEM}}
```

:::

### Looping over variables

To loop over the contents of a variable, use the `var` key followed by the name
of the variable you want to loop over. By default, string variables will be
split on any whitespace characters.

```yaml
version: '3'

tasks:
  default:
    vars:
      MY_VAR: foo.txt bar.txt
    cmds:
      - for: { var: MY_VAR }
        cmd: cat {{.ITEM}}
```

If you need to split a string on a different character, you can do this by
specifying the `split` property:

```yaml
version: '3'

tasks:
  default:
    vars:
      MY_VAR: foo.txt,bar.txt
    cmds:
      - for: { var: MY_VAR, split: ',' }
        cmd: cat {{.ITEM}}
```

You can also loop over arrays and maps directly:

```yaml
version: 3

tasks:
  foo:
    vars:
      LIST: [foo, bar, baz]
    cmds:
      - for:
          var: LIST
        cmd: echo {{.ITEM}}
```

When looping over a map we also make an additional <span v-pre>`{{.KEY}}`</span>
variable available that holds the string value of the map key. Remember that
maps are unordered, so the order in which the items are looped over is random.

All of this also works with dynamic variables!

```yaml
version: '3'

tasks:
  default:
    vars:
      MY_VAR:
        sh: find -type f -name '*.txt'
    cmds:
      - for: { var: MY_VAR }
        cmd: cat {{.ITEM}}
```

### Renaming variables

If you want to rename the iterator variable to make it clearer what the value
contains, you can do so by specifying the `as` property:

```yaml
version: '3'

tasks:
  default:
    vars:
      MY_VAR: foo.txt bar.txt
    cmds:
      - for: { var: MY_VAR, as: FILE }
        cmd: cat {{.FILE}}
```

### Looping over tasks

Because the `for` property is defined at the `cmds` level, you can also use it
alongside the `task` keyword to run tasks multiple times with different
variables.

```yaml
version: '3'

tasks:
  default:
    cmds:
      - for: [foo, bar]
        task: my-task
        vars:
          FILE: '{{.ITEM}}'

  my-task:
    cmds:
      - echo '{{.FILE}}'
```

Or if you want to run different tasks depending on the value of the loop:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - for: [foo, bar]
        task: task-{{.ITEM}}

  task-foo:
    cmds:
      - echo 'foo'

  task-bar:
    cmds:
      - echo 'bar'
```

### Looping over dependencies

All of the above looping techniques can also be applied to the `deps` property.
This allows you to combine loops with concurrency:

```yaml
version: '3'

tasks:
  default:
    deps:
      - for: [foo, bar]
        task: my-task
        vars:
          FILE: '{{.ITEM}}'

  my-task:
    cmds:
      - echo '{{.FILE}}'
```

It is important to note that as `deps` are run in parallel, the order in which
the iterations are run is not guaranteed and the output may vary. For example,
the output of the above example may be either:

```shell
foo
bar
```

or

```shell
bar
foo
```

## Forwarding CLI arguments to commands

If `--` is given in the CLI, all following parameters are added to a special
`.CLI_ARGS` variable. This is useful to forward arguments to another command.

The below example will run `yarn install`.

```shell
$ task yarn -- install
```

```yaml
version: '3'

tasks:
  yarn:
    cmds:
      - yarn {{.CLI_ARGS}}
```

## Wildcard arguments

Another way to parse arguments into a task is to use a wildcard in your task's
name. Wildcards are denoted by an asterisk (`*`) and can be used multiple times
in a task's name to pass in multiple arguments.

Matching arguments will be captured and stored in the `.MATCH` variable and can
then be used in your task's commands like any other variable. This variable is
an array of strings and so will need to be indexed to access the individual
arguments. We suggest creating a named variable for each argument to make it
clear what they contain:

```yaml
version: '3'

tasks:
  start:*:*:
    vars:
      SERVICE: '{{index .MATCH 0}}'
      REPLICAS: '{{index .MATCH 1}}'
    cmds:
      - echo "Starting {{.SERVICE}} with {{.REPLICAS}} replicas"

  start:*:
    vars:
      SERVICE: '{{index .MATCH 0}}'
    cmds:
      - echo "Starting {{.SERVICE}}"
```

This call matches the `start:*` task and the string "foo" is captured by the
wildcard and stored in the `.MATCH` variable. We then index the `.MATCH` array
and store the result in the `.SERVICE` variable which is then echoed out in the
cmds:

```shell
$ task start:foo
Starting foo
```

You can use whitespace in your arguments as long as you quote the task name:

```shell
$ task "start:foo bar"
Starting foo bar
```

If multiple matching tasks are found, the first one listed in the Taskfile will
be used. If you are using included Taskfiles, tasks in parent files will be
considered first.

```shell
$ task start:foo:3
Starting foo with 3 replicas
```

## Doing task cleanup with `defer`

With the `defer` keyword, it's possible to schedule cleanup to be run once the
task finishes. The difference with just putting it as the last command is that
this command will run even when the task fails.

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

::: info

Due to the nature of how the
[Go's own `defer` work](https://go.dev/tour/flowcontrol/13), the deferred
commands are executed in the reverse order if you schedule multiple of them.

:::

A special variable `.EXIT_CODE` is exposed when a command exited with a non-zero
[exit code](/docs/reference/cli#exit-codes). You can check its presence to know if
the task completed successfully or not:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - defer:
          echo '{{if .EXIT_CODE}}Failed with {{.EXIT_CODE}}!{{else}}Success!{{end}}'
      - exit 1
```

## Help

Running `task --list` (or `task -l`) lists all tasks with a description. The
following Taskfile:

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
      - esbuild --bundle --minify js/index.js > public/bundle.js

  css:
    cmds:
      - esbuild --bundle --minify css/index.css > public/bundle.css
```

would print the following output:

```shell
* build:   Build the go binary.
* test:    Run all the go tests.
```

If you want to see all tasks, there's a `--list-all` (alias `-a`) flag as well.

## Display summary of task

Running `task --summary task-name` will show a summary of a task. The following
Taskfile:

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

with running `task --summary release` would print the following output:

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

If a summary is missing, the description will be printed. If the task does not
have a summary or a description, a warning is printed.

Please note: _showing the summary will not execute the command_.

## Task aliases

Aliases are alternative names for tasks. They can be used to make it easier and
quicker to run tasks with long or hard-to-type names. You can use them on the
command line, when [calling sub-tasks](#calling-another-task) in your Taskfile
and when [including tasks](#including-other-taskfiles) with aliases from another
Taskfile. They can also be used together with
[namespace aliases](#namespace-aliases).

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

Sometimes you may want to override the task name printed on the summary,
up-to-date messages to STDOUT, etc. In this case, you can just set `label:`,
which can also be interpolated with variables:

```yaml
version: '3'

tasks:
  default:
    cmds:
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

## Warning Prompts

Warning Prompts are used to prompt a user for confirmation before a task is
executed.

Below is an example using `prompt` with a dangerous command, that is called
between two safe commands:

```yaml
version: '3'

tasks:
  example:
    cmds:
      - task: not-dangerous
      - task: dangerous
      - task: another-not-dangerous

  not-dangerous:
    cmds:
      - echo 'not dangerous command'

  another-not-dangerous:
    cmds:
      - echo 'another not dangerous command'

  dangerous:
    prompt: This is a dangerous command... Do you want to continue?
    cmds:
      - echo 'dangerous command'
```

```shell
❯ task dangerous
task: "This is a dangerous command... Do you want to continue?" [y/N]
```

Prompts can be a single value or a list of prompts, like below:

```yaml
version: '3'

tasks:
  example:
    cmds:
      - task: dangerous

  dangerous:
    prompt:
      - This is a dangerous command... Do you want to continue?
      - Are you sure?
    cmds:
      - echo 'dangerous command'
```

Warning prompts are called before executing a task. If a prompt is denied Task
will exit with [exit code](/docs/reference/cli#exit-codes) 205. If approved, Task
will continue as normal.

```shell
❯ task example
not dangerous command
task: "This is a dangerous command. Do you want to continue?" [y/N]
y
dangerous command
another not dangerous command
```

To skip warning prompts automatically, you can use the `--yes` (alias `-y`)
option when calling the task. By including this option, all warnings, will be
automatically confirmed, and no prompts will be shown.

::: warning

Tasks with prompts always fail by default on non-terminal environments, like a
CI, where an `stdin` won't be available for the user to answer. In those cases,
use `--yes` (`-y`) to force all tasks with a prompt to run.

:::

## Silent mode

Silent mode disables the echoing of commands before Task runs it. For the
following Taskfile:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "Print something"
```

Normally this will be printed:

```shell
echo "Print something"
Print something
```

With silent mode on, the below will be printed instead:

```shell
Print something
```

There are four ways to enable silent mode:

- At command level:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - cmd: echo "Print something"
        silent: true
```

- At task level:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "Print something"
    silent: true
```

- Globally at Taskfile level:

```yaml
version: '3'

silent: true

tasks:
  echo:
    cmds:
      - echo "Print something"
```

- Or globally with `--silent` or `-s` flag

If you want to suppress STDOUT instead, just redirect a command to `/dev/null`:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - echo "This will print nothing" > /dev/null
```

## Dry run mode

Dry run mode (`--dry`) compiles and steps through each task, printing the
commands that would be run without executing them. This is useful for debugging
your Taskfiles.

## Ignore errors

You have the option to ignore errors during command execution. Given the
following Taskfile:

```yaml
version: '3'

tasks:
  echo:
    cmds:
      - exit 1
      - echo "Hello World"
```

Task will abort the execution after running `exit 1` because the status code `1`
stands for `EXIT_FAILURE`. However, it is possible to continue with execution
using `ignore_error`:

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
for all commands. Nevertheless, keep in mind that this option will not propagate
to other tasks called either by `deps` or `cmds`!

## Output syntax

By default, Task just redirects the STDOUT and STDERR of the running commands to
the shell in real-time. This is good for having live feedback for logging
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

When using the `group` output, you can optionally provide a templated message to
print at the start and end of the group. This can be useful for instructing CI
systems to group all of the output for a given task, such as with
[GitHub Actions' `::group::` command](https://docs.github.com/en/actions/learn-github-actions/workflow-commands-for-github-actions#grouping-log-lines)
or
[Azure Pipelines](https://docs.microsoft.com/en-us/azure/devops/pipelines/scripts/logging-commands?expand=1&view=azure-devops&tabs=bash#formatting-commands).

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

```shell
$ task default
::group::default
Hello, World!
::endgroup::
```

When using the `group` output, you may swallow the output of the executed
command on standard output and standard error if it does not fail (zero exit
code).

```yaml
version: '3'

silent: true

output:
  group:
    error_only: true

tasks:
  passes: echo 'output-of-passes'
  errors: echo 'output-of-errors' && exit 1
```

```shell
$ task passes
$ task errors
output-of-errors
task: Failed to run task "errors": exit status 1
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
        vars: { TEXT: foo }
      - task: print
        vars: { TEXT: bar }
      - task: print
        vars: { TEXT: baz }

  print:
    cmds:
      - echo "{{.TEXT}}"
    prefix: 'print-{{.TEXT}}'
    silent: true
```

```shell
$ task default
[print-foo] foo
[print-bar] bar
[print-baz] baz
```

::: tip

The `output` option can also be specified by the `--output` or `-o` flags.

:::

## CI Integration

### Colored output

Task automatically enables colored output when running in CI environments
(`CI=true`). Most CI providers set this variable automatically.

You can also force colored output with `FORCE_COLOR=1` or disable it with
`NO_COLOR=1`.

### Error annotations

When running in GitHub Actions (`GITHUB_ACTIONS=true`), Task automatically emits
error annotations when a task fails. These annotations appear in the workflow
summary, making it easier to spot failures without scrolling through logs.

```shell
::error title=Task 'build' failed::exit status 1
```

This feature requires no configuration and works automatically.

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

Starting on Task v3, you can now write tasks with a shorter syntax if they have
the default settings (e.g. no custom `env:`, `vars:`, `desc:`, `silent:` , etc):

```yaml
version: '3'

tasks:
  build: go build -v -o ./app{{exeExt}} .

  run:
    - task: build
    - ./app{{exeExt}} -h localhost -p 8080
```

## `set` and `shopt`

It's possible to specify options to the
[`set`](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html)
and
[`shopt`](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html)
builtins. This can be added at global, task or command level.

```yaml
version: '3'

set: [pipefail]
shopt: [globstar]

tasks:
  # `globstar` required for double star globs to work
  default: echo **/*.go
```

::: info

Keep in mind that not all options are available in the
[shell interpreter library](https://github.com/mvdan/sh) that Task uses.

:::

## Watch tasks

With the flags `--watch` or `-w` task will watch for file changes and run the
task again. This requires the `sources` attribute to be given, so task knows
which files to watch.

The default watch interval is 100 milliseconds, but it's possible to change it
by either setting `interval: '500ms'` in the root of the Taskfile or by passing
it as an argument like `--interval=500ms`. This interval is the time Task will
wait for duplicated events. It will only run the task again once, even if
multiple changes happen within the interval.

Also, it's possible to set `watch: true` in a given task and it'll automatically
run in watch mode:

```yaml
version: '3'

interval: 500ms

tasks:
  build:
    desc: Builds the Go application
    watch: true
    sources:
      - '**/*.go'
    cmds:
      - go build # ...
```

::: info

Note that when setting `watch: true` to a task, it'll only run in watch mode
when running from the CLI via `task my-watch-task`, but won't run in watch mode
if called by another task, either directly or as a dependency.

:::

::: warning

The watcher can misbehave in certain scenarios, in particular for long-running
servers. There is a [known bug](https://github.com/go-task/task/issues/160)
where child processes of the running might not be killed appropriately. It's
advised to avoid running commands as `go run` and prefer `go build [...] &&
./binary` instead.

If you are having issues, you might want to try tools specifically designed for
live-reloading, like [Air](https://github.com/air-verse/air/). Also, be sure to
[report any issues](https://github.com/go-task/task/issues/new?template=bug_report.yml)
to us.

:::

[config]: /docs/reference/config
[gotemplate]: https://golang.org/pkg/text/template/
[templating-reference]: /docs/reference/templating
