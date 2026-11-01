---
title: Errors and cleanup
description:
  Continue after selected command failures with ignore_error, schedule cleanup
  with defer, and inspect a failed command's exit code.
section: Guide
docType: guide
outline: deep
---

# Errors and cleanup

By default, a failed command stops its task. Use `ignore_error` when a failure
can be tolerated, and `defer` for cleanup that must run after work finishes. For
concurrent dependencies, see
[fail-fast behavior](./dependencies.md#fail-fast-dependencies).

## Clean up after a task {#cleanup-with-defer}

Schedule a cleanup command with `defer`. Once Task reaches that entry in `cmds`,
it registers the command to run when the task finishes, including if a later
command fails. Deferred commands run in reverse order of registration. A `defer`
that Task never reaches is not registered, for example when an earlier command
or a dependency fails.

In this example, Task removes `tmpdir/` after the work command finishes:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - mkdir -p tmpdir/
      - defer: rm -rf tmpdir/
      - echo 'Work completed' > tmpdir/result.txt
```

Run `task`, then check that `tmpdir/` was removed. Replace the last command with
`exit 1` to try the failure path: cleanup still runs and Task reports the
failure. The directory name here is for the example; choose a task-specific path
for real work.

### Reuse a cleanup task

Use a deferred task call when several tasks share the same cleanup:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - mkdir -p tmpdir/
      - defer: { task: cleanup }
      - echo 'Work completed' > tmpdir/result.txt

  cleanup: rm -rf tmpdir/
```

### Inspect a failed command

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

## Continue after an error {#ignoring-command-errors}

Set `ignore_error: true` on a command to continue after it fails:

```yaml
version: '3'

tasks:
  example:
    cmds:
      - cmd: exit 1
        ignore_error: true
      - echo "Continuing after the failed command"
```

Without `ignore_error`, the `echo` command would not run.

You can also set `ignore_error` on a task to apply it to that task's commands.
The setting does not propagate to other tasks called through `cmds` or `deps`.
