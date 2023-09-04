---
slug: /deprecations/version-2-schema/
---

# Version 2 Schema

- Issue: [#1197][deprecate-version-2-schema]
- Breaks:
  - Any Taskfiles that use the version 2 schema
  - `Taskvar.yml` files

The Taskfile v2 schema was introduced in March 2018 and replaced by version 3 in August the following year. Users have had a long time to update and so we feel that it is time to tidy up the codebase and focus on new functionality instead.

This notice does not mean that we are immediately removing support for version 2 schemas. However, support will not be extended to future major releases and we _strongly recommend_ that anybody still using a version 2 schema upgrades to version 3 as soon as possible.

A list of changes between version 2 and version 3 are available in the [Task v3 Release Notes][version-3-release-notes].

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[deprecate-version-2-schema]: https://github.com/go-task/task/issues/1197
[version-3-release-notes]: https://github.com/go-task/task/releases/tag/v3.0.0
