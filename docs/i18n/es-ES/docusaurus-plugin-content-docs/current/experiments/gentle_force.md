---
slug: /experiments/gentle-force/
---

# Gentle Force

- Issue: [#1200][gentle-force-experiment]
- Environment variable: `TASK_X_FORCE=1`
- Breaks:
  - `--force` flag

The `--force` flag currently forces _all_ tasks to run regardless of the status checks. This can be useful, but we have found that most of the time users only expect the direct task they are calling to be forced and _not_ all of its dependant tasks.

This experiment changes the `--force` flag to only force the directly called task. All dependant tasks will have their statuses checked as normal and will only run if Task considers them to be out of date. A new `--force-all` flag will also be added to maintain the current behavior for users that need this functionality.

If you want to migrate, but continue to force all dependant tasks to run, you should replace all uses of the `--force` flag with `--force-all`. Alternatively, if you want to adopt the new behavior, you can continue to use the `--force` flag as you do now!

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[gentle-force-experiment]: https://github.com/go-task/task/issues/1200
