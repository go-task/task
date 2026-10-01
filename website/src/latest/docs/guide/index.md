---
title: Guide
description:
  Learn to define tasks, supply inputs, control execution, configure the
  environment and output, and include other Taskfiles.
section: Guide
docType: guide
outline: deep
---

# Guide

Use the guide to learn a feature or solve a specific problem. If you are new to
Task, start with the [Quick Start](../getting-started.md). For accepted keys,
flags and functions, use the [Taskfile Schema](../reference/schema.md),
[CLI](../reference/cli.md) and [Templating](../reference/templating.md)
references.

<GuideRedirect />

## Writing and running tasks {#writing-and-running-tasks}

- [Defining tasks](./defining-tasks.md): command lists, descriptions, aliases
  and display labels.
- [Running tasks](./running-tasks.md): choosing a Taskfile, running from other
  directories, previewing commands and using interactive applications.

## Variables and arguments {#variables-and-environment}

- [Variables](./variables.md): declaring values, computing them with shell
  commands, preserving types and understanding resolution order.
- [Command-line arguments](./arguments.md): assigning variables, forwarding
  arguments with `--` and capturing values from task names.
- [Validation and prompts](./required-variables.md): requiring inputs,
  restricting allowed values and asking for missing values.
- [Secret variables](./secret-variables.md): loading sensitive values and
  masking them in Task's command logs.

## Task execution {#controlling-what-runs}

- [Dependencies and task calls](./dependencies.md): running tasks concurrently
  or in sequence, limiting concurrency and controlling repeated calls.
- [Loops](./loops.md): repeating commands or task calls over values, files and
  matrices.
- [Conditional execution](./conditional-execution.md): skipping optional work
  with `if`, enforcing requirements with `preconditions` and asking for
  [confirmation](./conditional-execution.md#confirmation-prompts).
- [Errors and cleanup](./errors-and-cleanup.md): continuing after selected
  failures and running cleanup with `defer`.
- [Up-to-date checks](./up-to-date.md): skipping unchanged builds, checking for
  missing outputs and combining file checks with custom conditions.
- [Watch mode](./watch.md): rerunning tasks when source files change.

## Environment and output {#execution-environment}

- [Environment variables](./environment.md): setting the environment for
  commands and loading `.env` files.
- [Platforms and shells](./platforms.md): selecting platform-specific commands,
  understanding shell context and setting shell options.
- [Output and logging](./output.md): streaming, grouping and prefixing output,
  hiding command echoes and displaying CI annotations.

## Including Taskfiles {#composing-taskfiles}

- [Including Taskfiles](./includes.md): sharing tasks with namespaces,
  configuring includes and passing variables to them.
- [Remote Taskfiles](../remote-taskfiles.md): loading Taskfiles over HTTP or Git
  and managing trust, checksums and cached copies.
