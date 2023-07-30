---
slug: /experiments/
sidebar_position: 5
---

# Deneyler

:::caution

Tüm deneysel özellikler, _herhangi bir zamanda_ önemli değişikliklere ve/veya kaldırmaya tabidir. Bu özellikleri bir üretim ortamında kullanmamanızı kesinlikle öneririz. Yanlızca test ve geri bildirim amaçlıdır.

:::

Task'ın hızlı bir şekilde gelişmesine izin vermek için, deneysel bayrakların arkasındaki küçük sürümlerde büyük değişiklikler yapıyoruz. Bu, büyük bir sürüm yayınlanmadan önce son değişiklikler hakkında geri bildirim toplamamızı sağlar. Bu belgede, mevcut deneysel özellik grubu ve bunların yerini alması amaçlanan kullanımdan kaldırılmış özellik açıklanmaktadır.

Deneysel bir özelliği şu şekilde etkinleştirebilirsiniz:

1. İlgili ortam değişkenini bir görev komutunun önünde kullanma. Örneğin, `TASK_X_{FEATURE}=1 task {my-task}`. Bu, Task'ın deneysel özellikleri test etmek için tek seferlik çağrıları için tasarlanmıştır.
1. Using the relevant environment variable in your "dotfiles" (e.g. `.bashrc`, `.zshrc` etc.). This is intended for permanently enabling experimental features in your environment.
1. Creating a `.env` file in the same directory as your root Taskfile that contains the relevant environment variables. e.g.

```shell
# .env
TASK_X_FEATURE=1
```

## Current Experimental Features and Deprecations

Each section below details an experiment or deprecation and explains what the flags/environment variables to enable the experiment are and how the feature's behavior will change. It will also explain what you need to do to migrate any existing Taskfiles to the new behavior.

<!-- EXPERIMENT TEMPLATE - Include sections as necessary...

### ![experiment] {Feature} ([#{issue}](https://github.com/go-task/task/issues/{issue})), ...)

- Environment variable: `TASK_X_{feature}`
- Deprecates: {list any existing functionality that will be deprecated by this experiment}
- Breaks: {list any existing functionality that will be broken by this experiment}

{Short description of the feature}

{Short explanation of how users should migrate to the new behavior}

-->

### ![deprecated][] Version 2 Schema ([#1197][deprecate-version-2-schema])

The Taskfile v2 schema was introduced in March 2018 and replaced by version 3 in August the following year. Users have had a long time to update and so we feel that it is time to tidy up the codebase and focus on new functionality instead.

This notice does not mean that we are immediately removing support for version 2 schemas. However, support will not be extended to future major releases and we _strongly recommend_ that anybody still using a version 2 schema upgrades to version 3 as soon as possible.

A list of changes between version 2 and version 3 are available in the [Task v3 Release Notes][version-3-release-notes].

### ![experiment][] Gentle Force ([#1200](https://github.com/go-task/task/issues/1200))

- Environment variable: `TASK_X_FORCE=1`
- Breaks: `--force` flag

The `--force` flag currently forces _all_ tasks to run regardless of the status checks. This can be useful, but we have found that most of the time users only expect the direct task they are calling to be forced and _not_ all of its dependant tasks.

This experiment changes the `--force` flag to only force the directly called task. All dependant tasks will have their statuses checked as normal and will only run if Task considers them to be out of date. A new `--force-all` flag will also be added to maintain the current behavior for users that need this functionality.

If you want to migrate, but continue to force all dependant tasks to run, you should replace all uses of the `--force` flag with `--force-all`. Alternatively, if you want to adopt the new behavior, you can continue to use the `--force` flag as you do now!

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[deprecate-version-2-schema]: https://github.com/go-task/task/issues/1197
[version-3-release-notes]: https://github.com/go-task/task/releases/tag/v3.0.0
[deprecated]: https://img.shields.io/badge/deprecated-red
[experiment]: https://img.shields.io/badge/experiment-yellow
