---
title: Dependencies and task calls
description:
  Run tasks in parallel with `deps`, call another task from `cmds`, and schedule
  cleanup with `defer`.
outline: deep
---

# Dependencies and task calls

A task can pull in other tasks in three ways, and each has different ordering
guarantees.

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
toggle [silent mode](./output.md#silent-mode) on a call-by-call basis:

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
[included Taskfile](./includes.md), add a leading `:` like this:
`task: :task-name`.

:::

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
[exit code](../reference/cli.md#exit-codes). You can check its presence to know
if the task completed successfully or not:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - defer:
          echo '{{if .EXIT_CODE}}Failed with
          {{.EXIT_CODE}}!{{else}}Success!{{end}}'
      - exit 1
```

[config]: ../reference/config.md
