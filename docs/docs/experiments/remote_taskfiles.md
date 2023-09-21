---
slug: /experiments/remote-taskfiles/
---

# Remote Taskfiles

- Issue: [#1317][remote-taskfiles-experiment]
- Environment variable: `TASK_X_REMOTE_TASKFILES=1`

This experiment allows you to specify a remote Taskfile URL when including a
Taskfile. For example:

```yaml
version: '3'

includes:
  my-remote-namespace: https://raw.githubusercontent.com/my-org/my-repo/main/Taskfile.yml
```

This works exactly the same way that including a local file does. Any tasks in
the remote Taskfile will be available to run from your main Taskfile via the
namespace `my-remote-namespace`. For example, if the remote file contains the
following:

```yaml
version: '3'

tasks:
  hello:
    silent: true
    cmds:
      - echo "Hello from the remote Taskfile!"
```

and you run `task my-remote-namespace:hello`, it will print the text: "Hello
from the remote Taskfile!" to your console.

## Security

Running commands from sources that you do not control is always a potential
security risk. For this reason, we have added some checks when using remote
Taskfiles:

1. When running a task from a remote Taskfile for the first time, Task will
   print a warning to the console asking you to check that you are sure that you
   trust the source of the Taskfile. If you do not accept the prompt, then Task
   will exit with code `104` (not trusted) and nothing will run. If you accept
   the prompt, the remote Taskfile will run and further calls to the remote
   Taskfile will not prompt you again.
2. Whenever you run a remote Taskfile, Task will create and store a checksum of
   the file that you are running. If the checksum changes, then Task will print
   another warning to the console to inform you that the contents of the remote
   file has changed. If you do not accept the prompt, then Task will exit with
   code `104` (not trusted) and nothing will run. If you accept the prompt, the
   checksum will be updated and the remote Taskfile will run.

Sometimes you need to run Task in an environment that does not have an
interactive terminal, so you are not able to accept a prompt. In these cases you
are able to tell task to accept these prompts automatically by using the `--yes`
flag. Before enabling this flag, you should:

1. Be sure that you trust the source and contents of the remote Taskfile.
2. Consider using a pinned version of the remote Taskfile (e.g. A link
   containing a commit hash) to prevent Task from automatically accepting a
   prompt that says a remote Taskfile has changed.

Task currently supports both `http` and `https` URLs. However, the `http`
requests will not execute by default unless you run the task with the
`--insecure` flag. This is to protect you from accidentally running a remote
Taskfile that is hosted on and unencrypted connection. Sources that are not
protected by TLS are vulnerable to [man-in-the-middle
attacks][man-in-the-middle-attacks] and should be avoided unless you know what
you are doing.

## Caching & Running Offline

If for whatever reason, you don't have access to the internet, but you still
need to be able to run your tasks, you are able to use the `--download` flag to
store a cached copy of the remote Taskfile.

<!-- TODO: The following behavior may change -->

If Task detects that you have a local copy of the remote Taskfile, it will use
your local copy instead of downloading the remote file. You can force Task to
work offline by using the `--offline` flag. This will prevent Task from making
any calls to remote sources.

<!-- prettier-ignore-start -->
[remote-taskfiles-experiment]: https://github.com/go-task/task/issues/1317
[man-in-the-middle-attacks]: https://en.wikipedia.org/wiki/Man-in-the-middle_attack
<!-- prettier-ignore-end -->
