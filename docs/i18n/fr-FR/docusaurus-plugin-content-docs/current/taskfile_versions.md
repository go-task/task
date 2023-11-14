---
slug: /taskfile-versions/
sidebar_position: 5
---

# Versions Taskfile

La syntaxe et les fonctionnalités du fichier Taskfile changent avec le temps. Ce document explique quels sont les changments pour chacune des versions et comment vous pouvez mettre à jour votre Taskfile.

## Qu'est-ce que la version de Taskfile signifie

La version de Taskfile suit la version de Task. Par exemple : Le changement pour la version `2` dans Taskfile signifie que la version `v2.0.0` de Task doit être publiée pour pouvoir le supporter.

Le paramètrre `version:` dans Taskfile accepte une version suivant la nomenclature semver. Donc `2`, `2.0` et `2.0.0` sont acceptés. Si vous choisissez d'utiliser la version `2.0`, Task ne va pas activer les fonctionnalités des versions `2.1` et celles d'après. Mais si vous choississez d'utiliser la version `2`, alors toutes les fonctionnalités des versions `2.x.x` seront disponibles, et non celles des versions `3.0.0` et celles d'après.

## Version 3 ![Dernier](https://img.shields.io/badge/latest-brightgreen)

Voici quelques modifications majeures effectuées sur `v3`:

- Les logs de Task dans le terminal sont colorés
- Ajout du support des fichiers `.env` et similaires
- Ajout du paramètre `label:` dans les tâches pour que l'on puisse renommer la tâche dans les logs
- Le paramètre global `method:` a été ajouté pour permettre de définir la méthode par défault, et la valeur par défaut de Task a été changée pour `checksum`
- Deux variables magiques ont été ajoutées lors de l'utilisation de `status:`: `CHECKSUM` et `TIMESTAMP` qui contiennent respectivement le checksum XXH3 et le plus récent timestamp de modification des fichiers répertoriés dans `sources:`
- Aussi, la variable `TASK` est toujours disponible avec le nom de la tâche courante
- Les variables CLI sont toujours traitées comme des variables globales
- Ajout de l'option `dir:` dans `includes` pour permettre de choisir dans quel dossier un Taskfile doit être exécuté :

```yaml
includes:
  docs:
    taskfile: ./docs
    dir: ./docs
```

- Implémentation de syntaxes courtes. Toutes les syntaxes ci-dessous sont équivalentes:

```yaml
version: '3'

tasks:
  print:
    cmds:
      - echo "Hello, World!"
```

```yaml
version: '3'

tasks:
  print:
    - echo "Hello, World!"
```

```yaml
version: '3'

tasks:
  print: echo "Hello, World!"
```

- Il y a eu une réécriture majeure sur la manière dont les variables sont gérées. C'est maintenant plus simple à comprendre. Les paramètres `expansions:` ont été retirées vu qu'ils n'étaient plus nécessaires. C'est l'ordre dans lequel Task va traiter les variables, chaque niveau peut voir les variables définies par la précédente et les remplacer.
  - Variables d'environnement
  - Variables globales + CLI
  - Variables d'appel
  - Variables Task

## Version 2.6

:::caution

Le support du schéma v2 est [déprécié][deprecate-version-2-schema] et sera retiré dans une future version.

:::

La version 2.6 vient avec des `preconditions` dans les tâches.

```yaml
version: '2'

tasks:
  upload_environment:
    preconditions:
      - test -f .env
    cmds:
      - aws s3 cp .env s3://myenvironment
```

Veuillez consulter la [documentation][includes]

## Version 2.2

:::caution

Le support du schéma v2 est [déprécié][deprecate-version-2-schema] et sera retiré dans une future version.

:::

La version 2.2 est fournie avec une option globale `includes` pour inclure d'autres Taskfiles :

```yaml
version: '2'

includes:
  docs: ./documentation # will look for ./documentation/Taskfile.yml
  docker: ./DockerTasks.yml
```

## Version 2.1

:::caution

Le support du schéma v2 est [déprécié][deprecate-version-2-schema] et sera retiré dans une future version.

:::

La version 2.1 inclut une option globale `output` permettant d'avoir plus de contrôle sur la manière dont les logs sont affichés dans la console (voir la [documentation][output] pour plus d'informations):

```yaml
version: '2'

output: prefixed

tasks:
  server:
    cmds:
      - go run main.go
  prefix: server
```

À partir de cette version, il est également possible d'ignorer les erreurs d'une commande ou d'une tâche (vérifiez la documentation [ici][ignore_errors] ) :

```yaml
version: '2'

tasks:
  example-1:
    cmds:
      - cmd: exit 1
        ignore_error: true
      - echo "This will be print"

  example-2:
    cmds:
      - exit 1
      - echo "This will be print"
    ignore_error: true
```

## Version 2.0

:::caution

Le support du schéma v2 est [déprécié][deprecate-version-2-schema] et sera retiré dans une future version.

:::

À la version 2, nous avons introduit le paramètre `version:` pour nous permettre d'évoluer vers de nouvelles fonctionnalités avec sans casser les fichiers de tâches existants. La nouvelle syntaxe est la suivante:

```yaml
version: '2'

tasks:
  echo:
    cmds:
      - echo "Hello, World!"
```

La version 2 vous permet d'écrire des variables globales directement dans le fichier Taskfile, si vous ne voulez pas créer un fichier `Taskvars.yml`:

```yaml
version: '2'

vars:
  GREETING: Hello, World!

tasks:
  greet:
    cmds:
      - echo "{{.GREETING}}"
```

A présent, l'ordre de priorité des variables est :

1. Variables Task
2. Variables d'appel
3. Variables Taskfile
4. Variables du fichier Taskvars
5. Variables d'environnement

Une nouvelle option globale a été ajoutée pour configurer le nombre d'extensions de variables (par défaut 2):

```yaml
version: '2'

expansions: 3

vars:
  FOO: foo
  BAR: bar
  BAZ: baz
  FOOBAR: '{{.FOO}}{{.BAR}}'
  FOOBARBAZ: '{{.FOOBAR}}{{.BAZ}}'

tasks:
  default:
    cmds:
      - echo "{{.FOOBARBAZ}}"
```

## Version 1

:::caution

Le support du schéma v1 a été supprimé de Task >= v3.0.0.

:::

Dans la première version du `Taskfile`, le champ `version:` n'était pas disponible, parce que les tâches étaient à la racine du document YAML. Comme ceci:

```yaml
echo:
  cmds:
    - echo "Hello, World!"
```

L'ordre de priorité de la variable était également différent :

1. Variables d'appel
2. Variables d'environnement
3. Variables Task
4. Variables `Taskvars.yml`

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[output]: usage.md#output-syntax
[ignore_errors]: usage.md#ignore-errors
[includes]: usage.md#including-other-taskfiles
[deprecate-version-2-schema]: https://github.com/go-task/task/issues/1197
