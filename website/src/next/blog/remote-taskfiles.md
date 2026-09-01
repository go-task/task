---
title: Remote Taskfiles
description:
  Remote Taskfiles have been made generally available after nearly 3 years of
  experimentation.
author: pd93
date: 2026-08-18
tags: ['experiments', 'remote', 'new-features']
outline: deep
editLink: false
---

# Remote Taskfiles

<AuthorCard :author="$frontmatter.author" />

It's finally time! After nearly 3 years as an experiment, we're excited to
announce that today, [remote Taskfiles][remote-taskfiles] have been made
generally available. No more setting flags or config to enable it. It is enabled
by default and will continue to work seamlessly with your existing remote
taskfiles.

[remote-taskfiles]: ../docs/remote-taskfiles.md

<!-- more -->

## How did we get here?

We'll be the first to admit that it's taken a long time to get this stable, but
there's good reason for it. Remote taskfiles was the most upvoted feature
request and we wanted to get it right. Experiments allowed us to iterate slowly
and implement feedback without having to worry about releasing something that we
couldn't take back later.

We received a ton of feedback from you on the main issue thread and we're very
grateful for all the comments and suggestions as you've patiently waited for
something stable to land.

No substantial changes have been made to the API recently and the stream of
feedback has now slowed, so we feel that the time is right to move forwards with
it.

## How do I use it?

We are aware that, despite the stability warnings, many of you are already using
remote taskfiles in production today. So, you'll be happy to hear that you do
not need to change anything going forwards! Once you've updated to the latest
version of Task (v3.53.0+), remote taskfiles will be enabled by default and you
can remove any config/flags that you previously used to enable it in your own
time.

If you are still using an environment variable/flag/config to enable it, you may
see a small warning appear that lets you know that this is no longer necessary.

For those that haven't used it before, here's a little taste of what you're now
able to do:

### Using a remote Taskfile as an entrypoint

By default, Task will look for one of the supported file names on your local
filesystem. If you want to use a remote file instead, you can pass its URI into
the `--taskfile`/`-t` flag just like you would to specify a different local
file. For example:

::: code-group

```shell [HTTP/HTTPS]
$ task --taskfile https://raw.githubusercontent.com/go-task/task/main/website/src/public/Taskfile.yml
task: [hello] echo "Hello Task!"
Hello Task!
```

```shell [Git over HTTP]
$ task --taskfile https://github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main
task: [hello] echo "Hello Task!"
Hello Task!
```

```shell [Git over SSH]
$ task --taskfile git@github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main
task: [hello] echo "Hello Task!"
Hello Task!
```

:::

For testing purposes, we host an example Taskfile at
[taskfile.dev/Taskfile.yml](https://taskfile.dev/Taskfile.yml) that you can use
to try this out. This is useful to check that your installation of task is
working.

```shell
$ task --taskfile https://taskfile.dev/Taskfile.yml
# Or for short
$ task -t https://taskfile.dev
```

### Including remote Taskfiles

Including a remote file works exactly the same way that including a local file
does. You just need to replace the local path with a remote URI. Any tasks in
the remote Taskfile will be available to run from your main Taskfile.

::: code-group

```yaml [HTTP/HTTPS]
version: '3'

includes:
  my-remote-namespace: https://raw.githubusercontent.com/go-task/task/main/website/src/public/Taskfile.yml
```

```yaml [Git over HTTP]
version: '3'

includes:
  my-remote-namespace: https://github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main
```

```yaml [Git over SSH]
version: '3'

includes:
  my-remote-namespace: git@github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main
```

:::

```shell
$ task my-remote-namespace:hello
task: [hello] echo "Hello Task!"
Hello Task!
```

## Further reading

To learn more, you can check out our
[brand new documentation for remote Taskfiles](../docs/remote-taskfiles.md).
Make sure you're running v3.53.0+ before you get started.
