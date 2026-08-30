---
title: Guide
description:
  An index of every topic in the Task guide, from running your first task to
  composing Taskfiles across repositories.
outline: deep
---

# Guide

The guide covers everything Task can do once you have written your first
Taskfile. Each page below is self-contained; start wherever your problem is.

<GuideRedirect />

## Writing and running tasks

- [Running tasks](./running-tasks.md) — how Task finds a Taskfile, and how to
  run one from a subdirectory, your home directory, standard input or a dry run.
- [Defining tasks](./defining-tasks.md) — syntax shortcuts, internal tasks,
  aliases, the directory a task runs in, and its help text.
- [Passing arguments](./arguments.md) — forwarding command line arguments with
  `--`, and matching part of a task's name with a wildcard.

## Variables and environment

- [Variables](./variables.md) — static, dynamic, map and secret variables, their
  scope, and how they reference each other.
- [Environment variables](./environment.md) — setting them per task or globally,
  and loading them from `.env` files.
- [Required variables and prompts](./required-variables.md) — requiring
  variables, restricting them to allowed values, and prompting for them.

## Controlling what runs

- [Dependencies and task calls](./dependencies.md) — `deps`, calling a task from
  `cmds`, and cleanup with `defer`.
- [Skipping work that is up to date](./up-to-date.md) — source and generated
  file fingerprints, and your own `status` checks.
- [Conditional execution](./conditional-execution.md) — `preconditions`, `if`,
  and the flags that limit when a task runs.
- [Loops](./loops.md) — repeating a command over a list, a matrix, a variable,
  your sources, or other tasks.

## Composing Taskfiles

- [Including other Taskfiles](./includes.md) — namespaces, optional and internal
  includes, flattening, and per-include variables.
- [Remote Taskfiles](../remote-taskfiles.md) — running and including Taskfiles
  served over HTTP or Git, and the checksum rules that guard them.

## Execution environment

- [Output and logging](./output.md) — output modes, silent mode, ignoring
  errors, and CI annotations.
- [Platform-specific behaviour](./platforms.md) — restricting tasks to an OS or
  architecture, and shell options.
- [Watch mode](./watch.md) — re-running a task when its sources change.
