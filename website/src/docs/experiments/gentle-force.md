---
title: 'Gentle Force (#1200)'
description: Experiment to modify the behavior of the --force flag in Task
outline: deep
---

# Gentle Force (#1200)

::: warning

All experimental features are subject to breaking changes and/or removal _at any
time_. We strongly recommend that you do not use these features in a production
environment. They are intended for testing and feedback only.

:::

::: danger

This experiment breaks the following functionality:

- The `--force` flag

:::

::: info

To enable this experiment, set the environment variable:
`TASK_X_GENTLE_FORCE=1`. Check out
[our guide to enabling experiments](./index.md#enabling-experiments) for more
information.

:::

The `--force` flag currently forces _all_ tasks to run regardless of the status
checks. This can be useful, but we have found that most of the time users only
expect the direct task they are calling to be forced and _not_ all of its
dependant tasks.

This experiment changes the `--force` flag to only force the directly called
task. All dependant tasks will have their statuses checked as normal and will
only run if Task considers them to be out of date. A new `--force-all` flag will
also be added to maintain the current behavior for users that need this
functionality.

If you want to migrate, but continue to force all dependant tasks to run, you
should replace all uses of the `--force` flag with `--force-all`. Alternatively,
if you want to adopt the new behavior, you can continue to use the `--force`
flag as you do now!
