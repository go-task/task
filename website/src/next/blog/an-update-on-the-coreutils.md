---
title: 'An Update On The Core Utilities For Windows'
sidebarTitle: An Update On The Core Utilities For Windows
description: Task's built-in core utilities just got upgraded.
author: andreynering
date: 2026-11-27
tags: ['windows', 'core-utils']
outline: deep
editLink: false
---

# An Update On The Core Utilities For Windows

Past year, I published a [blog post][blogpost] here announcing support for the
core utilities for Windows. That was indeed a meaningful milestone for the
project, because without the core utilities the support for Windows was lacking.

## The previous implementation

At that post, I mentioned how we collaborated with other maintainers to make
that possible, including [mvdan] which maintains the [shell intepreter][mvdansh]
we use, and the maintainers of [u-root][uroot], a project that has Go
implementations of the core utilities we integrated at that moment.

To make that possible, we had to refactor some u-root core utilities to make
them importable, because they were `main` packages before. Also, we had to
isolate some state like environment variables, the working directory, etc.

These refactors added some complexity that the maintainers of u-root decided
they didn't want, so they reverted our refactors and we were stuck on an old
pinned commit since, without that ability to advance on that work (fix bugs,
add functionality, support more utilities, etc.).

## A brand new implementation

So, at [Charm][charm], the company I work for, we maintain [Crush][crush], a
coding agent that (surprise!) _also_ uses the same shell interpreter, and was
_also_ using u-root utilities to improve support on Windows. This means that
both projects had a common interest in having a good coreutils implementation.

Something that also happened in the meantime is that AI got so much better!
This made possible to ship a brand new implementation that much better fit the
need of both projects: [`charm.land/turtle`][turtle] 🐢

The new implementation is a big achievement on improving support for Windows.
The Turtle coreutils support _many, many_ more utilities. They also have a much
larger coverage in terms of supporting a lot more functionality from the POSIX
and GNU specs. They're largely covered by good tests as well.

We also took the opportunity to do some things right from the start, like
ensuring more state is isolated from the system, like the file system via an
interface.

In short, this was meant to happen at some point, and I'm happy that this day
has come!

## The current state

Starting on this release, Task ships with the new coreutils. By default, it's
enabled it on Windows because compatibility is more critical there. Also,
Windows is slow on calling external executables, so running built-in Go
utilities compiled within the binary makes things faster there.
On other operating systems, the system utilities will be prioritized but Task
will still fallback to Turtle implementation when a tool is not available in the
system.

If things goes well for this release, we plan to enable it by default on all
operating system in the next release!

Regardless of the default, we added an option to the `.taskrc.yml` so you can
choose how you want it to work. The option will be call `coreutils:`, with the
following options:

* `turtle`: uses built-in core utilities
* `system`: prefers system, falls back to turtle
* `auto`: prefers turtle on windows, system otherwise
* `off`: disables the middleware completely

::: info

With the exception of `off`, we're always prioritizing Turtle for the following
utilities on Windows, because Windows has conflicting implementations:
`expand`, `find`, `sort` and `timeout`.

:::

```yaml
# .taskrc.yml
coreutils: turtle # opt-in for the new coreutils sooner
```

To make sure it's working, run `--version` for any of the tools and you'll see
Turtle mentioned:

```bash
$ cat --version
cat (charm.land/turtle/coreutils) v0.1.0
https://charm.land/turtle
```

We're also still respecting the ENV mentioned on the previous blog post, but we
aliased it from `TASK_CORE_UTILS` to `TASK_COREUTILS` because it reads better.
You can either set any of the option above or a boolean like originally, in
which case we'll map `TASK_COREUTILS=true/1` to `turtle` and
`TASK_COREUTILS=false/0` to `auto`.

[blogpost]: /blog/windows-core-utils
[mvdan]: https://github.com/mvdan
[mvdansh]: https://github.com/mvdan/sh
[uroot]: https://github.com/u-root/u-root
[charm]: https://charm.land
[crush]: https://charm.land/crush
[turtle]: https://github.com/charmbracelet/turtle
