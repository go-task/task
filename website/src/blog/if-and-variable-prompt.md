---
title: New `if:` Control and Variable Prompt
description: Introduction of the `if:` control and required variable prompts.
author: vmaerten
date: 2026-01-24
outline: deep
---

# New `if:` control and interactivity support

<AuthorCard :author="$frontmatter.author" />

The [v3.47.0][release] release is here, and it brings two exciting new features
to Task. Let's take a closer look at them!

## The New `if:` Control

This first feature is simply the second most upvoted issue of all time (!) with
58 :thumbsup:s (!!) at the time of writing.

It introduces the `if:` control, which allow you to conditionally skip the
execution of certain tasks and proceeding. `if:` can be set on a task-level or
command-level, and can be either a Bash command or a Go template expression.

Let me show a couple of examples.

Task-level with Bash expression:

```yaml
version: '3'

tasks:
  deploy:
    if: '[ "$CI" = "true" ]'
    cmds:
      - echo "Deploying..."
      - ./deploy.sh
```

Command-level with Go template expression:

```yaml
version: '3'

tasks:
  conditional:
    vars:
      ENABLE_FEATURE: "true"
    cmds:
      - cmd: echo "Feature is enabled"
        if: '{{eq .ENABLE_FEATURE "true"}}'
      - cmd: echo "Feature is disabled"
        if: '{{ne .ENABLE_FEATURE "true"}}'
```

For more details, please check out the [documentation][if-docs].
The [examples][if-examples] from the test suite may be useful too.

::: info

We had similar functionality before, but nothing that perfectly fits this use
case. There were [`sources:`][sources] and [`status:`][status], but those were
meant to check if a task was up-to-date, and [`preconditions:`][preconditions],
but this would halt the execution of the task instead of skipping it.

:::

## Prompt for Required Variables

For backward-compatibility reasons, this feature is disabled by default.
To enable it, either pass `--interactive` flag or add `interactive: true` to
your `.taskrc.yml`.

Once you do that, Task will basically starting prompting you in runtime for any
required variables. In the example below, `NAME` will be prompted at runtime:

```yaml
version: '3'

tasks:
  # Simple text input prompt
  greet:
    desc: Greet someone by name
    requires:
      vars:
        - NAME
    cmds:
      - echo "Hello, {{.NAME}}!"
```

If a given variable has an enum, Task will actually show a selection menu so you
can choose the right option instead of typing:

```yaml
version: '3'

tasks:
  # Enum selection (dropdown menu)
  deploy:
    desc: Deploy to an environment
    requires:
      vars:
        - name: ENVIRONMENT
          enum: [dev, staging, prod]
    cmds:
      - echo "Deploying to {{.ENVIRONMENT}}..."
```

Once again, check out the [documentation][prompt-docs] for more details, and the
[prompt examples][prompt-examples] from the test suite.

## Feedback

Let's us know if you have any feedback! You can find us on our
[Discord server][discord].

[release]: https://github.com/go-task/task/releases/tag/v3.47.0
[vmaerten]: https://github.com/vmaerten
[sources]: https://taskfile.dev/docs/guide#by-fingerprinting-locally-generated-files-and-their-sources
[status]: https://taskfile.dev/docs/guide#using-programmatic-checks-to-indicate-a-task-is-up-to-date
[preconditions]: https://taskfile.dev/docs/guide#using-programmatic-checks-to-cancel-the-execution-of-a-task-and-its-dependencies
[if-docs]: https://taskfile.dev/docs/guide#conditional-execution-with-if
[if-examples]: https://github.com/go-task/task/blob/main/testdata/if/Taskfile.yml
[prompt-docs]: https://taskfile.dev/docs/guide#prompting-for-missing-variables-interactively
[prompt-examples]: https://github.com/go-task/task/blob/main/testdata/interactive_vars/Taskfile.yml
[discord]: https://discord.com/invite/6TY36E39UK
