---
title: Remote Taskfiles
description:
  Guide to loading and securely using Taskfiles from HTTP and Git sources
section: Guide
docType: guide
outline: deep
---

# Remote Taskfiles

Use a remote Taskfile to share tasks across projects without copying the file
into each repository. You can call it directly or include it under a namespace.

::: danger

Only run remote Taskfiles from sources you trust. Their commands execute on your
machine, just like commands from a local Taskfile.

:::

The examples use Task's public sample, whose `hello` task prints `Hello Task!`.
On first use, Task asks you to trust the downloaded file. Review it before
accepting; see [trust and checksums](#security) for subsequent changes and CI.

## Run a remote Taskfile {#specifying-a-remote-entrypoint}

Pass the remote location to `--taskfile` (or `-t`). With no task name, Task runs
the remote file's `default` task. Choose a transport below:

::: code-group

```shell [HTTP/HTTPS]
$ task --taskfile https://raw.githubusercontent.com/go-task/task/main/website/src/public/Taskfile.yml
task: [hello] echo "Hello Task!"
Hello Task!
```

```shell [Git over HTTP]
$ task --taskfile 'https://github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main'
task: [hello] echo "Hello Task!"
Hello Task!
```

```shell [Git over SSH]
$ task --taskfile 'git@github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main'
task: [hello] echo "Hello Task!"
Hello Task!
```

:::

## Include shared tasks {#including-remote-taskfiles}

Set the include location to a remote URL instead of a local path. The tasks
become available under the namespace you choose, just like
[local includes](./guide/includes.md):

::: code-group

```yaml [HTTP/HTTPS]
version: '3'

includes:
  shared: https://raw.githubusercontent.com/go-task/task/main/website/src/public/Taskfile.yml
```

```yaml [Git over HTTP]
version: '3'

includes:
  shared: https://github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main
```

```yaml [Git over SSH]
version: '3'

includes:
  shared: git@github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main
```

:::

```shell
$ task shared:hello
task: [hello] echo "Hello Task!"
Hello Task!
```

### Supply authentication {#authenticating-using-environment-variables}

The Taskfile location is processed by the templating system, so you can
reference environment variables in your URL if you need to add authentication.
For example:

```yaml
version: '3'

includes:
  shared: https://{{.TOKEN}}@raw.githubusercontent.com/my-org/my-repo/main/Taskfile.yml
```

Prefer the [`remote.auth`](./reference/config.md#remote-auth) configuration
option when the server accepts a header. A credential in the URL ends up in
error messages and in the confirmation prompt, and the include can no longer be
committed as-is.

## Choose a source {#node-types}

### HTTP/HTTPS

`https://raw.githubusercontent.com/go-task/task/main/website/src/public/Taskfile.yml`

Use a direct URL when the server exposes the Taskfile as plain text. Task
downloads the file from that URL. The file must be a valid Taskfile and can be
of any name. If a file is not found at the specified URL, Task will append each
of the supported file names in turn until it finds a valid file. If it still
does not find a valid Taskfile, an error is returned.

### Git over HTTP

`https://github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main`

Use a Git URL to select a file from a repository over HTTP/HTTPS. The first part
of the URL is the base URL of the Git repository. This is the same URL that you
would use to clone the repo over HTTP.

- You can optionally add the path to the Taskfile in the repository by appending
  `//<path>` to the URL.
- You can also optionally specify a branch or tag to use by appending
  `?ref=<ref>` to the end of the URL. If you omit a reference, the default
  branch will be used.

### Git over SSH

`git@github.com/go-task/task.git//website/src/public/Taskfile.yml?ref=main`

Use an SSH Git URL when repository access depends on your SSH credentials. The
first part of the URL is the user and base URL of the Git repository. This is
the same URL that you would use to clone the repo over SSH.

To use Git over SSH, you need to make sure that your SSH agent has your private
SSH keys added so that they can be used during authentication.

- You can optionally add the path to the Taskfile in the repository by appending
  `//<path>` to the URL.
- You can also optionally specify a branch or tag to use by appending
  `?ref=<ref>` to the end of the URL. If you omit a reference, the default
  branch will be used.

## Locate files and directories {#special-variables}

The file-path [special variables](./reference/templating.md#file-paths) behave
differently when a Taskfile is loaded from a remote source, because there is no
local file or directory that corresponds 1:1 to the Taskfile:

| Variable                     | Value when loaded remotely                                                                                                                              |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `TASKFILE` / `ROOT_TASKFILE` | The original URL, unchanged                                                                                                                             |
| `TASKFILE_DIR` / `ROOT_DIR`  | Empty string, a directory variable cannot point to a URL                                                                                                |
| `TASK_DIR`                   | Resolved against `USER_WORKING_DIR` (relative `dir:` → joined with `USER_WORKING_DIR`, empty `dir:` → `USER_WORKING_DIR`, absolute `dir:` → kept as-is) |

If a remote Taskfile includes a local Taskfile (or vice-versa), each variable
reflects the source of the Taskfile it refers to.

## Review trust and changes {#security}

### Review changed files {#automatic-checksums}

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
flag or the `--trusted-hosts` flag. The `--trusted-hosts` flag allows you to
specify trusted hosts for remote Taskfiles, while `--yes` applies to all prompts
in Task. You can also configure trusted hosts in your
[taskrc configuration](./reference/config.md#remote-trusted-hosts) using
`remote.trusted-hosts`. Before enabling automatic trust, you should:

1. Be sure that you trust the source and contents of the remote Taskfile.
2. Consider using a pinned version of the remote Taskfile (e.g. A link
   containing a commit hash) to prevent Task from automatically accepting a
   prompt that says a remote Taskfile has changed.

### Pin reviewed contents {#manual-checksum-pinning}

Alternatively, if you expect the contents of your remote files to be a constant
value, you can pin the checksum of the included file instead:

```yaml
version: '3'

includes:
  included:
    taskfile: https://raw.githubusercontent.com/go-task/task/main/website/src/public/Taskfile.yml
    checksum: '<sha256-of-reviewed-file>'
```

Replace the placeholder with the SHA-256 checksum of the file you reviewed. A
pinned checksum disables the automatic checksum prompts discussed above.
However, if the checksums do not match, Task will exit immediately with an
error. When setting this up for the first time, you may not know the correct
value of the checksum. There are a couple of ways you can obtain this:

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

### Use TLS {#tls}

Task currently supports both `http` and `https` URLs. However, the `http`
requests will not execute by default unless you run the task with the
`--insecure` flag. This is to protect you from accidentally running a remote
Taskfile that is downloaded via an unencrypted connection. Sources that are not
protected by TLS are vulnerable to man-in-the-middle attacks and should be
avoided unless you know what you are doing.

#### Add certificates {#custom-certificates}

If your remote Taskfiles are hosted on a server that uses a custom CA
certificate (e.g., a corporate internal server), you can specify the CA
certificate using the `--cacert` flag:

```shell
task --taskfile https://internal.example.com/Taskfile.yml --cacert /path/to/ca.crt
```

For servers that require client certificate authentication (mTLS), you can
provide a client certificate and key:

```shell
task --taskfile https://secure.example.com/Taskfile.yml \
  --cert /path/to/client.crt \
  --cert-key /path/to/client.key
```

::: warning

Encrypted private keys are not currently supported. If your key is encrypted,
you must decrypt it first:

```shell
openssl rsa -in encrypted.key -out decrypted.key
```

:::

These options can also be configured in the
[configuration file](#configuration).

## Reuse cached files {#caching-running-offline}

| What do you need?                            | Option          |
| -------------------------------------------- | --------------- |
| Reuse a downloaded copy for one hour         | `--expiry 1h`   |
| Use an existing cache without network access | `--offline`     |
| Fetch a fresh copy now                       | `--download`    |
| Remove cached remote files                   | `--clear-cache` |

For example, run `task --offline shared:hello` after downloading and trusting
the include above. Offline mode needs an existing cached copy.

Whenever you run a remote Taskfile, the latest copy will be downloaded from the
internet and cached locally. This cached file will be used for all future
invocations of the Taskfile until the cache expires. Once it expires, Task will
download the latest copy of the file and update the cache. By default, the cache
is set to expire immediately. This means that Task will always fetch the latest
version. However, the cache expiry duration can be modified by setting the
`--expiry` flag.

If for any reason you lose access to the internet or you are running Task in
offline mode (via the `--offline` flag or `TASK_REMOTE_OFFLINE` environment
variable), Task will run any available cached files _even if they are expired_.
An uncached remote file cannot run offline.

By default, Task will timeout requests to download remote files after 10 seconds
and look for a cached copy instead. This timeout can be configured by setting
the `--timeout` flag and specifying a duration. For example, `--timeout 5s` will
set the timeout to 5 seconds.

By default, the cache is stored in the Task temp directory (`.task`). You can
override the location of the cache by using the `--remote-cache-dir` flag, the
`remote.cache-dir` option in your
[configuration file](./reference/config.md#remote-cache-dir), or the
`TASK_REMOTE_CACHE_DIR` environment variable. This way, you can share the cache
between different projects.

You can force Task to ignore the cache and download the latest version by using
the `--download` flag.

You can use the `--clear-cache` flag to clear all cached remote files.

## Set remote defaults {#configuration}

It is possible to configure the default behavior of remote Taskfiles using
configuration. Check out the following references for more information on how to
set up this configuration using flags, environment variables or a configuration
file:

- [CLI flags](./reference/cli.md#remote)
- [Environment Variables](./reference/environment.md#remote-taskfile-variables)
- [Config File](./reference/config.md#remote)
