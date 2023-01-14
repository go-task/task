---
slug: /styleguide/
sidebar_position: 6
---

# Styleguide

This is the official Task styleguide for `Taskfile.yml` files. This guide
contains some basic instructions to keep your Taskfile clean and familiar to
other users.

This contains general guidelines, but they don't necessarily need to be strictly
followed. Feel free to disagree and proceed differently at some point if you
need or want to. Also, feel free to open issues or pull requests with
improvements to this guide.

## Use `Taskfile.yml` and not `taskfile.yml`

```yaml
# bad
taskfile.yml


# good
Taskfile.yml
```

This is important especially for Linux users. Windows and macOS have case
insensitive filesystems, so `taskfile.yml` will end up working, even that not
officially supported. On Linux, only `Taskfile.yml` will work, though.

## Use the correct order of keywords

- `version:`
- `includes:`
- Configuration ones, like `output:`, `silent:`, `method:` and `run:`
- `vars:`
- `env:`, `dotenv:`
- `tasks:`

## Use 2 spaces for indentation

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

## Separate with spaces the mains sections

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

## Add spaces between tasks

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

## Use upper-case variable names

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

## Don't wrap vars in spaces when templating

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

This convention is also used by most people for any Go templating.

## Separate task name words with a dash

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

## Use colon for task namespacing

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
