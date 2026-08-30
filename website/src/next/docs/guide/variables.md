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

Variables can be set in many places in a Taskfile, and when the same name is set
twice, one of them wins. The order is the same everywhere and it is described
once, in [Variable resolution](../concepts/variable-resolution.md#the-order),
including the two cases that surprise people most: a task's own `vars:` cannot
be overridden from the command line, and `vars:` given on an `includes:` entry
act as defaults rather than overrides.

Example of sending parameters with environment variables:

```shell
$ TASK_VARIABLE=a-value task do-something
```

::: tip

A special variable `.TASK` is always available containing the task name.

:::

Since some shells do not support the above syntax to set environment variables
(Windows) tasks also accept a similar style when not at the beginning of the
command.

```shell
$ task write-file FILE=file.txt "CONTENT=Hello, World!" print "MESSAGE=All done!"
```

Example of locally declared vars:

```yaml
version: '3'

tasks:
  print-var:
    cmds:
      - echo "{{.VAR}}"
    vars:
      VAR: Hello!
```

Example of global vars in a `Taskfile.yml`:

```yaml
version: '3'

vars:
  GREETING: Hello from Taskfile!

tasks:
  greet:
    cmds:
      - echo "{{.GREETING}}"
```

Example of a `default` value to be overridden from CLI:

```yaml
version: '3'

tasks:
  greet_user:
    desc: 'Greet the user with a name.'
    vars:
      USER_NAME: '{{.USER_NAME| default "DefaultUser"}}'
    cmds:
      - echo "Hello, {{.USER_NAME}}!"
```

```shell
$ task greet_user
task: [greet_user] echo "Hello, DefaultUser!"
Hello, DefaultUser!
$ task greet_user USER_NAME="Bob"
task: [greet_user] echo "Hello, Bob!"
Hello, Bob!
```

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
      - curl -H "Authorization: {{.API_KEY}}" api.example.com
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
