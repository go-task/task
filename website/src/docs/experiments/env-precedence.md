---
title: 'Env Precedence (#1038)'
description:
  Experiment to change the precedence of environment variables in Task
outline: deep
---

# Env Precedence (#1038)

::: warning

All experimental features are subject to breaking changes and/or removal _at any
time_. We strongly recommend that you do not use these features in a production
environment. They are intended for testing and feedback only.

:::

::: danger

This experiment breaks the following functionality:

- environment variable will take precedence over OS environment variables

:::

::: info

To enable this experiment, set the environment variable:
`TASK_X_ENV_PRECEDENCE=1`. Check out
[our guide to enabling experiments](./index.md#enabling-experiments) for more
information.

:::

Before this experiment, the OS variable took precedence over the task
environment variable. This experiment changes the precedence to make the task
environment variable take precedence over the OS variable.

Consider the following example:

```yml
version: '3'

tasks:
  default:
    env:
      KEY: 'other'
    cmds:
      - echo "$KEY"
```

Running `KEY=some task` before this experiment, the output would be `some`, but
after this experiment, the output would be `other`.

If you still want to get the OS variable, you can use the template function env
like follow : <span v-pre>`{{env "OS_VAR"}}`</span>.

```yml
version: '3'

tasks:
  default:
    env:
      KEY: 'other'
    cmds:
      - echo "$KEY"
      - echo {{env "KEY"}}
```

Running `KEY=some task`, the output would be `other` and `some`.

Like other variables/envs, you can also fall back to a given value using the
default template function:

```yml
MY_ENV: '{{.MY_ENV | default "fallback"}}'
```
