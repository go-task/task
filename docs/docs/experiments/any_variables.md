---
slug: /experiments/any-variables/
---

# Any Variables

- Issue: [#1415](https://github.com/go-task/task/issues/1415)
- Environment variable: `TASK_X_ANY_VARIABLES=1`
- Breaks:
  - Dynamically defined variables (using the `sh` keyword)

Currently, Task only supports string variables. This experiment allows you to
specify and use the following variable types:

- `string`
- `bool`
- `int`
- `float`
- `array`
- `map`

This allows you to have a lot more flexibility in how you use variables in
Task's templating engine. For example:

Evaluating booleans:

```yaml
version: 3

tasks:
  foo:
    vars:
      BOOL: false
    cmds:
      - '{{ if .BOOL }}echo foo{{ end}}'
```

Arithmetic:

```yaml
version: 3

tasks:
  foo:
    vars:
      INT: 10
      FLOAT: 3.14159
    cmds:
      - 'echo {{ add .INT .FLOAT }}'
```

Loops:

```yaml
version: 3

tasks:
  foo:
    vars:
      ARRAY: [1, 2, 3]
    cmds:
      - 'echo {{ range .ARRAY }}{{ . }}{{ end }}'
```

etc.

## Migration

Taskfiles with dynamically defined variables via the `sh` subkey will no longer
work with this experiment enabled. In order to keep using dynamically defined
variables, you will need to migrate your Taskfile to use the new syntax.

Previously, you might have defined a dynamic variable like this:

```yaml
version: 3

task:
  foo:
    vars:
      CALCULATED_VAR:
        sh: 'echo hello'
    cmds:
      - 'echo {{ .CALCULATED_VAR }}'
```

With this experiment enabled, you will need to remove the `sh` subkey and define
your command as a string that begins with a `$`. This will instruct Task to
interpret the string as a command instead of a literal value and the variable
will be populated with the output of the command. For example:

```yaml
version: 3

task:
  foo:
    vars:
      CALCULATED_VAR: '$echo hello'
    cmds:
      - 'echo {{ .CALCULATED_VAR }}'
```

If your current Taskfile contains a string variable that begins with a `$`, you
will now need to escape the `$` with a backslash (`\`) to stop Task from
executing it as a command.
