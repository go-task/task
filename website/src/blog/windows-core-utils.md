---
title: 'Announcing Built-in Core Utilities for Windows'
description: The journey of enhancing Windows support in Task.
author: andreynering
date: 2025-09-15
outline: deep
---

# Announcing Built-in Core Utilities for Windows

<AuthorCard :author="$frontmatter.author" />

When I started Task back in 2017, one of my biggest goals was to build a task
runner that would work well on all major platforms, including Windows. At the
time, I was using Windows as my main platform, and it caught my attention how
much of a pain it was to get a working version of Make on Windows, for example.

## The very beginning

The very first versions, which looked very prototyp-ish, already supported
Windows, but it was falling back to Command Prompt (`cmd.exe`) to run commands
if `bash` wasn't available in the system. That didn't mean you couldn't run Bash
commands on Windows necessarily, because if you used Task inside Git Bash, it
would expose `bash.exe` into your `$PATH`, which made possible for Task to use
it. Outside of it, you would be out of luck, though, because running on Command
Prompt meant that the commands wouldn't be really compatible.

## Adopting a shell interpreter

I didn't take too much time to discover that there was [a shell interpreter for
Go that was very solid][mvdan], and I quickly adopted it to ensure we would be
able to run commands with consistency across all platforms. It was fun because
once adopted, I had the opportunity to [make some contributions to make it more
stable][mvdan-prs], which I'm sure the author appreciated.

## The lack of core utilities

There was one important thing missing, though. If you needed to use any core
utilities on Windows, like copying files with `cp`, moving with `mv`, creating a
directory with `mkdir -p`, that likely would just fail :boom:. There were
workarounds, of course. You could run `task` inside Git Bash which exposed core
utils in `$PATH` for you, or you could install these core utils manually (there
are a good number of alternative implementations available for download).

That was still far from ideal, though. One of my biggest goals with Task is that
it should "just work", even on Windows. Requiring additional setup to make
things work is exactly what I wanted to avoid.

## They finally arrive!

And here we are, in 2025, 8 years after the initial release. We might be late,
but I'm happy nonetheless. From now on, the following core utilities will be
available on Windows. This is the start. We want to add more with time.

- `base64`
- `cat`
- `chmod`
- `cp`
- `find`
- `gzip`
- `ls`
- `mkdir`
- `mktemp`
- `mv`
- `rm`
- `shasum`
- `tar`
- `touch`
- `xargs`

## How we made this possible

This was made possible via a collaboration with the maintainers of other Go
projects.

### u-root/u-root

We are using the core utilities implementations in Go from the [u-root][u-root]
project. It wasn't as simple as it sounds because they have originally
implemented every core util as a standalone `main` package, which means we
couldn't just import and use them as libraries. We had some discussion and we
agreed on a common [interface][uroot-interface] and [base
implementation][uroot-base]. Then, I refactored one-by-one of the core utils in
the list above. This is the reason we don't have all of them: there are too
many! But the good news is that we can refactor more with time and include them
in Task.

### mvdan/sh

The other collaboration was with the maintainer of the shell interpreter. He
agreed on having [an official middleware][middleware] to expose these core
utilities. This means that other projects that use the shell interpreter can
also benefit from this work, and as more utilities are included, those projects
will benefit as well.

## Can I choose whether to use them or not?

Yes. We added a new environment variable called
[`TASK_CORE_UTILS`][task-core-utils] to control if the Go implementations are
used or not. By default, this is `true` on Windows and `false` on other
platforms. You can override it like this:

```bash
# Enable, even on non-Windows platforms
env TASK_CORE_UTILS=1 task ...

# Disable, even on Windows
env TASK_CORE_UTILS=0 task ...
```

We'll consider making this enabled by default on all platforms in the future. In
the meantime, we're still using the system core utils on non-Windows platforms
to avoid regressions as the Go implementations may not be 100% compatible with
the system ones.

## Feedback

If you have any feedback about this feature, join our [Discord server][discord]
or [open an issue][gh-issue] on GitHub.

Also, if Task is useful for you or your company, consider [sponsoring the
project][sponsor]!

[mvdan]: https://github.com/mvdan/sh
[mvdan-prs]:
  https://github.com/mvdan/sh/pulls?q=is%3Apr+author%3Aandreynering+is%3Aclosed+sort%3Acreated-asc
[u-root]: https://github.com/u-root/u-root
[uroot-interface]:
  https://github.com/u-root/u-root/blob/main/pkg/core/command.go
[uroot-base]: https://github.com/u-root/u-root/blob/main/pkg/core/base.go
[middleware]:
  https://github.com/mvdan/sh/blob/master/moreinterp/coreutils/coreutils.go
[task-core-utils]: /docs/reference/environment#task-core-utils
[discord]: https://discord.com/invite/6TY36E39UK
[gh-issue]: https://github.com/go-task/task/issues
[sponsor]: /donate
