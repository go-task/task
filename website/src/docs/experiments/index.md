---
title: Experiments
description: Guide to Task’s experimental features and how to use them
outline: deep
---

# Experiments

::: warning

All experimental features are subject to breaking changes and/or removal _at any
time_. We strongly recommend that you do not use these features in a production
environment. They are intended for testing and feedback only.

:::

In order to allow Task to evolve quickly, we sometimes roll out breaking changes
to minor versions behind experimental flags. This allows us to gather feedback
on breaking changes before committing to a major release. This process can also
be used to gather feedback on important non-breaking features before their
design is completed. This document describes the
[experiment workflow](#workflow) and how you can get involved.

You can view the full list of active experiments in the sidebar submenu to the
left of the page and click on each one to find out more about it.

## Enabling Experiments

Task uses environment variables to detect whether or not an experiment is
enabled. All of the experiment variables will begin with the same `TASK_X_`
prefix followed by the name of the experiment. You can find the exact name for
each experiment on their respective pages in the sidebar. If the variable is set
`=1` then it will be enabled. Some experiments may have multiple proposals, in
which case, you will need to set the variable equal to the number of the
proposal that you want to enable (`=2`, `=3` etc).

There are three main ways to set the environment variables for an experiment.
Which method you use depends on how you intend to use the experiment:

1. Prefixing your task commands with the relevant environment variable(s). For
   example, `TASK_X_{FEATURE}=1 task {my-task}`. This is intended for one-off
   invocations of Task to test out experimental features.
2. Adding the relevant environment variable(s) in your "dotfiles" (e.g.
   `.bashrc`, `.zshrc` etc.). This will permanently enable experimental features
   for your personal environment.

   ```shell
   # ~/.bashrc
   export TASK_X_FEATURE=1
   ```

3. Creating a `.env` or a `.taskrc.yml` file in the same directory as your root
   Taskfile.\
   The `.env` file should contain the relevant environment variable(s), while
   the `.taskrc.yml` file should use a YAML format where each experiment is
   defined as a key with a corresponding value.

   This allows you to enable an experimental feature at a project level. If you
   commit this file to source control, then other users of your project will
   also have these experiments enabled.

   If both files are present, the values in the `.taskrc.yml` file will take
   precedence.

::: code-group

```yaml [.taskrc.yml]
experiments:
  FEATURE: 1
```

```shell [.env]
TASK_X_FEATURE=1
```

:::

## Workflow

Experiments are a way for us to test out new features in Task before committing
to them in a major release. Because this concept is built around the idea of
feedback from our community, we have built a workflow for the process of
introducing these changes. This ensures that experiments are given the attention
and time that they need and that we are getting the best possible results out of
them.

The sections below describe the various stages that an experiment must go
through from its proposal all the way to being released in a major version of
Task.

### 1. Proposal

All experimental features start with a proposal in the form of a GitHub issue.
If the maintainers decide that an issue has enough support and is a breaking
change or is complex/controversial enough to require user feedback, then the
issue will be marked with the `status: proposal` label. At this point, the issue
becomes a proposal and a period of consultation begins. During this period, we
request that users provide feedback on the proposal and how it might effect
their use of Task. It is up to the discretion of the maintainers to decide how
long this period lasts.

### 2. Draft

Once a proposal's consultation ends, a contributor may pick up the work and
begin the initial implementation. Once a PR is opened, the maintainers will
ensure that it meets the requirements for an experimental feature (i.e. flags
are in the right format etc) and merge the feature. Once this code is released,
the status will be updated via the `status: draft` label. This indicates that an
implementation is now available for use in a release and the experiment is open
for feedback.

::: info

During the draft period, major changes to the implementation may be made based
on the feedback received from users. There are _no stability guarantees_ and
experimental features may be abandoned _at any time_.

:::

### 3. Candidate

Once an acceptable level of consensus has been reached by the community and
feedback/changes are less frequent/significant, the status may be updated via
the `status: candidate` label. This indicates that a proposal is _likely_ to
accepted and will enter a period for final comments and minor changes.

### 4. Stable

Once a suitable amount of time has passed with no changes or feedback, an
experiment will be given the `status: stable` label. At this point, the
functionality will be treated like any other feature in Task and any changes
_must_ be backward compatible. This allows users to migrate to the new
functionality without having to worry about anything breaking in future
releases. This provides the best experience for users migrating to a new major
version.

### 5. Released

When making a new major release of Task, all experiments marked as
`status: stable` will move to `status: released` and their behaviors will become
the new default in Task. Experiments in an earlier stage (i.e. not stable)
cannot be released and so will continue to be experiments in the new version.

### Abandoned / Superseded

If an experiment is unsuccessful at any point then it will be given the
`status: abandoned` or `status: superseded` labels depending on which is more
suitable. These experiments will be removed from Task.
