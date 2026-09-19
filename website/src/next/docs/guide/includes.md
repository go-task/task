---
title: Including Taskfiles
description:
  Reuse tasks across projects with `includes`, covering namespaces, optional and
  internal includes, flattening, and per-include variables.
section: Guide
docType: guide
outline: deep
---

# Including Taskfiles {#including-other-taskfiles}

Use `includes` to split a project into smaller Taskfiles or reuse tasks across
projects. Give each included file a namespace, then call its tasks through that
name.

Start with a Markdown file at `docs/README.md`, then create these two Taskfiles:

::: code-group

```yaml [Taskfile.yml]
version: '3'

includes:
  docs:
    taskfile: ./docs/Taskfile.yml
    dir: ./docs
```

```yaml [docs/Taskfile.yml]
version: '3'

tasks:
  build:
    cmds:
      - mkdir -p build
      - cp README.md build/index.md
```

:::

Run `task docs:build` to copy `docs/README.md` to `docs/build/index.md`. The
`docs` namespace comes from the include key, not the directory name. You can
also call `docs:build` from another task's `cmds` or `deps`.

An include path can point to a file or a directory. With `docs: ./docs`, Task
looks for a supported Taskfile name in that directory. Paths are relative to the
including file, and included Taskfiles must use the same schema version.

## Set the working directory {#directory-of-included-taskfile}

The `dir: ./docs` setting makes the included task read `docs/README.md` and
write into `docs/build/`, even when you run `task docs:build` from the project
root.

Without `dir`, included tasks use the including Taskfile's execution directory.
The same commands would look for `README.md` and create `build/` at the project
root. A Taskfile's location alone does not set its execution directory.

Individual tasks can also specify their own
[working directory](./defining-tasks.md#task-directory).

## Configure reusable tasks {#vars-of-included-taskfiles}

Use `vars` on an include to configure reusable tasks. This example includes the
same file twice with different image names:

::: code-group

```yaml [Taskfile.yml]
version: '3'

includes:
  backend:
    taskfile: ./taskfiles/Docker.yml
    vars:
      DOCKER_IMAGE: backend_image
  frontend:
    taskfile: ./taskfiles/Docker.yml
    vars:
      DOCKER_IMAGE: frontend_image
```

```yaml [taskfiles/Docker.yml]
version: '3'

vars:
  DOCKER_IMAGE: '{{.DOCKER_IMAGE | default "app"}}'

tasks:
  build:
    cmds:
      - echo 'Building {{.DOCKER_IMAGE}}'
```

:::

`task backend:build` prints `Building backend_image`; `task frontend:build`
prints `Building frontend_image`.

The template default in the included file is deliberate. A constant value in
that file's `vars` would override the value supplied by the include. See
[variable precedence](./variables.md#resolution-order) for the full rules.

## Shorten namespace names {#namespace-aliases}

Add namespace `aliases` to shorten frequently used task names:

```yaml
version: '3'

includes:
  documentation:
    taskfile: ./docs/Taskfile.yml
    dir: ./docs
    aliases: [docs]
```

Both `task documentation:build` and `task docs:build` call the same task.
Namespace aliases can be combined with
[task aliases](./defining-tasks.md#task-aliases).

## Allow a missing file {#optional-includes}

Use `optional: true` when a missing local Taskfile should not prevent the rest
of the project from running:

```yaml
version: '3'

includes:
  local:
    taskfile: ./Taskfile.local.yml
    optional: true

tasks:
  greet:
    cmds:
      - echo 'Hello, World!'
```

`task greet` works even without `Taskfile.local.yml`. Optional includes do not
make tasks from a missing file available.

## Keep included tasks internal {#internal-includes}

Use `internal: true` when all tasks in an included file are helpers for your
public tasks:

```yaml
version: '3'

includes:
  docs:
    taskfile: ./docs/Taskfile.yml
    dir: ./docs
    internal: true

tasks:
  build:
    cmds:
      - task: docs:build
```

With the `docs/Taskfile.yml` from the first example, `task build` works, but
`task docs:build` cannot be called directly. Internal tasks are also hidden from
task listings.

## Remove the namespace {#flatten-includes}

Use `flatten: true` to expose included tasks without a namespace:

```yaml
version: '3'

includes:
  docs:
    taskfile: ./docs/Taskfile.yml
    dir: ./docs
    flatten: true
```

With the first example's included file, call `task build` instead of
`task docs:build`. Calls within the Taskfile can also use the flattened name.

If the root Taskfile already defines `build`, this inclusion fails with a name
collision. Keep the namespace or exclude the conflicting included task.

## Exclude selected tasks {#exclude-tasks-from-being-included}

Use `excludes` to remove selected tasks from an include. Names match exactly;
append `:*` to exclude a namespace:

::: code-group

```yaml [Taskfile.yml]
version: '3'

includes:
  tools:
    taskfile: ./Tools.yml
    excludes: [debug, 'internal:*']
```

```yaml [Tools.yml]
version: '3'

tasks:
  check: echo 'Checking project'
  debug: echo 'Debug details'
  internal:setup: echo 'Internal setup'
```

:::

`task tools:check` works. `task tools:debug` and `task tools:internal:setup`
fail because those tasks were excluded. `excludes` also works with `flatten`.

## Choose a file by OS {#os-specific-taskfiles}

Use the `OS` template function to select a file for the current platform:

```yaml
version: '3'

includes:
  build: ./Taskfile_{{OS}}.yml
```

Provide files such as `Taskfile_linux.yml`, `Taskfile_darwin.yml`, and
`Taskfile_windows.yml` for the platforms you support. If only a few commands
differ,
[platform restrictions](./platforms.md#platform-specific-tasks-and-commands) may
be easier to maintain.

::: tip Related guide

<span id="remote-taskfiles"></span>

Replace the local path with an HTTP or Git location to reuse tasks maintained
outside the project. Only run remote Taskfiles from sources you trust; they
execute commands on your machine. See [Remote Taskfiles](../remote-taskfiles.md)
for a complete example, checksum pinning, and offline use.

:::
