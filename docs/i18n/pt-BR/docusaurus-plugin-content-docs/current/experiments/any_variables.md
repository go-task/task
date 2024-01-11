---
slug: /experiments/any-variables/
---

# Any Variables (#1415)

:::caution

All experimental features are subject to breaking changes and/or removal _at any
time_. We strongly recommend that you do not use these features in a production
environment. They are intended for testing and feedback only.

:::

:::warning

This experiment breaks the following functionality:

- Dynamically defined variables (using the `sh` keyword)

:::

:::info

To enable this experiment, set the environment variable:
`TASK_X_ANY_VARIABLES=1`. Check out [our guide to enabling experiments
][enabling-experiments] for more information.

:::

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
      - '{{if .BOOL}}echo foo{{end}}'
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
      - 'echo {{add .INT .FLOAT}}'
```

Ranging:

```yaml
version: 3

tasks:
  foo:
    vars:
      ARRAY: [1, 2, 3]
    cmds:
      - 'echo {{range .ARRAY}}{{.}}{{end}}'
```

There are many more templating functions which can be used with the new types of
variables. For a full list, see the [slim-sprig][slim-sprig] documentation.

## Looping over variables

Previously, you would have to use a delimiter separated string to loop over an
arbitrary list of items in a variable and split them by using the `split` subkey
to specify the delimiter:

```yaml
version: 3

tasks:
  foo:
    vars:
      LIST: 'foo,bar,baz'
    cmds:
      - for:
          var: LIST
          split: ','
        cmd: echo {{.ITEM}}
```

Because this experiment adds support for "collection-type" variables, the `for`
keyword has been updated to support looping over arrays directly:

```yaml
version: 3

tasks:
  foo:
    vars:
      LIST: [foo, bar, baz]
    cmds:
      - for:
          var: LIST
        cmd: echo {{.ITEM}}
```

This also works for maps. When looping over a map we also make an additional
`{{.KEY}}` variable availabe that holds the string value of the map key.
Remember that maps are unordered, so the order in which the items are looped
over is random:

```yaml
version: 3

tasks:
  foo:
    vars:
      MAP:
        KEY_1:
          SUBKEY: sub_value_1
        KEY_2:
          SUBKEY: sub_value_2
        KEY_3:
          SUBKEY: sub_value_3
    cmds:
      - for:
          var: MAP
        cmd: echo {{.KEY}} {{.ITEM.SUBKEY}}
```

String splitting is still supported and remember that for simple cases, you have
always been able to loop over an array without using variables at all:

```yaml
version: 3

tasks:
  foo:
    cmds:
      - for: [foo, bar, baz]
        cmd: echo {{.ITEM}}
```

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
      - 'echo {{.CALCULATED_VAR}}'
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
      - 'echo {{.CALCULATED_VAR}}'
```

If your current Taskfile contains a string variable that begins with a `$`, you
will now need to escape the `$` with a backslash (`\`) to stop Task from
executing it as a command.

<!-- prettier-ignore-start -->

[enabling-experiments]: /experiments/#enabling-experiments

[slim-sprig]: https://go-task.github.io/slim-sprig/

<!-- prettier-ignore-end -->
