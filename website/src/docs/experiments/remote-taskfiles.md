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

### Authenticating using HTTP headers

For a more secure approach that keeps credentials out of URLs, you can configure
HTTP headers in your [taskrc configuration file](#headers). This is particularly useful for
services like GitLab that use custom authentication headers.

<div v-pre>

::: code-group

```yaml [GitHub with Bearer token]
remote:
  headers:
    raw.githubusercontent.com:
      Authorization: "Bearer {{.GITHUB_TOKEN}}"
```

```yaml [GitLab with private token]
remote:
  headers:
    gitlab.com:
      PRIVATE-TOKEN: "{{.GITLAB_TOKEN}}"
```

```yaml [Multiple hosts]
remote:
  headers:
    raw.githubusercontent.com:
      Authorization: "Bearer {{.GITHUB_TOKEN}}"
    gitlab.com:
      PRIVATE-TOKEN: "{{.GITLAB_TOKEN}}"
    internal.company.com:
      X-API-Key: "{{.INTERNAL_API_KEY}}"
```

:::

</div>

The header values support environment variable expansion using the templating
system. Simply set the corresponding environment variables before running Task:

```shell
export GITHUB_TOKEN="ghp_..."
export GITLAB_TOKEN="glpat-..."
task my-remote-namespace:hello
```

#### Using the `--header` flag

You can also specify headers directly via the `--header` flag for ad-hoc use or
testing, without modifying your `.taskrc.yml` file. The format is:

```shell
--header "host:Header-Name=value"
```

Examples:

```shell
# Single header
task --header "raw.githubusercontent.com:Authorization=Bearer $GITHUB_TOKEN" \
     -t https://raw.githubusercontent.com/...

# Multiple headers
task --header "gitlab.com:PRIVATE-TOKEN=$GITLAB_TOKEN" \
     --header "gitlab.com:X-Custom-Header=value" \
     my-remote-task

# Override config file headers
task --header "github.com:Authorization=Bearer $DIFFERENT_TOKEN" \
     my-remote-task
```

The `--header` flag can be repeated multiple times. When both CLI flags and
config file headers are present, **CLI flags take precedence** for the specific
host and header name combination.

#### How headers work

Headers are matched by hostname (including port if specified in the URL), and
are automatically applied to all requests to that host. This approach is more
secure than embedding credentials in URLs because:

- Credentials are never visible in process listings or logs
- Headers are not part of the cached URL/file path
- Different headers can be configured per host
- CLI flags allow for ad-hoc authentication without modifying config files

::: warning Security Considerations

When using HTTP headers for authentication:

1. **Always use environment variables**: Never commit actual credentials to your
   `.taskrc.yml` file. Always use template syntax like <span v-pre>`{{.TOKEN}}`</span> and set the
   value via environment variables.

2. **Use HTTPS only**: Authentication headers should only be sent over HTTPS
   connections. Avoid using the `--insecure` flag with authenticated requests,
   as this sends credentials in plaintext over HTTP.

3. **Protected headers**: For security, Task prevents you from overriding
   critical HTTP headers like `Host`, `Content-Length`, `Transfer-Encoding`,
   `Connection`, and `Upgrade`. These are managed automatically by the HTTP
   library.

4. **Host matching is exact**: Headers for `github.com` will not be sent to
   `api.github.com` or `evil.github.com`. You must configure headers for each
   specific hostname you use.

:::

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

#### Custom Certificates

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

These options can also be configured in the [configuration file](#configuration).

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

<div v-pre>

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
  cacert: ""
  cert: ""
  cert-key: ""
  headers:
    raw.githubusercontent.com:
      Authorization: "Bearer {{.GITHUB_TOKEN}}"
    gitlab.com:
      PRIVATE-TOKEN: "{{.GITLAB_TOKEN}}"
```

</div>

#### `insecure`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Allow insecure connections when fetching remote Taskfiles
- **CLI equivalent**: `--insecure`
- **Environment variable**: `TASK_REMOTE_INSECURE`

```yaml
remote:
  insecure: true
```

#### `offline`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Work in offline mode, preventing remote Taskfile fetching
- **CLI equivalent**: `--offline`
- **Environment variable**: `TASK_REMOTE_OFFLINE`

```yaml
remote:
  offline: true
```

#### `timeout`

- **Type**: `string`
- **Default**: 10s
- **Pattern**: `^[0-9]+(ns|us|µs|ms|s|m|h)$`
- **Description**: Timeout duration for remote operations (e.g., '30s', '5m')
- **CLI equivalent**: `--timeout`
- **Environment variable**: `TASK_REMOTE_TIMEOUT`

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
- **CLI equivalent**: `--expiry`
- **Environment variable**: `TASK_REMOTE_CACHE_EXPIRY`

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
- **Environment variable**: `TASK_REMOTE_CACHE_DIR`

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
- **Environment variable**: `TASK_REMOTE_TRUSTED_HOSTS` (comma-separated)

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

#### `cacert`

- **Type**: `string`
- **Default**: `""`
- **Description**: Path to a custom CA certificate file for TLS verification

```yaml
remote:
  cacert: "/path/to/ca.crt"
```

#### `cert`

- **Type**: `string`
- **Default**: `""`
- **Description**: Path to a client certificate file for mTLS authentication

```yaml
remote:
  cert: "/path/to/client.crt"
```

#### `cert-key`

- **Type**: `string`
- **Default**: `""`
- **Description**: Path to the client certificate private key file

```yaml
remote:
  cert-key: "/path/to/client.key"
```

#### `headers`

- **Type**: `map[string]map[string]string`
- **Default**: `{}` (empty map)
- **Description**: HTTP headers to send when fetching remote Taskfiles, organized
  by hostname. Header values support environment variable expansion using the
  templating system (e.g., <span v-pre>`{{.TOKEN}}`</span>). Headers are matched by exact hostname
  (including port if specified).
- **CLI equivalent**: `--header "host:Header-Name=value"` (can be repeated)

<div v-pre>

```yaml
remote:
  headers:
    raw.githubusercontent.com:
      Authorization: "Bearer {{.GITHUB_TOKEN}}"
    gitlab.com:
      PRIVATE-TOKEN: "{{.GITLAB_TOKEN}}"
    internal.company.com:8080:
      X-API-Key: "{{.API_KEY}}"
      X-Custom-Header: "custom-value"
```

</div>

This is particularly useful for authentication with services that use custom
header names (like GitLab's `PRIVATE-TOKEN`) or when you want to keep
credentials out of URLs. The headers are automatically applied to all HTTP
requests to the matching hostname.

Headers can also be specified via the `--header` flag for ad-hoc use. When both
CLI flags and config file headers are present, CLI flags take precedence for the
specific host and header name combination.

**Security notes:**

- Always use environment variable templates (e.g., <span v-pre>`{{.TOKEN}}`</span>) instead of
  hardcoding credentials in your config file
- Headers are only sent to exactly matching hostnames - headers for `github.com`
  will not be sent to `api.github.com` or subdomains
- Critical HTTP headers like `Host`, `Content-Length`, `Transfer-Encoding`, and
  `Connection` cannot be overridden for security reasons
- Always use HTTPS (not HTTP with `--insecure`) when sending authentication
  headers to prevent credentials from being transmitted in plaintext
