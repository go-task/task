---
title: Introducing Experiments
description: A look at where task is, where it's going and how we're going to get there.
slug: task-in-2023
authors:
  - pd93
tags:
  - experiments
  - breaking-changes
  - roadmap
  - v4
image: https://i.imgur.com/mErPwqL.png
hide_table_of_contents: false
---

Lately, Task has been growing extremely quickly and I've found myself thinking a lot about the future of the project and how we continue to evolve and grow. I'm not much of a writer, but I think one of the things we could do better is to communicate these kinds of thoughts to the community. So, with that in mind, this is the first (hopefully of many) blog posts talking about Task and what we're up to.

<!--truncate-->

## :calendar: So, what have we been up to?

Over the past 12 months or so, [@andreynering][] (Author and maintainer of the project) and I ([@pd93][]) have been working in our spare time to maintain and improve v3 of Task and we've made some amazing progress. Here are just some of the things we've released in that time:

- An official [extension for VS Code][vscode-task].
- Internal Tasks ([#818](https://github.com/go-task/task/pull/818)).
- Task aliases ([#879](https://github.com/go-task/task/pull/879)).
- Looping over tasks ([#1220](https://github.com/go-task/task/pull/1200)).
- A series of refactors to the core codebase to make it more maintainable and extensible.
- Loads of bug fixes and improvements.
- An integration with [Crowdin][crowdin]. Work is in progress on making our docs available in **7 new languages** (Special thanks to all our translators for the huge help with this!).
- And much, much more! :sparkles:

We're also working on adding some really exciting and highly requested features to Task such as having the ability to run remote Taskfiles ([#1317](https://github.com/go-task/task/issues/1317)).

None of this would have been possible without the [150 or so (and growing) contributors][contributors] to the project, numerous sponsors and a passionate community of users. Together we have more than doubled the number of GitHub stars to over 8400 :star: since the beginning of 2022 and this continues to accelerate. We can't thank you all enough for your help and support! :rocket:

[![Star History Chart](https://api.star-history.com/svg?repos=go-task/task&type=Date)](https://star-history.com/#go-task/task&Date)

## What's next? :thinking_face:

It's extremely motivating to see so many people using and loving Task. However, in this time we've also seen an increase in the number of issues and feature requests. In particular, issues that require some kind of breaking change to Task. This isn't a bad thing, but as we grow we need to be more responsible about how we address these changes in a way that ensures stability and compatibility for existing users and their Taskfiles.

At this point you're probably thinking something like:

> "But you use [semantic versioning][semver] - Just release a new major version with your breaking changes."

And you'd be right... sort of. In theory, this sounds great, but the reality is that we don't have the time to commit to a major overhaul of Task in one big bang release. This would require a colossal amount of time and coordination and with full time jobs and personal lives to tend to, this is a difficult commitment to make. Smaller, more frequent major releases are also a significant inconvenience for users as they have to constantly keep up-to-date with our breaking changes. Fortunately, there is a better way.

## What's going to change? :face_with_monocle:

Going forwards, breaking changes will be allowed into _minor_ versions of Task as "experimental features". To access these features users will need opt-in by enabling feature flags. This will allow us to release new features slowly and gather feedback from the community before making them the default behavior in a future major release.

To prepare users for the next major release, we will maintain a list of [deprecated features][deprecations] and [experiments][experiments] on our docs website and publish information on how to migrate to the new behavior.

You can read the [full breaking change proposal][breaking-change-proposal] and view all the [current experiments and their status][experiments-project] on GitHub including the [Gentle Force][gentle-force-experiment] and [Remote Taskfiles][remote-taskfiles-experiment] experiments.

## What will happen to v2/v3 features?

v2 has been [officially deprecated][deprecate-version-2-schema]. If you're still using a Taskfile with `version: "2"` at the top we _strongly recommend_ that you upgrade as soon as possible. Removing v2 will allow us to tidy up the codebase and focus on new functionality instead.

When v4 is released, we will continue to support v3 for a period of time (bug fixes etc). However, since we are moving from a backward-compatibility model to a forwards-compatibility model, **v4 itself will not be backwards compatible with v3**.

## v4 When? :eyes:

:shrug: When it's ready.

In all seriousness, we don't have a timeline for this yet. We'll be working on the most serious deficiencies of the v3 API first and regularly evaluating the state of the project. When we feel its in a good, stable place and we have a clear upgrade path for users and a number of stable experiments, we'll start to think about v4.

## :wave: Final thoughts

Task is growing fast and we're excited to see where it goes next. We hope that the steps we're taking to improve the project and our process will help us to continue to grow. As always, if you have any questions or feedback, we encourage you to comment on or open [issues][issues] and [discussions][discussions] on GitHub. Alternatively, you can join us on [Discord][discord].

I plan to write more of these blog posts in the future on a variety of Task-related topics, so make sure to check in occasionally and see what we're up to!

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[vscode-task]: https://github.com/go-task/vscode-task
[crowdin]: https://crowdin.com
[contributors]: https://github.com/go-task/task/graphs/contributors
[semver]: https://semver.org
[breaking-change-proposal]: https://github.com/go-task/task/discussions/1191
[@andreynering]: https://github.com/andreynering
[@pd93]: https://github.com/pd93
[experiments]: https://taskfile.dev/experiments
[deprecations]: https://taskfile.dev/deprecations
[deprecate-version-2-schema]: https://github.com/go-task/task/issues/1197
[issues]: https://github.com/go-task/task/issues
[discussions]: https://github.com/go-task/task/discussions
[discord]: https://discord.gg/6TY36E39UK
[experiments-project]: https://github.com/orgs/go-task/projects/1
[gentle-force-experiment]: https://github.com/go-task/task/issues/1200
[remote-taskfiles-experiment]: https://github.com/go-task/task/issues/1317
