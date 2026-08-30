---
title: Task documentation for coding agents
description:
  A compact map of Task's documentation, plus the execution semantics that are
  easiest to get wrong when generating a Taskfile.
outline: deep
---

# Task documentation for coding agents

Task is a cross-platform task runner and build tool. Its configuration file is
normally named `Taskfile.yml`, and new files should use schema version `3`.

Every page linked below is also available as raw Markdown: append `.md` to its
URL. The curated index is at [/llms.txt](/llms.txt) and the full corpus at
[/llms-full.txt](/llms-full.txt).

## Where to look

- [Getting Started](./docs/getting-started.md): the shape of a Taskfile.
- [Taskfile Schema](./docs/reference/schema.md): the source of truth for keys,
  types and accepted values. Check here before assuming a field exists.
- [CLI](./docs/reference/cli.md): commands, flags and exit codes.
- [Templating](./docs/reference/templating.md): every template function and
  special variable. Check here before inventing one.
- [Guide](./docs/guide/): one page per topic, for how to do a thing.
- [Variable resolution](./docs/concepts/variable-resolution.md) and
  [Dependencies and concurrency](./docs/concepts/dependencies-and-concurrency.md):
  for when the behaviour matters more than the procedure.

## Semantics that are easy to get wrong

1. `vars` are template values only: `$FOO` in a command never sees one. `env` is
   exported, so `$FOO` works. A template also sees `env` declared at the root of
   the Taskfile, but **not** `env` declared on a task, which renders empty.
2. A task's own `vars:` cannot be overridden from the command line. Put the
   default in global `vars:` if the caller needs to supply a value.
3. `vars:` given on an `includes:` entry are defaults, not overrides: the
   included Taskfile's own `vars:` are applied after them and win.
4. Everything in `deps` may run concurrently and in any order. A `task:`
   reference inside `cmds` runs at its position and blocks the next command. If
   order matters, use `cmds`.
5. Each command runs in its own shell. Nothing carries over between them, not
   `cd` and not an exported variable. Use Task's `dir:` and `env:` instead.
6. A template renders text. Use `ref:` to pass an array or a map without
   flattening it to a string.
7. A passing `status:` means the task is already up to date and is skipped. A
   failing `preconditions:` means the task must not run at all. They are not
   interchangeable.
8. `defer:` commands run in reverse order of declaration, and run whether the
   task succeeded or failed.
9. Remote Taskfiles execute code from wherever they are fetched. See
   [Remote Taskfiles](./docs/remote-taskfiles.md) for the trust and checksum
   rules.
10. Portability is about every command inside a task, not just Task itself. A
    task is only cross-platform if its commands are.

## Before writing a Taskfile

- Confirm the feature exists in the schema for the version in use.
- Prefer plain, readable tasks over dense templating.
- Use `deps` only where concurrent execution is actually correct.
- Never embed credentials. Read them from the environment or a secret manager.
  `secret: true` masks a value in Task's own logs, but it only works on `vars:`,
  not on `env:`, and it never masks what a command itself prints. Treat it as
  one less place a secret is echoed, not as protection.
- Give tasks a `desc:` so they show up in `task --list`, and validate inputs
  with `requires:` or `preconditions:`.
