---
title: 'Template Functions'
description:
  Deprecation of some templating functions in Task, with guidance on their
  replacements.
outline: deep
---

# Template Functions

::: danger

This deprecation breaks the following functionality:

- A small set of templating functions

:::

The following templating functions are deprecated. Any replacement functions are
listed besides the function being removed.

| Deprecated function | Replaced by |
| ------------------- | ----------- |
| `IsSH`              | -           |
| `FromSlash`         | `fromSlash` |
| `ToSlash`           | `toSlash`   |
| `ExeExt`            | `exeExt`    |

## Functions renamed by sprout

Task's generic template functions moved from slim-sprig to
[sprout](https://docs.atom.codes/sprout), which renamed a number of them. The
old names still work as aliases, and Task reports their use when run with
`--verbose`.

| Deprecated function               | Replaced by                      |
| --------------------------------- | -------------------------------- |
| `upper`                           | `toUpper`                        |
| `lower`                           | `toLower`                        |
| `title`                           | `toTitleCase`                    |
| `atoi`, `int`                     | `toInt`                          |
| `int64`                           | `toInt64`                        |
| `float64`                         | `toFloat64`                      |
| `toDecimal`                       | `toOctal`                        |
| `toStrings`                       | `strSlice`                       |
| `b64enc`, `b64dec`                | `base64Encode`, `base64Decode`   |
| `b32enc`, `b32dec`                | `base32Encode`, `base32Decode`   |
| `base`, `dir`, `ext`              | `pathBase`, `pathDir`, `pathExt` |
| `clean`, `isAbs`                  | `pathClean`, `pathIsAbs`         |
| `expandenv`                       | `expandEnv`                      |
| `ago`                             | `dateAgo`                        |
| `trimall`                         | `trimAll`                        |
| `push`, `mustPush`                | `append`                         |
| `tuple`                           | `list`                           |
| `biggest`                         | `max`                            |
| `date_in_zone`                    | `dateInZone`                     |
| `date_modify`, `must_date_modify` | `dateModify`                     |

Ten functions also changed argument order, and the old order is deprecated. See
[Migrating from slim-sprig](../reference/templating.md#migrating-from-slim-sprig)
for the full list and for the behaviour changes that came with the move.
