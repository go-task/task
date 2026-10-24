---
title: Running tasks
description:
  How Task finds a Taskfile, and how to run one from a subdirectory, from your
  home directory, from standard input or as a dry run.
section: Guide
docType: guide
outline: deep
---

# Running tasks

Run `task <name>` from your project to execute a task. With no name, Task runs
the `default` task. Start by listing tasks, then run one:

```shell
task --list
task build
```

`--list` shows tasks with descriptions. Use `--list-all` to include tasks
without descriptions; [internal tasks](./defining-tasks.md#internal-tasks)
remain hidden. See [Defining tasks](./defining-tasks.md) to create these
entries.

Pass several names, such as `task lint test`, to run them in order. Use
[`--parallel`](./dependencies.md#limiting-how-much-runs-at-once) for independent
tasks that can run concurrently. To pass values or command arguments, see
[Command-line arguments](./arguments.md).

## Preview commands {#dry-run-mode}

Use `--dry` (or `-n`) to print the commands Task would run:

```shell
task --dry build
```

Task still resolves templates and dynamic variables while preparing the
commands. A variable declared with `sh` can therefore execute its shell command
during a dry run. See [Variables](./variables.md#when-values-are-computed).

## Run from a subdirectory {#running-a-taskfile-from-a-subdirectory}

If Task cannot find a Taskfile in the current directory, it searches parent
directories. Tasks run from the directory containing that Taskfile unless
[their `dir` setting](./defining-tasks.md#task-directory) changes it.

Use <span v-pre>`{{.USER_WORKING_DIR}}`</span> when the task should act on the
directory where you invoked it. In a monorepo, this lets each service use the
same task to start its Docker Compose services:

```yaml
version: '3'

tasks:
  up:
    dir: '{{.USER_WORKING_DIR}}'
    preconditions:
      - test -f docker-compose.yml
    cmds:
      - docker compose up -d
```

From a service directory containing `docker-compose.yml`, run `task up`. The
root Taskfile supplies the task; the service directory supplies its files.

## Use a personal Taskfile {#running-a-global-taskfile}

Store a Taskfile in your home directory and use `--global` (or `-g`) to call it
from anywhere:

```shell
task --global from-working-directory
```

Global tasks run in your home directory by default. Set `dir` to
<span v-pre>`{{.USER_WORKING_DIR}}`</span> when a personal utility should work
on the current project:

```yaml
version: '3'

tasks:
  from-home:
    cmds:
      - pwd

  from-working-directory:
    dir: '{{.USER_WORKING_DIR}}'
    cmds:
      - pwd
```

## Read a Taskfile from stdin {#running-a-taskfile-from-stdin}

Use `--taskfile -` (or `-t -`) to read a generated Taskfile from standard input.
For example, to read an existing file through stdin:

```shell
task --taskfile - < ./Taskfile.yml
```

You can also pipe the output of a Taskfile generator into `task --taskfile -`.
With no task name, these commands run the generated file's `default` task.

## Run a terminal app {#interactive-cli-application}

Set `interactive: true` on a task that launches a terminal application:

```yaml
version: '3'

tasks:
  edit:
    interactive: true
    cmds:
      - vim my-file.txt
```

Run `task edit` to open the editor. Interactive applications need direct access
to the terminal; buffered or prefixed [output modes](./output.md#output-syntax)
and other tasks running concurrently can interfere with their input and display.

This task setting is distinct from the CLI `--interactive` option, which
[prompts for missing variables](./required-variables.md#prompting-for-missing-variables-interactively).

## Choose a file name {#supported-file-names}

Use `--taskfile` to choose a specific file or `--dir` to run in another project:

```shell
task --taskfile Taskfile.ci.yml build
task --dir ./backend build
```

Task looks for these names in order of priority:

- `Taskfile.yml`
- `taskfile.yml`
- `Taskfile.yaml`
- `taskfile.yaml`
- `Taskfile.dist.yml`
- `taskfile.dist.yml`
- `Taskfile.dist.yaml`
- `taskfile.dist.yaml`

Commit a `.dist` variant if users need to supply their own complete
`Taskfile.yml`. The local file takes priority; add it to `.gitignore` if it
should remain personal. To customize only some variables or tasks, use a local
override instead.

## Customize a Taskfile locally {#local-overrides}

When Task discovers the root Taskfile automatically, it also loads an optional
`Taskfile.override.yml` from the same directory. Keep the shared configuration
in the main file and put personal changes in the override:

::: code-group

```yaml [Taskfile.yml]
version: '3'

vars:
  PORT: 3000

tasks:
  dev:
    cmds:
      - echo 'PORT={{.PORT}}'
```

```yaml [Taskfile.override.yml]
version: '3'

vars:
  PORT: 4000
```

:::

`task dev` prints `PORT=4000`. Add `Taskfile.override.yml` to `.gitignore` to
keep it personal.

Besides `version`, which must match the main Taskfile, an override accepts:

- `vars` and `env`, merged by key with the override winning. An overridden
  variable keeps its position in the main Taskfile, so it can only reference
  variables declared before it.
- `tasks`, which adds new tasks or replaces existing ones whole: a replaced task
  keeps nothing from the original. To change a single value of a task, move it
  into a variable in the main Taskfile and override that variable.

Any other root key, such as `includes` or `dotenv`, is an error.

`taskfile.override.yml`, `Taskfile.override.yaml` and `taskfile.override.yaml`
are also recognized; only the first one present is loaded. Overrides apply
before includes are resolved, so a variable can change which file is included,
but an override cannot redefine an included task such as `docs:build`.
Included, remote and stdin Taskfiles never load an override.

Naming the file with `--taskfile` skips the override:

```shell
task -t Taskfile.yml dev # prints PORT=3000
```
