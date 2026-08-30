---
title: Dependencies and concurrency
description:
  What runs in parallel, what runs in order, and why the output of a Taskfile is
  not always in the order you wrote it.
outline: deep
---

# Dependencies and concurrency

A task can pull in other tasks two ways, and they behave differently. Choosing
the wrong one is the most common cause of a Taskfile that works on one machine
and not another.

## `deps` run together, `cmds` run in order

Everything in `deps` starts at once. Task waits for all of them to finish, then
runs `cmds`:

```yaml
version: '3'

tasks:
  build:
    deps: [compile, generate-assets]
    cmds:
      - echo "packaging"
```

`compile` and `generate-assets` run concurrently in an unspecified order, and
`packaging` is printed only once both have finished. Nothing orders the
dependencies relative to each other — if `generate-assets` needs `compile` to
have run, it must say so itself, with its own `deps`.

A task reference inside `cmds` is different: it runs at its position in the
list, and the next command waits for it.

```yaml
version: '3'

tasks:
  release:
    cmds:
      - task: build
      - task: publish
```

Here `build` finishes before `publish` starts.

**The rule of thumb:** `deps` expresses "these must have happened", `cmds`
expresses "do this, then this". If order matters, it belongs in `cmds`.

## Interleaved output is expected

Because dependencies run concurrently, their output arrives interleaved and in a
different order between runs. That is not a bug, and it is why the default
output mode can look scrambled on a parallel build.

Set `output: prefixed` to label each line with the task it came from, or
`output: group` to hold each task's output and print it in one block when it
finishes. See [Output and logging](../guide/output.md).

## Limiting how much runs at once

`--concurrency` / `-C` caps how many tasks run simultaneously. The default is
`0`, meaning no limit. It is the setting to reach for when parallel tasks
compete for the same resource — a database, a port, the network.

## When one dependency fails

By default Task waits for the other dependencies to finish before reporting the
failure. `--failfast` / `-F` stops everything as soon as one of them fails.

## Running a task only once

A task marked `run: once` executes a single time per invocation of `task`, no
matter how many other tasks depend on it:

```yaml
version: '3'

tasks:
  setup:
    run: once
    cmds:
      - echo "setting up"

  test:
    deps: [setup]
  lint:
    deps: [setup]

  check:
    deps: [test, lint]
```

`task check` prints `setting up` once, not twice. Without `run: once`, a shared
dependency runs for each dependent that asks for it.

## Cleanup runs in reverse

`defer` schedules a command to run when the task ends, whether it succeeded or
failed. Deferred commands run in reverse order of declaration, so the first
thing you set up is the last thing torn down:

```yaml
version: '3'

tasks:
  deploy:
    cmds:
      - defer: echo "stop the tunnel"
      - defer: echo "remove the temp dir"
      - echo "deploying"
```

That prints `deploying`, then `remove the temp dir`, then `stop the tunnel`.

## Related

- [Dependencies and task calls](../guide/dependencies.md) — the syntax for each.
- [Output and logging](../guide/output.md) — output modes for parallel runs.
- [CLI](../reference/cli.md) — `--concurrency`, `--failfast`.
