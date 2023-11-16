---
slug: /
sidebar_position: 1
title: Accueil
---

# Task

<div align="center">
  <img id="logo" src="img/logo.svg" height="250px" width="250px" />
</div>

Task est un exécuteur de tâches / de build qui vise à être plus simple et facile à utiliser que, par exemple, [GNU Make][make].

Comme il est écrit en [Go][go], Task n'est qu'un binaire unique et n'a aucune dépendance. Cela signifie que vous n'avez pas besoin d'une installation compliquée simplement pour utiliser un outil de build.

Une fois [installé](installation.md), il vous suffit de décrire vos tâches de build en utilisant un simple schéma [YAML][yaml] dans un fichier appelé `Taskfile.yml`:

```yaml title="Taskfile.yml"
version: '3'

tasks:
  hello:
    cmds:
      - echo 'Hello World from Task!'
    silent: true
```

Et appelez-le en exécutant `task hello` depuis votre terminal.

L'exemple ci-dessus n'est que le début, vous pouvez jeter un coup d'œil au [guide d'utilisation](/usage) pour vérifier la documentation complète du schéma et les fonctionnalités de Task.

## Fonctionnalités

- [Installation facile](installation.md): il suffit de télécharger un seul binaire, ajoutez le à `$PATH` et vous avez terminé ! Ou vous pouvez également installer en utilisant [Homebrew][homebrew], [Snapcraft][snapcraft] ou [Scoop][scoop] si vous le souhaitez.
- Disponible sur les CIs: en ajoutant [cette commande simple](installation.md#install-script) à installer sur votre script CI, vous êtes prêt à utiliser Task dans le cadre de votre pipeline CI ;
- Multi-plateforme : alors que la plupart des outils de compilation ne fonctionnent bien que sous Linux ou macOS, Task prend également en charge Windows grâce à [cet interpréteur shell pour Go][sh].
- Idéal pour la génération de code : vous pouvez facilement [empêcher une tâche de s'exécuter](/usage#prevent-unnecessary-work) si un ensemble donné de fichiers n'ont pas changé depuis le dernier lancement (basé soit sur son horodatage soit son contenu).

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[make]: https://www.gnu.org/software/make/
[go]: https://go.dev/
[yaml]: http://yaml.org/
[homebrew]: https://brew.sh/
[snapcraft]: https://snapcraft.io/
[scoop]: https://scoop.sh/
[sh]: https://github.com/mvdan/sh
