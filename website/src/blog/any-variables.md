---
title: Any Variables
author: pd93
date: 2024-05-09
outline: deep
---

# Any Variables

<AuthorCard :author="$frontmatter.author" />

Task has always had variables, but even though you were able to define them
using different YAML types, they would always be converted to strings by Task.
This limited users to string manipulation and encouraged messy workarounds for
simple problems. Starting from [v3.37.0][v3.37.0], this is no longer the case!
Task now supports most variable types, including **booleans**, **integers**,
**floats** and **arrays**!

## What's the big deal?

These changes allow you to use variables in a much more natural way and opens up
a wide variety of sprig functions that were previously useless. Take a look at
some of the examples below for some inspiration.

### Evaluating booleans

No more comparing strings to "true" or "false". Now you can use actual boolean
values in your templates:

::: code-group

```yaml [Before]
version: 3

tasks:
  foo:
    vars:
      BOOL: true # <-- Parsed as a string even though its a YAML boolean
    cmds:
      - '{{if eq .BOOL "true"}}echo foo{{end}}'
```

```yaml [After]
version: 3

tasks:
  foo:
    vars:
      BOOL: true # <-- Parsed as a boolean
    cmds:
      - '{{if .BOOL}}echo foo{{end}}' # <-- No need to compare to "true"
```

:::

### Arithmetic

You can now perform basic arithmetic operations on integer and float variables:

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

You can use any of the following arithmetic functions: `add`, `sub`, `mul`,
`div`, `mod`, `max`, `min`, `floor`, `ceil`, `round` and `randInt`. Check out
the [slim-sprig math documentation][slim-sprig-math] for more information.

### Arrays

You can now range over arrays inside templates and use list-based functions:

```yaml
version: 3

tasks:
  foo:
    vars:
      ARRAY: [1, 2, 3]
    cmds:
      - 'echo {{range .ARRAY}}{{.}}{{end}}'
```

You can use any of the following list-based functions: `first`, `rest`, `last`,
`initial`, `append`, `prepend`, `concat`, `reverse`, `uniq`, `without`, `has`,
`compact`, `slice` and `chunk`. Check out the [slim-sprig lists
documentation][slim-sprig-list] for more information.

### Looping over variables using `for`

Previously, you would have to use a delimiter separated string to loop over an
arbitrary list of items in a variable and split them by using the `split` subkey
to specify the delimiter. However, we have now added support for looping over
"collection-type" variables using the `for` keyword, so now you are able to loop
over list variables directly:

::: code-group

```yaml [Before]
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

```yaml [After]
version: 3

tasks:
  foo:
    vars:
      LIST: ['foo', 'bar', 'baz']
    cmds:
      - for:
          var: LIST
        cmd: echo {{.ITEM}}
```

:::

## What about maps?

Maps were originally included in the Any Variables experiment. However, they
weren't quite ready yet. Instead of making you wait for everything to be ready
at once, we have released support for all other variable types and we will
continue working on map support in the new "[Map Variables][map-variables]"
experiment.

We're looking for feedback on a couple of different proposals, so please give
them a go and let us know what you think. :pray:

[v3.37.0]: https://github.com/go-task/task/releases/tag/v3.37.0
[slim-sprig-math]: https://sprig.taskfile.dev/math.html
[slim-sprig-list]: https://sprig.taskfile.dev/lists.html
