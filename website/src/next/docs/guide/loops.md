---
title: Loops
description:
  Repeat a command over a static list, a matrix, a variable, your task's
  sources, or other tasks.
section: Guide
docType: guide
outline: deep
---

# Loops

Task allows you to loop over certain values and execute a command for each.
There are a number of ways to do this depending on the type of value you want to
loop over.

## Looping over a static list

The simplest kind of loop is an explicit one. This is useful when you want to
loop over a set of values that are known ahead of time.

```yaml
version: '3'

tasks:
  default:
    cmds:
      - for: ['foo.txt', 'bar.txt']
        cmd: cat {{ .ITEM }}
```

## Looping over a matrix

If you need to loop over all permutations of multiple lists, you can use the
`matrix` property. This should be familiar to anyone who has used a matrix in a
CI/CD pipeline.

```yaml
version: '3'

tasks:
  default:
    silent: true
    cmds:
      - for:
          matrix:
            OS: ['windows', 'linux', 'darwin']
            ARCH: ['amd64', 'arm64']
        cmd: echo "{{.ITEM.OS}}/{{.ITEM.ARCH}}"
```

This will output:

```txt
windows/amd64
windows/arm64
linux/amd64
linux/arm64
darwin/amd64
darwin/arm64
```

You can also use references to other variables as long as they are also lists:

```yaml
version: '3'

vars:
  OS_VAR: ['windows', 'linux', 'darwin']
  ARCH_VAR: ['amd64', 'arm64']

tasks:
  default:
    cmds:
      - for:
          matrix:
            OS:
              ref: .OS_VAR
            ARCH:
              ref: .ARCH_VAR
        cmd: echo "{{.ITEM.OS}}/{{.ITEM.ARCH}}"
```

## Looping over your task's sources or generated files

You are also able to loop over the sources of your task or the files it
generates:

::: code-group

```yaml [Sources]
version: '3'

tasks:
  default:
    sources:
      - foo.txt
      - bar.txt
    cmds:
      - for: sources
        cmd: cat {{ .ITEM }}
```

```yaml [Generates]
version: '3'

tasks:
  default:
    generates:
      - foo.txt
      - bar.txt
    cmds:
      - for: generates
        cmd: cat {{ .ITEM }}
```

:::

This will also work if you use globbing syntax in `sources` or `generates`. For
example, if you specify a source for `*.txt`, the loop will iterate over all
files that match that glob.

Paths will always be returned as paths relative to the task directory. If you
need to convert this to an absolute path, you can use the built-in `joinPath`
function. There are some
[special variables](../reference/templating.md#special-variables) that you may
find useful for this.

::: code-group

```yaml [Sources]
version: '3'

tasks:
  default:
    vars:
      MY_DIR: /path/to/dir
    dir: '{{.MY_DIR}}'
    sources:
      - foo.txt
      - bar.txt
    cmds:
      - for: sources
        cmd: cat {{joinPath .MY_DIR .ITEM}}
```

```yaml [Generates]
version: '3'

tasks:
  default:
    vars:
      MY_DIR: /path/to/dir
    dir: '{{.MY_DIR}}'
    generates:
      - foo.txt
      - bar.txt
    cmds:
      - for: generates
        cmd: cat {{joinPath .MY_DIR .ITEM}}
```

:::

## Looping over variables

To loop over the contents of a variable, use the `var` key followed by the name
of the variable you want to loop over. By default, string variables will be
split on any whitespace characters.

```yaml
version: '3'

tasks:
  default:
    vars:
      MY_VAR: foo.txt bar.txt
    cmds:
      - for: { var: MY_VAR }
        cmd: cat {{.ITEM}}
```

If you need to split a string on a different character, you can do this by
specifying the `split` property:

```yaml
version: '3'

tasks:
  default:
    vars:
      MY_VAR: foo.txt,bar.txt
    cmds:
      - for: { var: MY_VAR, split: ',' }
        cmd: cat {{.ITEM}}
```

You can also loop over arrays and maps directly:

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

When looping over a map we also make an additional <span v-pre>`{{.KEY}}`</span>
variable available that holds the string value of the map key. Remember that
maps are unordered, so the order in which the items are looped over is random.

All of this also works with dynamic variables!

```yaml
version: '3'

tasks:
  default:
    vars:
      MY_VAR:
        sh: find -type f -name '*.txt'
    cmds:
      - for: { var: MY_VAR }
        cmd: cat {{.ITEM}}
```

## Renaming variables

If you want to rename the iterator variable to make it clearer what the value
contains, you can do so by specifying the `as` property:

```yaml
version: '3'

tasks:
  default:
    vars:
      MY_VAR: foo.txt bar.txt
    cmds:
      - for: { var: MY_VAR, as: FILE }
        cmd: cat {{.FILE}}
```

## Looping over tasks

Because the `for` property is defined at the `cmds` level, you can also use it
alongside the `task` keyword to run tasks multiple times with different
variables.

```yaml
version: '3'

tasks:
  default:
    cmds:
      - for: [foo, bar]
        task: my-task
        vars:
          FILE: '{{.ITEM}}'

  my-task:
    cmds:
      - echo '{{.FILE}}'
```

Or if you want to run different tasks depending on the value of the loop:

```yaml
version: '3'

tasks:
  default:
    cmds:
      - for: [foo, bar]
        task: task-{{.ITEM}}

  task-foo:
    cmds:
      - echo 'foo'

  task-bar:
    cmds:
      - echo 'bar'
```

## Looping over dependencies

All of the above looping techniques can also be applied to the `deps` property.
This allows you to combine loops with concurrency:

```yaml
version: '3'

tasks:
  default:
    deps:
      - for: [foo, bar]
        task: my-task
        vars:
          FILE: '{{.ITEM}}'

  my-task:
    cmds:
      - echo '{{.FILE}}'
```

It is important to note that as `deps` are run in parallel, the order in which
the iterations are run is not guaranteed and the output may vary. For example,
the output of the above example may be either:

```shell
foo
bar
```

or

```shell
bar
foo
```
