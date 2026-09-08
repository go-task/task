---
title: Variables
description:
  Static, dynamic, map and secret variables, how they are scoped, and how they
  reference each other.
section: Guide
docType: guide
outline: deep
---

# Variables

Declare variables with `vars`, then read them in a command with a template:

```yaml
version: '3'

vars:
  NAME: World

tasks:
  greet:
    cmds:
      - echo "Hello, {{.NAME}}!"
```

Run `task greet` to print `Hello, World!`, or `task greet NAME=Bob` to print
`Hello, Bob!`. This example puts the default at the root of the Taskfile so
callers can override it.

Jump to [command line values](#command-line-values),
[resolution order](#resolution-order), [dynamic variables](#dynamic-variables),
[variable types](#variable-types) or [secrets](#secret-variables).

## Command line values

Pass values after the task name. This works across shells, including Windows:

```shell
task greet NAME=Bob
```

Command line values are shared by all tasks in the invocation. Quote values
containing spaces:

```shell
task greet "NAME=Jane Doe"
```

On shells that support it, you can also supply an environment variable:

```shell
TASK_VARIABLE=a-value task do-something
```

Variables can be declared at the root of the Taskfile, on an include or call, or
on a task itself. When the same name appears in several places, the
[resolution order](#resolution-order) determines which value wins.

::: tip

The special variable `.TASK` contains the name of the task being run.

:::

## Resolution order

Applied first to last. Later wins.

| #   | Source                              | Set by                                                                        |
| --- | ----------------------------------- | ----------------------------------------------------------------------------- |
| 1   | The process environment             | the shell that ran `task`                                                     |
| 2   | Special variables                   | Task itself (`TASK`, `ROOT_DIR`, `CLI_ARGS`, …)                               |
| 3   | Taskfile `env:`                     | the `env:` block; `dotenv:` files fill only names `env:` does not already set |
| 4   | Global `vars:`                      | the `vars:` block of every Taskfile in the run                                |
| 5   | Include `vars:`                     | the `vars:` given on an `includes:` entry                                     |
| 6   | The included Taskfile's own `vars:` | the `vars:` block of the file being included                                  |
| 7   | Call variables                      | `task foo BAR=1`, or `vars:` on a `task:` command                             |
| 8   | The task's `vars:`                  | the `vars:` block of the task being run                                       |

## What this means in practice

### Task variables and CLI overrides {#a-task-s-own-variables-cannot-be-overridden-from-the-command-line}

Step 8 comes after step 7, so a variable declared on the task always wins:

```yaml
version: '3'

tasks:
  greet:
    vars:
      NAME: from-task
    cmds:
      - echo "{{.NAME}}"
```

```shell
$ task greet NAME=from-cli
from-task
```

To let a caller supply a value, give the default somewhere earlier, in global
`vars:`, or use a template default:

```yaml
version: '3'

vars:
  NAME: from-global

tasks:
  greet:
    cmds:
      - echo "{{.NAME}}"
```

```shell
$ task greet NAME=from-cli
from-cli
```

To keep the default on the task itself, read the caller's value in a template:

```yaml
vars:
  NAME: '{{.NAME | default "World"}}'
```

### Let callers configure an included Taskfile

Step 6 comes after step 5, so a constant declared in the included Taskfile's
`vars:` overrides the value supplied by `includes.vars`.

To keep a configurable default in the included Taskfile, use a template that
reads the previously resolved value:

```yaml
version: '3'

vars:
  DOCKER_IMAGE: '{{.DOCKER_IMAGE | default "app"}}'

tasks:
  build:
    cmds:
      - echo "building {{.DOCKER_IMAGE}}"
```

An inclusion that supplies `DOCKER_IMAGE: backend_image` now prints
`building backend_image`; without a supplied value, it prints `building app`.
The default stays in one place and applies to every task in the included file.

### Shared global names {#global-variable-names-are-shared-across-every-taskfile-in-the-run}

Global `vars:` are merged into one set before any task runs, so a name declared
in both the entrypoint and an included Taskfile resolves to the included one,
including for tasks defined in the entrypoint.

Give globals that belong to an included Taskfile a distinctive name, or move
them onto the tasks that use them, where step 8 keeps them local.

### `env:` and `vars:` are not the same thing

A `vars:` entry is not exported to commands in `cmds`, whichever level it was
declared at. Read it with a template instead of `$FOO`. Dynamic variables
(`sh:`) have a different environment, described below.

`env:` is exported to the environment of the commands Task runs, so `$FOO`
works. Whether a template also sees it depends on where it was declared:

| Declared at              | <span v-pre>`{{.FOO}}`</span>              | `$FOO` |
| ------------------------ | ------------------------------------------ | ------ |
| the root of the Taskfile | yes, it is step 3 above                    | yes    |
| on a task                | keeps the value from other sources, if any | yes    |

A task's `env:` is assembled after the variable set has been resolved, so it
never takes part in the order on this page. If global `env:` sets `FOO: root`
and a task's `env:` sets `FOO: task`, the template <span v-pre>`{{.FOO}}`</span>
renders `root`, while `$FOO` in a command reads `task` (assuming the process
environment does not already set `FOO`). The template only renders empty if no
other source defines the variable. Read a task-level environment value with
`$FOO`, or declare it in `vars:` if a template needs it.

## When values are computed

Dynamic variables (`sh:`) are executed while the set is being built, in the
order above. Within a block, variables are resolved in declaration order, so a
`sh:` command can also reference variables declared earlier in the same block:

```yaml
vars:
  INPUT: hello
  OUTPUT:
    sh: echo {{.INPUT}}
```

Here `OUTPUT` resolves to `hello`. Variables from later declarations or later
steps are not available yet.

The shell used by `sh:` also receives previously resolved scalar variables, so
`sh: echo $INPUT` works in this example too. The process environment takes
precedence for shell lookups unless the
[Env Precedence experiment](../experiments/env-precedence.md) is enabled.
Commands in `cmds` only receive the process environment and Task's `env:` and
`dotenv:` values; they do not inherit `vars:` this way.

Results are cached for the run, keyed on the command string, so the same `sh:`
command appearing twice runs once.

To pass a variable without flattening it to text, an array or a map, use `ref:`
instead of <span v-pre>`{{ }}`</span>. A template renders a string; `ref:`
preserves the type.

## Dynamic variables

The below syntax (`sh:` prop in a variable) is considered a dynamic variable.
The value will be treated as a command and the output assigned. If there are one
or more trailing newlines, the last newline will be trimmed.

```yaml
version: '3'

tasks:
  build:
    cmds:
      - go build -ldflags="-X main.Version={{.GIT_COMMIT}}" main.go
    vars:
      GIT_COMMIT:
        sh: git log -n 1 --format=%h
```

This works for all types of variables.

## Variable types

Task allows you to set variables using the `vars` keyword. The following
variable types are supported:

- `string`
- `bool`
- `int`
- `float`
- `array`
- `map`

::: info

Defining a map requires that you use a special `map` subkey (see example below).

:::

```yaml
version: 3

tasks:
  foo:
    vars:
      STRING: 'Hello, World!'
      BOOL: true
      INT: 42
      FLOAT: 3.14
      ARRAY: [1, 2, 3]
      MAP:
        map: { A: 1, B: 2, C: 3 }
    cmds:
      - 'echo {{.STRING}}' # Hello, World!
      - 'echo {{.BOOL}}' # true
      - 'echo {{.INT}}' # 42
      - 'echo {{.FLOAT}}' # 3.14
      - 'echo {{.ARRAY}}' # [1 2 3]
      - 'echo {{index .ARRAY 0}}' # 1
      - 'echo {{.MAP}}' # map[A:1 B:2 C:3]
      - 'echo {{.MAP.A}}' # 1
```

## Referencing other variables

Templating is great for referencing string values if you want to pass a value
from one task to another. However, the templating engine is only able to output
strings. If you want to pass something other than a string to another task then
you will need to use a reference (`ref`) instead.

::: code-group

```yaml [Templating Engine]
version: 3

tasks:
  foo:
    vars:
      FOO: [A, B, C] # <-- FOO is defined as an array
    cmds:
      - task: bar
        vars:
          FOO: '{{.FOO}}' # <-- FOO gets converted to a string when passed to bar
  bar:
    cmds:
      - 'echo {{index .FOO 0}}' # <-- FOO is a string so the task outputs '91' which is the ASCII code for '[' instead of the expected 'A'
```

```yaml [Reference]
version: 3

tasks:
  foo:
    vars:
      FOO: [A, B, C] # <-- FOO is defined as an array
    cmds:
      - task: bar
        vars:
          FOO:
            ref: .FOO # <-- FOO gets passed by reference to bar and maintains its type
  bar:
    cmds:
      - 'echo {{index .FOO 0}}' # <-- FOO is still a map so the task outputs 'A' as expected
```

:::

This also works the same way when calling `deps` and when defining a variable
and can be used in any combination:

```yaml
version: 3

tasks:
  foo:
    vars:
      FOO: [A, B, C] # <-- FOO is defined as an array
      BAR:
        ref: .FOO # <-- BAR is defined as a reference to FOO
    deps:
      - task: bar
        vars:
          BAR:
            ref: .BAR # <-- BAR gets passed by reference to bar and maintains its type
  bar:
    cmds:
      - 'echo {{index .BAR 0}}' # <-- BAR still refers to FOO so the task outputs 'A'
```

All references use the same templating syntax as regular templates, so in
addition to calling `.FOO`, you can also pass subkeys (`.FOO.BAR`) or indexes
(`index .FOO 0`) and use functions (`len .FOO`) as described in the
[templating-reference][templating-reference]:

```yaml
version: 3

tasks:
  foo:
    vars:
      FOO: [A, B, C] # <-- FOO is defined as an array
    cmds:
      - task: bar
        vars:
          FOO:
            ref: index .FOO 0 # <-- The element at index 0 is passed by reference to bar
  bar:
    cmds:
      - 'echo {{.FOO}}' # <-- FOO is just the letter 'A'
```

## Parsing JSON/YAML into map variables

If you have a raw JSON or YAML string that you want to process in Task, you can
use a combination of the `ref` keyword and the `fromJson` or `fromYaml`
templating functions to parse the string into a map variable. For example:

```yaml
version: '3'

tasks:
  task-with-map:
    vars:
      JSON: '{"a": 1, "b": 2, "c": 3}'
      FOO:
        ref: 'fromJson .JSON'
    cmds:
      - echo {{.FOO}}
```

```txt
map[a:1 b:2 c:3]
```

## Secret variables

Task supports marking variables as `secret` to prevent their values from being
displayed in command logs. When a variable is marked as secret, its value will
be replaced with `*****` in the task output logs.

::: warning

**Security Notice**: This feature helps prevent accidental exposure of secrets
in logs, but is **not a substitute** for proper secret management practices.

**What this protects:**

- ✅ Secret values in console/terminal logs
- ✅ Secret values in CI/CD logs
- ✅ Accidental copy-paste of logs containing secrets

**What this does NOT protect:**

- ❌ Secrets visible in process inspection (e.g., `ps aux`)
- ❌ Secrets in shell history
- ❌ Secrets in command output (stdout/stderr)
- ❌ Secret values copied into derived (non-secret) variables

Always use proper secret management tools (HashiCorp Vault, AWS Secrets Manager,
etc.) for production environments.

:::

To mark a variable as secret, add `secret: true` to the variable definition:

```yaml
version: '3'

vars:
  API_KEY:
    value: 'sk-1234567890abcdef'
    secret: true

tasks:
  deploy:
    cmds:
      - 'curl -H "Authorization: {{.API_KEY}}" api.example.com'
      # Logged as: task: [deploy] curl -H "Authorization: *****" api.example.com
```

Secret variables work with all variable types:

::: code-group

```yaml [Simple Value]
version: '3'

vars:
  PASSWORD:
    value: 'my-secret-password'
    secret: true

tasks:
  connect:
    cmds:
      - psql -U user -p {{.PASSWORD}} mydb
      # Logged as: psql -U user -p ***** mydb
```

```yaml [Shell Command]
version: '3'

vars:
  DB_PASSWORD:
    sh: vault read -field=password secret/db
    secret: true

tasks:
  migrate:
    cmds:
      - psql -U admin -p {{.DB_PASSWORD}} mydb
      # Password from vault is masked in logs
```

```yaml [Task-Level Secret]
version: '3'

vars:
  PUBLIC_URL: https://example.com

tasks:
  deploy:
    vars:
      DEPLOY_TOKEN:
        value: 'secret-token-123'
        secret: true
    cmds:
      - echo "Deploying to {{.PUBLIC_URL}} with token {{.DEPLOY_TOKEN}}"
      # Logged as: echo "Deploying to https://example.com with token *****"
```

:::

Multiple secrets in the same command are all masked:

```yaml
version: '3'

vars:
  API_KEY:
    value: 'api-key-123'
    secret: true
  PASSWORD:
    value: 'password-456'
    secret: true

tasks:
  setup:
    cmds:
      - ./setup.sh --api {{.API_KEY}} --pwd {{.PASSWORD}}
      # Logged as: ./setup.sh --api ***** --pwd *****
```

::: tip

**Best practices for secret variables:**

1. **Use shell commands to load secrets**, not hardcoded values:

   ```yaml
   # ❌ BAD - Secret visible in Taskfile
   vars:
     API_KEY:
       value: 'hardcoded-secret'
       secret: true

   # ✅ GOOD - Secret loaded from external source
   vars:
     API_KEY:
       sh: vault kv get -field=api_key secret/myapp
       secret: true
   ```

2. **Combine with environment variables:**

   ```yaml
   vars:
     API_KEY:
       sh: echo $MY_API_KEY
       secret: true
   ```

3. **Use .gitignore for secret files:**

   If you use dotenv files, add them to `.gitignore`:

   ```yaml
   dotenv: ['.env.local'] # Load from .env.local (in .gitignore)
   ```

:::

::: warning

**Secrets are not propagated to derived variables.** The `secret` flag only
masks the variable it is set on. A non-secret variable that references a secret
will expose the resolved value in logs:

```yaml
version: '3'

vars:
  API_KEY:
    value: 'secret-api-key-123'
    secret: true
  HEADER:
    value: 'Bearer {{.API_KEY}}' # ❌ not marked as secret

tasks:
  call:
    cmds:
      - curl -H "{{.HEADER}}" api.example.com
      # Logged as: curl -H "Bearer secret-api-key-123" api.example.com (LEAK)
```

Mark every variable that carries a secret value as `secret: true`:

```yaml
vars:
  HEADER:
    value: 'Bearer {{.API_KEY}}'
    secret: true # ✅ masked
```

:::

[templating-reference]: ../reference/templating.md
