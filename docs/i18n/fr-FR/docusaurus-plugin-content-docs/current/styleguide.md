---
slug: /styleguide/
sidebar_position: 10
---

# Guide de style

Ceci est le guide officiel du style Task pour les fichiers `Taskfile.yml`. Ce guide contient quelques instructions de base pour garder votre Taskfile propre et familier à autres utilisateurs.

Il contient des directives générales, mais elles ne doivent pas nécessairement être strictement respectées. N'hésitez pas à procéder différemment si vous en avez le besoin ou que vous le souhaitez. Aussi, n'hésitez pas à [ouvrir une issue](https://github. com/go-task/task/issues/new/choose) ou [faire une pull request](https://github. com/go-task/task/compare) pour améliorer ce guide.

## Utiliser `Taskfile.yml` et non `taskfile.yml`

```yaml
# bad
taskfile.yml


# good
Taskfile.yml
```

C'est important, surtout pour les utilisateurs Linux. Windows et MacOS ont un système de fichiers insensibles à la casse, donc `taskfile.yml` fonctionnera, même si ce n'est pas officiellement supporté. Sur Linux, uniquement `Taskfile.yml` fonctionnera.

## Utiliser les mots-clés dans l'ordre correct

- `version:`
- `includes:`
- Configuration ones, like `output:`, `silent:`, `method:` and `run:`
- `vars:`
- `env:`, `dotenv:`
- `tasks:`

## Utiliser 2 espaces pour l'indentation

C'est la convention la plus courante pour les fichiers YAML et Task suit cette convention.

```yaml
# bad
tasks:
    foo:
        cmds:
            - echo 'foo'


# good
tasks:
  foo:
    cmds:
      - echo 'foo'
```

## Séparer les sections principales avec un retour à la ligne

```yaml
# bad
version: '3'
includes:
  docker: ./docker/Taskfile.yml
output: prefixed
vars:
  FOO: bar
env:
  BAR: baz
tasks:
  # ...


# good
version: '3'

includes:
  docker: ./docker/Taskfile.yml

output: prefixed

vars:
  FOO: bar

env:
  BAR: baz

tasks:
  # ...
```

## Ajouter des retours à la ligne entre les tâches

```yaml
# bad
version: '3'

tasks:
  foo:
    cmds:
      - echo 'foo'
  bar:
    cmds:
      - echo 'bar'
  baz:
    cmds:
      - echo 'baz'


# good
version: '3'

tasks:
  foo:
    cmds:
      - echo 'foo'

  bar:
    cmds:
      - echo 'bar'

  baz:
    cmds:
      - echo 'baz'
```

## Utiliser des noms de variables en majuscule

```yaml
# bad
version: '3'

vars:
  binary_name: myapp

tasks:
  build:
    cmds:
      - go build -o {{.binary_name}} .


# good
version: '3'

vars:
  BINARY_NAME: myapp

tasks:
  build:
    cmds:
      - go build -o {{.BINARY_NAME}} .
```

## Ne pas mettre d'espaces autour des variables lors de l'utilisation

```yaml
# bad
version: '3'

tasks:
  greet:
    cmds:
      - echo '{{ .MESSAGE }}'


# good
version: '3'

tasks:
  greet:
    cmds:
      - echo '{{.MESSAGE}}'
```

Cette convention est aussi utilisée par la plupart des gens pour n'importe quel modèle Go.

## Séparer les mots du nom de la tâche par un tiret

```yaml
# bad
version: '3'

tasks:
  do_something_fancy:
    cmds:
      - echo 'Do something'


# good
version: '3'

tasks:
  do-something-fancy:
    cmds:
      - echo 'Do something'
```

## Utiliser les deux-points pour nommer les namespaces de tâche

```yaml
# good
version: '3'

tasks:
  docker:build:
    cmds:
      - docker ...

  docker:run:
    cmds:
      - docker-compose ...
```

C'est aussi fait automatiquement quand vous incluez des Taskfiles.

## Préférer les scripts externes à des commandes complexes à plusieurs lignes

```yaml
# Incorrect
version: '3'

tasks:
  build:
    cmds:
      - |
        for i in $(seq 1 10); do
          echo $i
          echo "some other complex logic"
        done'

# Correct
version: '3'

tasks:
  build:
    cmds:
      - ./scripts/my_complex_script.sh
```
