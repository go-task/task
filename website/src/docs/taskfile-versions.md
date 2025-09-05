---
title: Taskfile Versions
description:
  How to use the Taskfile schema version to ensure users are using the correct
  versions of Task
outline: deep
---

# Taskfile Versions

The Taskfile schema slowly changes as new features are added and old ones are
removed. This document explains how to use a Taskfile's schema version to ensure
that the users of your Taskfile are using the correct versions of Task.

## What the Taskfile version means

The schema version at the top of every Taskfile corresponds to a version of the
Task CLI, and by extension, the features that are provided by that version. When
creating a Taskfile, you should specify the _minimum_ version of Task that
supports the features you require. If you try to run a Taskfile with a version
of Task that does not meet this minimum required version, it will exit with an
error. For example, given a Taskfile that starts with:

```yaml
version: '3.2.1'
```

When executed with Task `v3.2.0`, it will exit with an error. Running with
version `v3.2.1` or higher will work as expected.

Task accepts any [SemVer][semver] compatible string including versions which
omit the minor or patch numbers. For example, `3`, `3.0`, and `3.0.0` all mean
the same thing and are all valid. Most Taskfiles only specify the major version
number. However it can be useful to be more specific when you intend to share a
Taskfile with others.

For example, the Taskfile below makes use of aliases:

```yaml
version: '3'

tasks:
  hello:
    aliases:
      - hi
      - hey
    cmds:
      - echo "Hello, world!"
```

Aliases were introduced in Task `v3.17.0`, but the Taskfile only specifies `3`
as the version. This means that a user who has `v3.16.0` or lower installed will
get a potentially confusing error message when trying to run the Task as the
Taskfile specifies that any version greater or equal to `v3.0.0` is fine.

Instead, we should start the file like this:

```yaml
version: '3.17'
```

Now when someone tries to run the Taskfile with an older version of Task, they
will receive an error prompting them to upgrade their version of Task to
`v3.17.0` or greater.

## Versions 1 & 2

Version 1 and 2 of Task are no longer officially supported and anyone still
using them is strongly encouraged to upgrade to the latest version of Task.

While `version: 2` of Task did support schema versions, the behavior did not
work in quite the same way and cannot be relied upon for the purposes discussed
above.

[semver]: https://semver.org/
