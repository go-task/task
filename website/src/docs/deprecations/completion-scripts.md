---
title: 'Completion Scripts'
description: Deprecation of direct completion scripts in Task’s Git directory
outline: deep
---

# Completion Scripts

::: danger

This deprecation breaks the following functionality:

- Any direct references to the completion scripts in the Task git repository

:::

Direct use of the completion scripts in the `completion/*` directory of the
[github.com/go-task/task][task] Git repository is deprecated. Any shell
configuration that directly refers to these scripts will potentially break in
the future as the scripts may be moved or deleted entirely. Any configuration
should be updated to use the [new method for generating shell
completions][completions] instead.

[completions]: /docs/installation#setup-completions
[task]: https://github.com/go-task/task
