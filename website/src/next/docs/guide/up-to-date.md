---
title: Up-to-date checks
description:
  Stop a task from running again when nothing has changed, using source and
  generated file fingerprints or your own `status` checks.
section: Guide
docType: guide
outline: deep
---

# Up-to-date checks {#skipping-work-that-is-up-to-date}

Task can skip a task entirely when its work is already done. There are two
mechanisms: let Task compare files for you, or tell it yourself.

## File fingerprints {#by-fingerprinting-locally-generated-files-and-their-sources}

Declare the input files in `sources` and the output files in `generates`:

```yaml
version: '3'

tasks:
  build:
    sources: [main.go]
    generates: ['app{{exeExt}}']
    cmds:
      - go build -o app{{exeExt}} main.go
```

Run `task build` once to create the executable. Run it again without changing
`main.go`, and Task reports that `build` is up to date. Editing the source or
deleting the output makes Task run the build again.

`sources` and `generates` accept individual files or glob patterns. By default,
Task compares source checksums. For checks that do not depend on local files,
use
[status checks](#using-programmatic-checks-to-indicate-a-task-is-up-to-date).

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

::: tip

If your globs match files that are ignored by Git (build artifacts, caches,
etc.), you can set `use_gitignore: true` at the root of your Taskfile to exclude
anything matched by `.gitignore` rules from `sources` and `generates`
resolution. The setting can also be enabled or disabled per task, which takes
precedence over the root value.

:::

## Status checks {#using-programmatic-checks-to-indicate-a-task-is-up-to-date}

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
[exit code](../reference/cli.md#exit-codes) if any of the tasks are not
up-to-date.

`status` can be combined with the
[fingerprinting](#by-fingerprinting-locally-generated-files-and-their-sources)
to have a task run if either the source/generated artifacts changes, or the
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
