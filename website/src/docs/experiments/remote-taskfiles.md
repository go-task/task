---
title: 'Remote Taskfiles (#1317)'
description: Experimentation for using Taskfiles stored in remote locations
outline: deep
---

# Remote Taskfiles (#1317)

::: warning

All experimental features are subject to breaking changes and/or removal _at any
time_. We strongly recommend that you do not use these features in a production
environment. They are intended for testing and feedback only.

:::

::: info

To enable this experiment, set the environment variable:
`TASK_X_REMOTE_TASKFILES=1`. Check out
[our guide to enabling experiments](./index.md#enabling-experiments) for more
information.

:::

::: danger

Never run remote Taskfiles from sources that you do not trust.

:::

This experiment allows you to use Taskfiles which are stored in remote
locations. This applies to both the root Taskfile (aka. Entrypoint) and also
when including Taskfiles.

Task uses "nodes" to reference remote Taskfiles. There are a few different types
of node which you can use:

::: code-group

```text [HTTP/HTTPS]
https://raw.githubusercontent.com/go-task/task/main/website/src/public/Taskfile.yml
```

```text [Git over HTTP]
https://github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main
```

```text [Git over SSH]
git@github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main
```

:::

## Node Types

### HTTP/HTTPS

`https://raw.githubusercontent.com/go-task/task/main/website/src/public/Taskfile.yml`

This is the most basic type of remote node and works by downloading the file
from the specified URL. The file must be a valid Taskfile and can be of any
name. If a file is not found at the specified URL, Task will append each of the
supported file names in turn until it finds a valid file. If it still does not
find a valid Taskfile, an error is returned.

### Git over HTTP

`https://github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main`

This type of node works by downloading the file from a Git repository over
HTTP/HTTPS. The first part of the URL is the base URL of the Git repository.
This is the same URL that you would use to clone the repo over HTTP.

- You can optionally add the path to the Taskfile in the repository by appending
  `//<path>` to the URL.
- You can also optionally specify a branch or tag to use by appending
  `?ref=<ref>` to the end of the URL. If you omit a reference, the default
  branch will be used.

### Git over SSH

`git@github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main`

This type of node works by downloading the file from a Git repository over SSH.
The first part of the URL is the user and base URL of the Git repository. This
is the same URL that you would use to clone the repo over SSH.

To use Git over SSH, you need to make sure that your SSH agent has your private
SSH keys added so that they can be used during authentication.

- You can optionally add the path to the Taskfile in the repository by appending
  `//<path>` to the URL.
- You can also optionally specify a branch or tag to use by appending
  `?ref=<ref>` to the end of the URL. If you omit a reference, the default
  branch will be used.

Task has an example remote Taskfile in our repository that you can use for
testing and that we will use throughout this document:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - task: hello

  hello:
    cmds:
      - echo "Hello Task!"
```

## Specifying a remote entrypoint

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

## Including remote Taskfiles

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

### Authenticating using environment variables

The Taskfile location is processed by the templating system, so you can
reference environment variables in your URL if you need to add authentication.
For example:

```yaml
version: '3'

includes:
  my-remote-namespace: https://{{.TOKEN}}@raw.githubusercontent.com/my-org/my-repo/main/Taskfile.yml
```

## Security

### Automatic checksums

Running commands from sources that you do not control is always a potential
security risk. For this reason, we have added some automatic checks when using
remote Taskfiles:

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
flag or the `--trust` flag. The `--trust` flag allows you to specify trusted
hosts for remote Taskfiles, while `--yes` applies to all prompts in Task. You
can also configure trusted hosts in your [taskrc configuration](#trusted-hosts) using
`remote.trusted-hosts`. Before enabling automatic trust, you should:

1. Be sure that you trust the source and contents of the remote Taskfile.
2. Consider using a pinned version of the remote Taskfile (e.g. A link
   containing a commit hash) to prevent Task from automatically accepting a
   prompt that says a remote Taskfile has changed.

### Manual checksum pinning

Alternatively, if you expect the contents of your remote files to be a constant
value, you can pin the checksum of the included file instead:

```yaml
version: '3'

includes:
  included:
    taskfile: https://taskfile.dev
    checksum: c153e97e0b3a998a7ed2e61064c6ddaddd0de0c525feefd6bba8569827d8efe9
```

This will disable the automatic checksum prompts discussed above. However, if
the checksums do not match, Task will exit immediately with an error. When
setting this up for the first time, you may not know the correct value of the
checksum. There are a couple of ways you can obtain this:

1. Add the include normally without the `checksum` key. The first time you run
   the included Taskfile, a `.task/remote` temporary directory is created. Find
   the correct set of files for your included Taskfile and open the file that
   ends with `.checksum`. You can copy the contents of this file and paste it
   into the `checksum` key of your include. This method is safest as it allows
   you to inspect the downloaded Taskfile before you pin it.
2. Alternatively, add the include with a temporary random value in the
   `checksum` key. When you try to run the Taskfile, you will get an error that
   will report the incorrect expected checksum and the actual checksum. You can
   copy the actual checksum and replace your temporary random value.

### TLS

Task currently supports both `http` and `https` URLs. However, the `http`
requests will not execute by default unless you run the task with the
`--insecure` flag. This is to protect you from accidentally running a remote
Taskfile that is downloaded via an unencrypted connection. Sources that are not
protected by TLS are vulnerable to man-in-the-middle attacks and should be
avoided unless you know what you are doing.

## Caching & Running Offline

Whenever you run a remote Taskfile, the latest copy will be downloaded from the
internet and cached locally. This cached file will be used for all future
invocations of the Taskfile until the cache expires. Once it expires, Task will
download the latest copy of the file and update the cache. By default, the cache
is set to expire immediately. This means that Task will always fetch the latest
version. However, the cache expiry duration can be modified by setting the
`--expiry` flag.

If for any reason you lose access to the internet or you are running Task in
offline mode (via the `--offline` flag or `TASK_OFFLINE` environment variable),
Task will run the any available cached files _even if they are expired_. This
means that you should never be stuck without the ability to run your tasks as
long as you have downloaded a remote Taskfile at least once.

By default, Task will timeout requests to download remote files after 10 seconds
and look for a cached copy instead. This timeout can be configured by setting
the `--timeout` flag and specifying a duration. For example, `--timeout 5s` will
set the timeout to 5 seconds.

By default, the cache is stored in the Task temp directory (`.task`). You can
override the location of the cache by using the `--remote-cache-dir` flag, the
`remote.cache-dir` option in your [configuration file](#cache-dir), or the
`TASK_REMOTE_DIR` environment variable. This way, you can share the cache
between different projects.

You can force Task to ignore the cache and download the latest version by using
the `--download` flag.

You can use the `--clear-cache` flag to clear all cached remote files.

## Configuration

This experiment adds a new `remote` section to the
[configuration file](../reference/config.md).

- **Type**: `object`
- **Description**: Remote configuration settings for handling remote Taskfiles

```yaml
remote:
  insecure: false
  offline: false
  timeout: "30s"
  cache-expiry: "24h"
  cache-dir: ~/.task
  trusted-hosts:
    - github.com
    - gitlab.com
```

#### `insecure`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Allow insecure connections when fetching remote Taskfiles

```yaml
remote:
  insecure: true
```

#### `offline`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Work in offline mode, preventing remote Taskfile fetching

```yaml
remote:
  offline: true
```

#### `timeout`

- **Type**: `string`
- **Default**: 10s
- **Pattern**: `^[0-9]+(ns|us|µs|ms|s|m|h)$`
- **Description**: Timeout duration for remote operations (e.g., '30s', '5m')

```yaml
remote:
  timeout: "1m"
```

#### `cache-expiry`

- **Type**: `string`
- **Default**: 0s (no cache)
- **Pattern**: `^[0-9]+(ns|us|µs|ms|s|m|h)$`
- **Description**: Cache expiry duration for remote Taskfiles (e.g., '1h',
  '24h')

```yaml
remote:
  cache-expiry: "6h"
```

#### `cache-dir`

- **Type**: `string`
- **Default**: `.task`
- **Description**: Directory where remote Taskfiles are cached. Can be an
  absolute path (e.g., `/var/cache/task`) or relative to the Taskfile directory.
- **CLI equivalent**: `--remote-cache-dir`
- **Environment variable**: `TASK_REMOTE_DIR` (lowest priority)

```yaml
remote:
  cache-dir: ~/.task
```

#### `trusted-hosts`

- **Type**: `array of strings`
- **Default**: `[]` (empty list)
- **Description**: List of trusted hosts for remote Taskfiles. Hosts in this
  list will not prompt for confirmation when downloading Taskfiles
- **CLI equivalent**: `--trusted-hosts`

```yaml
remote:
  trusted-hosts:
    - github.com
    - gitlab.com
    - raw.githubusercontent.com
    - example.com:8080
```

Hosts in the trusted hosts list will automatically be trusted without prompting for
confirmation when they are first downloaded or when their checksums change. The
host matching includes the port if specified in the URL. Use with caution and
only add hosts you fully trust.

You can also specify trusted hosts via the command line:

```shell
# Trust specific host for this execution
task --trusted-hosts github.com -t https://github.com/user/repo.git//Taskfile.yml

# Trust multiple hosts (comma-separated)
task --trusted-hosts github.com,gitlab.com -t https://github.com/user/repo.git//Taskfile.yml

# Trust a host with a specific port
task --trusted-hosts example.com:8080 -t https://example.com:8080/Taskfile.yml
```
