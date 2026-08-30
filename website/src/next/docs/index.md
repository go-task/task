---
title: Documentation
description:
  Task is a task runner and build tool that aims to be simpler and easier to use
  than GNU Make. Start here to install it, learn it, or look something up.
section: Overview
docType: overview
outline: deep
---

# Documentation

Task is a task runner and build tool that aims to be simpler and easier to use
than [GNU Make](https://www.gnu.org/software/make/). You describe your tasks in
a YAML file called a `Taskfile`, and Task runs them.

## New to Task

Install the binary, then write your first Taskfile. It takes about five minutes.

- [Installation](./installation.md) — package managers, prebuilt binaries,
  building from source, and shell completions.
- [Getting Started](./getting-started.md) — your first Taskfile, run end to end.

## Using Task

The [Guide](./guide/index.md) covers everything Task can do, one topic per page:
running and defining tasks, variables, dependencies, up-to-date checks,
conditional execution, loops, includes, output modes and watch mode.

## Looking something up

- [Taskfile Schema](./reference/schema.md) — every key you can put in a
  Taskfile.
- [CLI](./reference/cli.md) — commands, flags and exit codes.
- [Templating](./reference/templating.md) — template functions and special
  variables.
- [Configuration](./reference/config.md) and
  [Environment](./reference/environment.md) — settings outside the Taskfile.

## Keeping up

- [Changelog](./changelog.md) — what shipped, and when.
- [Experiments](./experiments/index.md) and
  [Deprecations](./deprecations/index.md) — what is coming, and what is going
  away.
- [FAQ](./faq.md) — the questions that come up most often.
- [Community](./community.md) — integrations and tools built by other people.

## Using an AI coding assistant

[Task documentation for coding agents](../agents.md) is a compact map of these
pages plus the semantics that are easiest to get wrong. Every page is also
available as raw Markdown by appending `.md` to its URL, and the whole corpus is
at [/llms.txt](/llms.txt).
