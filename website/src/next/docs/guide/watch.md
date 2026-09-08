---
title: Watch mode
description: Re-run a task automatically whenever its sources change.
section: Guide
docType: guide
outline: deep
---

# Watch mode

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
advised to avoid running commands as `go run` and prefer
`go build [...] && ./binary` instead.

If you are having issues, you might want to try tools specifically designed for
live-reloading, like [Air](https://github.com/air-verse/air/). Also, be sure to
[report any issues](https://github.com/go-task/task/issues/new?template=bug_report.yml)
to us.

:::
