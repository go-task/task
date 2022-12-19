---
slug: /taskfile-versions/
sidebar_position: 11
---

# Taskfile Versions

The Taskfile syntax and features changed with time. This document explains what
changed on each version and how to upgrade your Taskfile.

## What the Taskfile version mean

The Taskfile version follows the Task version. E.g. the change to Taskfile
version `2` means that Task `v2.0.0` should be release to support it.

The `version:` key on Taskfile accepts a semver string, so either `2`, `2.0` or
`2.0.0` is accepted. If you choose to use `2.0` Task will not enable future
`2.1` features, but if you choose to use `2`, then any `2.x.x` features will be
available, but not `3.0.0+`.

## Version 1

> NOTE: Taskfiles in version 1 are not supported on Task >= v3.0.0 anymore.

In the first version of the `Taskfile`, the `version:` key was not available,
because the tasks was in the root of the YAML document. Like this:

```yaml
echo:
  cmds:
    - echo "Hello, World!"
```

The variable priority order was also different:

1. Call variables
2. Environment
3. Task variables
4. `Taskvars.yml` variables

## Version 2.0

At version 2, we introduced the `version:` key, to allow us to evolve Task
with new features without breaking existing Taskfiles. The new syntax is as
follows:

```yaml
version: '2'

tasks:
  echo:
    cmds:
      - echo "Hello, World!"
```

Version 2 allows you to write global variables directly in the Taskfile,
if you don't want to create a `Taskvars.yml`:

```yaml
version: '2'

vars:
  GREETING: Hello, World!

tasks:
  greet:
    cmds:
      - echo "{{.GREETING}}"
```

The variable priority order changed to the following:

1. Task variables
2. Call variables
3. Taskfile variables
4. Taskvars file variables
5. Environment variables

A new global option was added to configure the number of variables expansions
(which default to 2):

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

## Version 2.1

Version 2.1 includes a global `output` option, to allow having more control
over how commands output are printed to the console
(see [documentation][output] for more info):

```yaml
version: '2'

output: prefixed

tasks:
  server:
    cmds:
      - go run main.go
  prefix: server
```

From this version it's also possible to ignore errors of a command or task
(check documentation [here][ignore_errors]):

```yaml
version: '2'

tasks:
  example-1:
    cmds:
      - cmd: exit 1
        ignore_error: true
      - echo "This will be print"

  example-2:
    cmds:
      - exit 1
      - echo "This will be print"
    ignore_error: true
```

## Version 2.2

Version 2.2 comes with a global `includes` options to include other
Taskfiles:

```yaml
version: '2'

includes:
  docs: ./documentation # will look for ./documentation/Taskfile.yml
  docker: ./DockerTasks.yml
```

## Version 2.6

Version 2.6 comes with `preconditions` stanza in tasks.

```yaml
version: '2'

tasks:
  upload_environment:
    preconditions:
      - test -f .env
    cmds:
      - aws s3 cp .env s3://myenvironment
```

Please check the [documentation][includes]

[output]: usage.md#output-syntax
[ignore_errors]: usage.md#ignore-errors
[includes]: usage.md#including-other-taskfiles

## Version 3

These are some major changes done on `v3`:

- Task's output will now be colored
- Added support for `.env` like files
- Added `label:` setting to task so one can override how the task name
  appear in the logs
- A global `method:` was added to allow setting the default method,
  and Task's default changed to `checksum`
- Two magic variables were added when using `status:`: `CHECKSUM` and
  `TIMESTAMP` which contains, respectively, the md5 checksum and greatest
  modification timestamp of the files listed on `sources:`
- Also, the `TASK` variable is always available with the current task name
- CLI variables are always treated as global variables
- Added `dir:` option to `includes` to allow choosing on which directory an
  included Taskfile will run:

```yaml
includes:
  docs:
    taskfile: ./docs
    dir: ./docs
```

- Implemented short task syntax. All below syntaxes are equivalent:

```yaml
version: '3'

tasks:
  print:
    cmds:
      - echo "Hello, World!"
```

```yaml
version: '3'

tasks:
  print:
    - echo "Hello, World!"
```

```yaml
version: '3'

tasks:
  print: echo "Hello, World!"
```

- There was a major refactor on how variables are handled. They're now easier
  to understand. The `expansions:` setting was removed as it became unncessary.
  This is the order in which Task will process variables, each level can see
  the variables set by the previous one and override those.
  - Environment variables
  - Global + CLI variables
  - Call variables
  - Task variables
