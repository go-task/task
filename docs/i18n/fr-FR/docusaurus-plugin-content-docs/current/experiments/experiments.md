---
slug: /experiments/
sidebar_position: 6
---

# Expérimentations

:::caution

Toutes les fonctionnalités expérimentales sont sujettes à des changements brisant la fonctionnalité et/ou à la suppression de cette dernière _à tout moment _. Nous vous recommandons fortement de ne pas utiliser ces fonctionnalités dans en production. Ils sont seulement destinés à être testés et obtenir des retours dessus.

:::

Afin de permettre à Task d'évoluer rapidement, nous déployons des changements cassants sur des versions mineures activables par des options expérimentales. Cela nous permet d'obtenir des retours sur des changements brisants des fonctionnalités avant de les livrer dans une version majeure. Ce document décrit l'ensemble actuel de fonctionnalités expérimentales et leur statut dans le [workflow](#workflow).

Vous pouvez consulter une liste complète des expérimentations en cours dans la section "Expérimentations" de la barre latérale .

Vous pouvez activer une fonctionnalité expérimentale par:

1. Utiliser la variable d'environnement pertinente devant une commande Task. Par exemple, `TASK_X_{FEATURE}=1 task {my-task}`. Ceci est prévu pour faire appel à une expérimentation durant une seule commande Task afin de tester une fonctionnalité expérimentale.
1. Utiliser la variable d'environnement pertinente dans vos "dotfiles" (par exemple `.bashrc`, `.zshrc` etc.). Ceci est destiné à l'activation permanente des fonctionnalités expérimentales dans votre environnement.
1. Création d'un fichier `.env` dans le même répertoire que votre fichier Taskfile contenant les variables d'environnement pertinentes pour vos tests. Par exemple :

```shell
# .env
TASK_X_FEATURE=1
```

## Current Experimental Features and Deprecations

Each section below details an experiment or deprecation and explains what the flags/environment variables to enable the experiment are and how the feature's behavior will change. It will also explain what you need to do to migrate any existing Taskfiles to the new behavior.

## Workflow

Experiments are a way for us to test out new features in Task before committing to them in a major release. Because this concept is built around the idea of feedback from our community, we have built a workflow for the process of introducing these changes. This ensures that experiments are given the attention and time that they need and that we are getting the best possible results out of them.

The sections below describe the various stages that an experiment must go through from its proposal all the way to being released in a major version of Task.

### 1. Proposal

All experimental features start with a proposal in the form of a GitHub issue. If the maintainers decide that an issue has enough support and is a breaking change or is complex/controversial enough to require user feedback, then the issue will be marked with the ![proposal][] label. At this point, the issue becomes a proposal and a period of consultation begins. During this period, we request that users provide feedback on the proposal and how it might effect their use of Task. It is up to the discretion of the maintainers to decide how long this period lasts.

### 2. Draft

Once a proposal's consultation ends, a contributor may pick up the work and begin the initial implementation. Once a PR is opened, the maintainers will ensure that it meets the requirements for an experimental feature (i.e. flags are in the right format etc) and merge the feature. Once this code is released, the status will be updated via the ![draft][] label. This indicates that an implementation is now available for use in a release and the experiment is open for feedback.

:::note

During the draft period, major changes to the implementation may be made based on the feedback received from users. There are _no stability guarantees_ and experimental features may be abandoned _at any time_.

:::

### 3. Candidate

Once an acceptable level of consensus has been reached by the community and feedback/changes are less frequent/significant, the status may be updated via the ![candidate][] label. This indicates that a proposal is _likely_ to accepted and will enter a period for final comments and minor changes.

### 4. Stable

Once a suitable amount of time has passed with no changes or feedback, an experiment will be given the ![stable][] label. At this point, the functionality will be treated like any other feature in Task and any changes _must_ be backward compatible. This allows users to migrate to the new functionality without having to worry about anything breaking in future releases. This provides the best experience for users migrating to a new major version.

### 5. Released

When making a new major release of Task, all experiments marked as ![stable][] will move to ![released][] and their behaviors will become the new default in Task. Experiments in an earlier stage (i.e. not stable) cannot be released and so will continue to be experiments in the new version.

### Abandoned / Superseded

If an experiment is unsuccessful at any point then it will be given the ![abandoned][] or ![superseded][] labels depending on which is more suitable. These experiments will be removed from Task.

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[proposal]: https://img.shields.io/badge/experiment:%20proposal-purple
[draft]: https://img.shields.io/badge/experiment:%20draft-purple
[candidate]: https://img.shields.io/badge/experiment:%20candidate-purple
[stable]: https://img.shields.io/badge/experiment:%20stable-purple
[released]: https://img.shields.io/badge/experiment:%20released-purple
[abandoned]: https://img.shields.io/badge/experiment:%20abandoned-purple
[superseded]: https://img.shields.io/badge/experiment:%20superseded-purple
