---
title: Watch mode
description: Re-run a task automatically whenever its sources change.
section: Guide
docType: guide
outline: deep
---

# Watch mode

Use `--watch` (or `-w`) to keep a task running and rerun it when its source
files change. For a Go project:

```yaml
version: '3'

tasks:
  build:
    sources:
      - '**/*.go'
      - go.mod
      - go.sum
    cmds:
      - go build ./...
```

Run `task --watch build`, edit a Go file, and save it to trigger another build.
Press Ctrl+C to stop watching.

Watch mode detects changes; [up-to-date checks](./up-to-date.md) decide whether
the task's commands need to run. Both use the same `sources`.

## Enable watching by default

Set `watch: true` on a task if its normal CLI use should start a watcher:

```yaml
version: '3'

tasks:
  build:
    watch: true
    sources: ['**/*.go']
    cmds:
      - go build ./...
```

Now `task build` starts watch mode. The `watch` setting applies when the task is
called from the CLI; calls from another task through `cmds` or `deps` do not
start a watcher.

## Combine rapid changes

The default watch interval is 100 milliseconds. Increase it if saving several
files produces duplicate events:

```shell
task --watch --interval=500ms build
```

You can also set `interval: '500ms'` at the root of the Taskfile. The interval
groups events close together so that Task reruns the task once for the batch.

## Watch long-running apps

Builds and checks finish after each run. Servers keep running and need to stop
before a replacement can take over their port.

::: warning

There is a [known issue](https://github.com/go-task/task/issues/160) where watch
mode may leave child processes running. Prefer building and running the
executable directly over `go run`, which starts another process.

:::

For applications that need dedicated live-reload behavior, consider tools such
as [Air](https://github.com/air-verse/air/). Report Task-specific problems with
a small reproducer in the
[issue tracker](https://github.com/go-task/task/issues/new?template=bug_report.yml).
