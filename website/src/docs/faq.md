---
title: FAQ
description:
  Frequently asked questions about Task, including ETAs, shell limitations, and
  Windows compatibility
outline: deep
---

# FAQ

This page contains a list of frequently asked questions about Task.

## When will \<feature\> be released? / ETAs

Task is _free_ and _open source_ project maintained by a small group of
volunteers with full time jobs and lives outside of the project. Because of
this, it is difficult to predict how much time we will be able to dedicate to
the project in advance and we don't want to make any promises that we can't
keep. For this reason, we are unable to provide ETAs for new features or
releases. We make a "best effort" to provide regular releases and fix bugs in a
timely fashion, but sometimes our personal lives must take priority.

ETAs are probably the number one question we (and maintainers of other open
source projects) get asked. We understand that you are passionate about the
project, but it can be overwhelming to be asked this question so often. Please
be patient and avoid asking for ETAs.

The best way to speed things up is to contribute to the project yourself. We
always appreciate new contributors. If you are interested in contributing, check
out the [contributing guide](./contributing.md).

## Why won't my task update my shell environment?

This is a limitation of how shells work. Task runs as a subprocess of your
current shell, so it can't change the environment of the shell that started it.
This limitation is shared by other task runners and build tools too.

A common way to work around this is to create a task that will generate output
that can be parsed by your shell. For example, to set an environment variable on
your shell you can write a task like this:

```yaml
my-shell-env:
  cmds:
    - echo "export FOO=foo"
    - echo "export BAR=bar"
```

Now run `eval $(task my-shell-env)` and the variables `$FOO` and `$BAR` will be
available in your shell.

## I can't reuse my shell in a task's commands

Task runs each command as a separate shell process, so something you do in one
command won't effect any future commands. For example, this won't work:

```yaml
version: '3'

tasks:
  foo:
    cmds:
      - a=foo
      - echo $a
      # outputs ""
```

To work around this you can either use a multiline command:

```yaml
version: '3'

tasks:
  foo:
    cmds:
      - |
        a=foo
        echo $a
      # outputs "foo"
```

Or for more complex multi-line commands it is recommended to move your code into
a separate file and call that instead:

```yaml
version: '3'

tasks:
  foo:
    cmds:
      - ./foo-printer.bash
```

```shell
#!/bin/bash
a=foo
echo $a
```

## Are shell core utilities available on Windows?

The most common ones, yes. And we might add more in the future.
This is possible because Task compiles a small set of core utilities in Go and
enables them by default on Windows for greater compatibility.

It's possible to control whether these builtin core utilities are used or not
with the [`TASK_CORE_UTILS`](/docs/reference/environment#task-core-utils)
environment variable:

```bash
# Enable, even on non-Windows platforms
env TASK_CORE_UTILS=1 task ...

# Disable, even on Windows
env TASK_CORE_UTILS=0 task ...
```

This is the list of core utils that are currently available:

* `base64`
* `cat`
* `chmod`
* `cp`
* `find`
* `gzip`
* `ls`
* `mkdir`
* `mktemp`
* `mv`
* `rm`
* `shasum`
* `tar`
* `touch`
* `xargs`
* (more might be added in the future)
