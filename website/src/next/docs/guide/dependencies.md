---
title: Dependencies and task calls
description:
  Run checks before a build, put tasks in order, pass variables, and control
  repeated calls and cleanup.
section: Guide
docType: guide
outline: deep
---

# Dependencies and task calls

A build often needs several steps: lint the code, run tests, then compile. Use
`deps` when the checks can run in parallel, or task calls in `cmds` when each
step must finish before the next starts.

## Run dependencies first {#task-dependencies}

In a Go project, this Taskfile runs lint and tests before building:

```yaml
version: '3'

tasks:
  build:
    deps: [lint, test]
    cmds:
      - go build ./...

  lint:
    cmds:
      - go vet ./...

  test:
    cmds:
      - go test ./...
```

Run `task build`. Task starts `lint` and `test` concurrently and waits for both
to succeed before running `go build`. If either fails, the build command does
not run.

The order of entries in `deps` does not set their execution order. Use
[task calls in `cmds`](#calling-another-task) if tests must wait for lint to
finish.

You can also group tasks without adding commands. Add this task to the same
Taskfile to run only the checks with `task check`:

```yaml
tasks:
  check:
    deps: [lint, test]
```

## Run task calls in order {#calling-another-task}

To run lint, then tests, then the build, replace `build` in the example above
with:

```yaml
tasks:
  build:
    cmds:
      - task: lint
      - task: test
      - go build ./...
```

Task waits for each command or task call to succeed before starting the next. If
lint fails, neither tests nor the build run.

Each called task still runs its own dependencies first. You can combine the two
forms: use `deps` for independent prerequisites and `cmds` for the steps that
need an order.

::: tip Calling tasks from another Taskfile

To call a task declared in the root Taskfile from an
[included Taskfile](./includes.md), add a leading `:`, for example
`task: :build`.

:::

## Pass variables to a task

Use the object form of a task call to provide `vars`. You can also set
`silent: true` to hide the command being executed for that call:

```yaml
version: '3'

tasks:
  greet:
    cmds:
      - echo "Hello, {{.RECIPIENT}}!"

  greet:all:
    cmds:
      - task: greet
        vars: { RECIPIENT: 'World' }
      - task: greet
        vars: { RECIPIENT: 'Cruel World' }
        silent: true
```

`task greet:all` prints both greetings in order. The second call still prints
its greeting: [silent mode](./output.md#silent-mode) hides the command, not its
output.

The same `task`, `vars`, and `silent` fields work in `deps`. Use
[references](./variables.md#referencing-other-variables) to pass arrays and maps
without turning them into strings. To call a task for each value in a list, see
[Looping over tasks](./loops.md#looping-over-tasks).

## Control parallel work

Dependencies run concurrently by default. To run tasks from the command line in
parallel too, use `task --parallel lint test` (or `task -p lint test`).

### Limit concurrent tasks {#limiting-how-much-runs-at-once}

Use `--concurrency` (or `-C`) when tasks compete for resources such as database
connections or memory:

```shell
task --concurrency 2 build
```

The default is `0`, meaning no limit. A limit of `1` restricts concurrency but
does not establish an order between dependencies; use task calls in `cmds` when
order matters.

### Cancel on failure {#fail-fast-dependencies}

By default, Task lets the other dependencies finish if one fails. In either
case, the parent task's commands do not run after a dependency failure.

Set `failfast: true` on a task to cancel its remaining dependency work when one
dependency fails. For the build example:

```yaml
tasks:
  build:
    deps: [lint, test]
    failfast: true
    cmds:
      - go build ./...
```

Already-running commands receive cancellation; this does not undo work they have
completed. You can also set `failfast: true` in [`.taskrc.yml`][config] or use
`task --failfast build`. The CLI flag also applies to tasks started with
`--parallel`.

### Read parallel output {#interleaved-output-is-expected}

Concurrent tasks can interleave their output, and the order may change between
runs. Set `output: prefixed` to label each line with its task, or
`output: group` to print each command's buffered output in one block when that
command finishes. A group is per command, so blocks from different tasks can
still alternate. See [Output and logging](./output.md) for examples.

## Control repeated calls {#repeated-calls}

A shared dependency can be reached through several tasks. By default, Task
attempts to run it for each call. Use `run` to control repeated calls within one
invocation:

| Value              | Behavior                                          |
| ------------------ | ------------------------------------------------- |
| `always` (default) | Attempt to run the task on every call.            |
| `once`             | Run the task once, even if it is called again.    |
| `when_changed`     | Run once for each distinct set of call variables. |

Set `run` on a task, or at the root of the Taskfile to provide a default for all
tasks. A task's own setting takes precedence.

### Run shared setup once {#running-a-task-only-once}

If lint and tests both need setup, mark the setup task with `run: once`:

```yaml
version: '3'

tasks:
  check:
    deps: [lint, test]

  lint:
    deps: [setup]
    cmds:
      - echo "Linting"

  test:
    deps: [setup]
    cmds:
      - echo "Testing"

  setup:
    run: once
    cmds:
      - echo "Setting up"
```

`task check` prints `Setting up` once. Both lint and tests wait for that setup
to finish. Without `run: once`, setup runs for each task that depends on it.

### Run once per input {#once-per-set-of-variables}

Use `run: when_changed` when a task may be called several times with the same
inputs:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - task: generate
        vars: { CONTENT: '1' }
      - task: generate
        vars: { CONTENT: '2' }
      - task: generate
        vars: { CONTENT: '2' }

  generate:
    run: when_changed
    cmds:
      - echo {{.CONTENT}}
```

This prints `1` and `2`. The third call repeats the second call's variables and
is skipped.

The `run` settings do not remember earlier invocations of Task. To skip work
when files or external artifacts have not changed, use
[up-to-date checks](./up-to-date.md).

## Clean up after a task {#doing-task-cleanup-with-defer}

Use `defer` when a task acquires a resource that needs to be released. For
example, this task starts Docker Compose services, runs integration tests, and
stops the services even if the tests fail:

```yaml
version: '3'

tasks:
  integration:
    cmds:
      - docker compose up -d
      - defer: docker compose down
      - go test -tags=integration ./...
```

Cleanup is registered when Task reaches the `defer` entry. Dependencies run
before `cmds`, so a failed dependency prevents that registration. Keep the
resource setup, `defer`, and the work that uses the resource in the command
sequence, as above.

For multiple cleanup steps, cleanup tasks, and inspecting failures with
`.EXIT_CODE`, see
[Errors and cleanup](./errors-and-cleanup.md#cleanup-with-defer).

[config]: ../reference/config.md
