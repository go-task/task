---
title: Style Guide
description:
  Official style guide for Taskfile.yml files with best practices and
  recommended conventions
outline: deep
---

# Style Guide

This is the official style guide for `Taskfile.yml` files. It provides basic
instructions for keeping your Taskfiles clean and familiar to other users.

This guide contains general guidelines, but they do not necessarily need to be
followed strictly. Feel free to disagree and do things differently if you need
or want to. Any improvements to this guide are welcome! Please open an issue or
create a pull request to contribute.

## Use the suggested ordering of the main sections

```yaml
version:
includes:
# optional configurations (output, silent, method, run, etc.)
vars:
env: # followed or replaced by dotenv
tasks:
```

## Use two spaces for indentation

This is the most common convention for YAML files, and Task follows it.

```yaml
# bad
tasks:
    foo:
        cmds:
            - echo 'foo'


# good
tasks:
  foo:
    cmds:
      - echo 'foo'
```

## Separate the main sections with empty lines

```yaml
# bad
version: '3'
includes:
  docker: ./docker/Taskfile.yml
output: prefixed
vars:
  FOO: bar
env:
  BAR: baz
tasks:
  # ...


# good
version: '3'

includes:
  docker: ./docker/Taskfile.yml

output: prefixed

vars:
  FOO: bar

env:
  BAR: baz

tasks:
  # ...
```

## Separate tasks with empty lines

```yaml
# bad
version: '3'

tasks:
  foo:
    cmds:
      - echo 'foo'
  bar:
    cmds:
      - echo 'bar'
  baz:
    cmds:
      - echo 'baz'


# good
version: '3'

tasks:
  foo:
    cmds:
      - echo 'foo'

  bar:
    cmds:
      - echo 'bar'

  baz:
    cmds:
      - echo 'baz'
```

## Use only uppercase letters for variable names

```yaml
# bad
version: '3'

vars:
  binary_name: myapp

tasks:
  build:
    cmds:
      - go build -o {{.binary_name}} .


# good
version: '3'

vars:
  BINARY_NAME: myapp

tasks:
  build:
    cmds:
      - go build -o {{.BINARY_NAME}} .
```

## Avoid using whitespace when templating variables

```yaml
# bad
version: '3'

tasks:
  greet:
    cmds:
      - echo '{{ .MESSAGE }}'


# good
version: '3'

tasks:
  greet:
    cmds:
      - echo '{{.MESSAGE}}'
```

This convention is also commonly used in templates for the Go programming
language.

## Use kebab case for task names

```yaml
# bad
version: '3'

tasks:
  do_something_fancy:
    cmds:
      - echo 'Do something'


# good
version: '3'

tasks:
  do-something-fancy:
    cmds:
      - echo 'Do something'
```

## Use a colon to separate the task namespace and name

```yaml
# good
version: '3'

tasks:
  docker:build:
    cmds:
      - docker ...

  docker:run:
    cmds:
      - docker-compose ...
```

This is also done automatically when using included Taskfiles.

## Prefer using external scripts instead of multi-line commands

```yaml
# bad
version: '3'

tasks:
  build:
    cmds:
      - |
        for i in $(seq 1 10); do
          echo $i
          echo "some other complex logic"
        done'

# good
version: '3'

tasks:
  build:
    cmds:
      - ./scripts/my_complex_script.sh
```
