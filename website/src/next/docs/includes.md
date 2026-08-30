---
title: Including other Taskfiles
description:
  Reuse tasks across projects with `includes` — namespaces, optional and
  internal includes, flattening, and per-include variables.
outline: deep
---

# Including other Taskfiles

If you want to share tasks between different projects (Taskfiles), you can use
the importing mechanism to include other Taskfiles using the `includes` keyword:

```yaml
version: '3'

includes:
  docs: ./documentation # will look for ./documentation/Taskfile.yml
  docker: ./DockerTasks.yml
```

The tasks described in the given Taskfiles will be available with the informed
namespace. So, you'd call `task docs:serve` to run the `serve` task from
`documentation/Taskfile.yml` or `task docker:build` to run the `build` task from
the `DockerTasks.yml` file.

Relative paths are resolved relative to the directory containing the including
Taskfile.

## Remote Taskfiles

::: danger

Never run remote Taskfiles from sources that you do not trust.

:::

It is possible to include a Taskfile from a remote source via HTTP(S) or Git.
This is useful if you want to reuse a set of tasks in multiple projects. For
more information, take a look at our
[remote Taskfiles documentation](./remote-taskfiles.md).

```yaml
version: '3'

includes:
  my-remote-namespace: https://raw.githubusercontent.com/go-task/task/main/website/src/public/Taskfile.yml
```

## OS-specific Taskfiles

You can include OS-specific Taskfiles by using a templating function:

```yaml
version: '3'

includes:
  build: ./Taskfile_{{OS}}.yml
```

## Directory of included Taskfile

By default, included Taskfile's tasks are run in the current directory, even if
the Taskfile is in another directory, but you can force its tasks to run in
another directory by using this alternative syntax:

```yaml
version: '3'

includes:
  docs:
    taskfile: ./docs/Taskfile.yml
    dir: ./docs
```

::: info

The included Taskfiles must be using the same schema version as the main
Taskfile uses.

:::

## Optional includes

Includes marked as optional will allow Task to continue execution as normal if
the included file is missing.

```yaml
version: '3'

includes:
  tests:
    taskfile: ./tests/Taskfile.yml
    optional: true

tasks:
  greet:
    cmds:
      - echo "This command can still be successfully executed if
        ./tests/Taskfile.yml does not exist"
```

## Internal includes

Includes marked as internal will set all the tasks of the included file to be
internal as well (see [Internal tasks](./defining-tasks.md#internal-tasks)).
This is useful when including utility tasks that are not intended to be used
directly by the user.

```yaml
version: '3'

includes:
  tests:
    taskfile: ./taskfiles/Utils.yml
    internal: true
```

## Flatten includes

You can flatten the included Taskfile tasks into the main Taskfile by using the
`flatten` option. It means that the included Taskfile tasks will be available
without the namespace.

::: code-group

```yaml [Taskfile.yml]
version: '3'

includes:
  lib:
    taskfile: ./Included.yml
    flatten: true

tasks:
  greet:
    cmds:
      - echo "Greet"
      - task: foo
```

```yaml [Included.yml]
version: '3'

tasks:
  foo:
    cmds:
      - echo "Foo"
```

:::

If you run `task -a` it will print :

```sh
task: Available tasks for this project:
* greet:
* foo
```

You can run `task foo` directly without the namespace.

You can also reference the task in other tasks without the namespace. So if you
run `task greet` it will run `greet` and `foo` tasks and the output will be :

```text
Greet
Foo
```

If multiple tasks have the same name, an error will be thrown:

::: code-group

```yaml [Taskfile.yml]
version: '3'
includes:
  lib:
    taskfile: ./Included.yml
    flatten: true

tasks:
  greet:
    cmds:
      - echo "Greet"
      - task: foo
```

```yaml [Included.yml]
version: '3'

tasks:
  greet:
    cmds:
      - echo "Foo"
```

:::

If you run `task -a` it will print:

```text
task: Found multiple tasks (greet) included by "lib"
```

If the included Taskfile has a task with the same name as a task in the main
Taskfile, you may want to exclude it from the flattened tasks.

You can do this by using the
[`excludes` option](#exclude-tasks-from-being-included).

## Exclude tasks from being included

You can exclude tasks or entire namespaces from being included by using the
`excludes` option. This option takes the list of tasks or namespaces to be
excluded from this include. Task names are matched exactly. To exclude a
namespace, append `:*` to its name.

::: code-group

```yaml [Taskfile.yml]
version: '3'

includes:
  included:
    taskfile: ./Included.yml
    excludes: [foo, 'internal:*', 'debug:*']
```

```yaml [Included.yml]
version: '3'

tasks:
  foo: echo "Foo"
  bar: echo "Bar"
  internal:setup: echo "Internal setup"
  debug:status: echo "Debug status"
```

:::

`task included:foo`, `task included:internal:setup`, and
`task included:debug:status` will throw errors because they are excluded, but
`task included:bar` will work and display `Bar`.

It's compatible with the `flatten` option.

## Vars of included Taskfiles

You can also specify variables when including a Taskfile. This may be useful for
having a reusable Taskfile that can be tweaked or even included more than once:

```yaml
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

## Namespace aliases

When including a Taskfile, you can give the namespace a list of `aliases`. This
works in the same way as [task aliases](./defining-tasks.md#task-aliases) and
can be used together to create shorter and easier-to-type commands.

```yaml
version: '3'

includes:
  generate:
    taskfile: ./taskfiles/Generate.yml
    aliases: [gen]
```

::: info

Vars declared in the included Taskfile have preference over the variables in the
including Taskfile! If you want a variable in an included Taskfile to be
overridable, use the
[default function](https://sprig.taskfile.dev/defaults.html):
<span v-pre>`MY_VAR: '{{.MY_VAR | default "my-default-value"}}'`</span>.

:::
