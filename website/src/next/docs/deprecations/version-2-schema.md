---
title: 'Version 2 Schema (#1197)'
description: Deprecation of Taskfile schema version 2 and migration to version 3
outline: deep
---

# Version 2 Schema (#1197)

::: danger

This deprecation breaks the following functionality:

- Any Taskfiles that use the version 2 schema
- `Taskvar.yml` files

:::

The Taskfile version 2 schema was introduced in March 2018 and replaced by
version 3 in August 2019. In May 2023 [we published a deprecation
notice][deprecation-notice] for the version 2 schema on the basis that the vast
majority of users had already upgraded to version 3 and removing support for
version 2 would allow us to tidy up the codebase and focus on new functionality
instead.

In December 2023, the final version of Task that supports the version 2 schema
([v3.33.0][v3.33.0]) was published and all legacy code was removed from Task's
main branch. To use a more recent version of Task, you will need to ensure that
your Taskfile uses the version 3 schema instead. A list of changes between
version 2 and version 3 are available in the [Task v3 Release Notes][v3.0.0].

[v3.0.0]: https://github.com/go-task/task/releases/tag/v3.0.0
[v3.33.0]: https://github.com/go-task/task/releases/tag/v3.33.0
[deprecation-notice]: https://github.com/go-task/task/issues/1197
