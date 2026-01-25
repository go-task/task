---
title: 'Scoped Taskfiles (#2035)'
description:
  Experiment for variable isolation and env namespace in included Taskfiles
outline: deep
---

# Scoped Taskfiles (#2035)

::: warning

All experimental features are subject to breaking changes and/or removal _at any
time_. We strongly recommend that you do not use these features in a production
environment. They are intended for testing and feedback only.

:::

::: danger

This experiment breaks the following functionality:

- **Environment variables are no longer available at root level in templates**
  - Before: <span v-pre>`{{.PATH}}`</span>, <span v-pre>`{{.MY_ENV}}`</span>
  - After: <span v-pre>`{{.env.PATH}}`</span>,
    <span v-pre>`{{.env.MY_ENV}}`</span>
- **Variables from sibling includes are no longer visible**
  - Include A cannot access variables defined in Include B
  - Each include only sees: root vars + its own vars + parent vars

:::

::: info

To enable this experiment, set the environment variable:
`TASK_X_SCOPED_TASKFILES=1`. Check out
[our guide to enabling experiments](./index.md#enabling-experiments) for more
information.

:::

This experiment introduces two major changes to how variables work in Task:

1. **Environment namespace**: Environment variables (both OS and Taskfile `env:`
   sections) are moved to a dedicated <span v-pre>`{{.env.XXX}}`</span>
   namespace, separating them from regular variables
2. **Variable scoping**: Variables defined in included Taskfiles are isolated -
   sibling includes cannot see each other's variables

## Environment Namespace

With this experiment enabled, environment variables are no longer mixed with
regular variables at the template root level. Instead, they are accessible
through the <span v-pre>`{{.env.XXX}}`</span> namespace.

### Comparison Table

| Template                                        | Legacy | SCOPED_TASKFILES          |
| ----------------------------------------------- | ------ | ------------------------- |
| <span v-pre>`{{.MY_VAR}}`</span> (from `vars:`) | Works  | Works                     |
| <span v-pre>`{{.MY_ENV}}`</span> (from `env:`)  | Works  | `<no value>`              |
| <span v-pre>`{{.env.MY_ENV}}`</span>            | -      | Works                     |
| <span v-pre>`{{.PATH}}`</span> (OS)             | Works  | `<no value>`              |
| <span v-pre>`{{.env.PATH}}`</span> (OS)         | -      | Works                     |
| <span v-pre>`{{.TASK}}`</span> (special)        | Works  | Works (stays at root)     |

### Example

```yaml
version: '3'

env:
  DB_HOST: localhost

vars:
  DB_NAME: mydb

tasks:
  show:
    cmds:
      # Access Taskfile env: section
      - echo "Host: {{.env.DB_HOST}}"

      # Access regular vars (unchanged)
      - echo "Name: {{.DB_NAME}}"

      # Access OS environment variables
      - echo "Path: {{.env.PATH}}"

      # Special variables stay at root level
      - echo "Task: {{.TASK}}"
```

## Variable Scoping

Variables defined in included Taskfiles are now isolated from each other.
Sibling includes cannot access each other's variables, but child includes can
still inherit variables from their parent.

### Example

::: code-group

```yaml [Taskfile.yml]
version: '3'

vars:
  ROOT_VAR: from_root

includes:
  api: ./api
  web: ./web
```

```yaml [api/Taskfile.yml]
version: '3'

vars:
  API_VAR: from_api

tasks:
  show:
    cmds:
      # Inherited from root - works
      - echo "ROOT_VAR={{.ROOT_VAR}}"

      # Own variable - works
      - echo "API_VAR={{.API_VAR}}"

      # From sibling include - NOT visible
      - echo "WEB_VAR={{.WEB_VAR}}"
```

```yaml [web/Taskfile.yml]
version: '3'

vars:
  WEB_VAR: from_web

tasks:
  show:
    cmds:
      # Inherited from root - works
      - echo "ROOT_VAR={{.ROOT_VAR}}"

      # Own variable - works
      - echo "WEB_VAR={{.WEB_VAR}}"

      # From sibling include - NOT visible
      - echo "API_VAR={{.API_VAR}}"
```

:::

## Variable Priority

With this experiment, variables follow a clear priority order (lowest to
highest):

| Priority | Source                   | Description                              |
| -------- | ------------------------ | ---------------------------------------- |
| 1        | Root Taskfile vars       | `vars:` in the root Taskfile             |
| 2        | Include Taskfile vars    | `vars:` in the included Taskfile         |
| 3        | Include passthrough vars | `includes: name: vars:` from parent      |
| 4        | Task vars                | `tasks: name: vars:` in the task         |
| 5        | Call vars                | `task: name` with `vars:` when calling   |
| 6        | CLI vars                 | `task foo VAR=value` on command line     |

### Example: Call vars override task vars

```yaml
version: '3'

tasks:
  greet:
    vars:
      NAME: default
    cmds:
      - echo "Hello {{.NAME}}"

  caller:
    cmds:
      - task: greet
        vars:
          NAME: from_caller
```

```bash
# Direct call uses task default
task greet
# Output: Hello default

# Call vars override task vars
task caller
# Output: Hello from_caller

# CLI vars override everything
task greet NAME=cli
# Output: Hello cli
```

## Migration Guide

To migrate your Taskfiles to use this experiment:

1. **Update environment variable references** in your templates:

   - <span v-pre>`{{.PATH}}`</span> becomes
     <span v-pre>`{{.env.PATH}}`</span>
   - <span v-pre>`{{.HOME}}`</span> becomes
     <span v-pre>`{{.env.HOME}}`</span>
   - <span v-pre>`{{.MY_TASKFILE_ENV}}`</span> becomes
     <span v-pre>`{{.env.MY_TASKFILE_ENV}}`</span>

2. **Variables in `vars:` sections remain unchanged**:

   - <span v-pre>`{{.MY_VAR}}`</span> still works the same way

3. **Special variables stay at root level**:

   - <span v-pre>`{{.TASK}}`</span>, <span v-pre>`{{.ROOT_DIR}}`</span>,
     <span v-pre>`{{.TASKFILE}}`</span>, <span v-pre>`{{.TASKFILE_DIR}}`</span>,
     etc.

4. **Review cross-include variable dependencies**:
   - If your included Taskfiles rely on variables from sibling includes, you'll
     need to either move those variables to the root Taskfile or pass them
     explicitly via the `vars:` attribute in the `includes:` section.

5. **Use `flatten: true` for gradual migration**:
   - If an include needs the legacy behavior (access to sibling variables), you
     can use `flatten: true` on that include as an escape hatch.

## Using `flatten: true`

The `flatten: true` option on includes bypasses scoping for that specific
include. When an include has `flatten: true`:

- Its variables are merged globally (legacy behavior)
- It can access variables from sibling includes
- Sibling includes can access its variables

This is useful for gradual migration or when you have includes that genuinely
need to share variables.

### Example

```yaml
version: '3'

vars:
  ROOT_VAR: from_root

includes:
  # Scoped include - isolated from siblings
  api:
    taskfile: ./api

  # Flattened include - uses legacy merge behavior
  shared:
    taskfile: ./shared
    flatten: true

  # Another scoped include
  web:
    taskfile: ./web
```

In this example:

- `api` and `web` are isolated from each other (cannot see each other's vars)
- `shared` uses legacy behavior: its vars are merged globally
- Both `api` and `web` can access variables from `shared`
- `shared` can access variables from `api` and `web`

::: tip

Use `flatten: true` sparingly. The goal of scoped taskfiles is to improve
isolation and predictability. Flattening should be a temporary measure during
migration or for utility includes that genuinely need global scope.

:::
