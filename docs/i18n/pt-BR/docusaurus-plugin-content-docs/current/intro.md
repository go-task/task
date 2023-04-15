---
slug: /
sidebar_position: 1
title: Página Inicial
---

# Task

<div align="center">
  <img id="logo" src="img/logo.svg" height="250px" width="250px" />
</div>

Task is a task runner / build tool that aims to be simpler and easier to use than, for example, [GNU Make](https://www.gnu.org/software/make/).

Por ser escrito em [Go](https://go.dev/), o Task é simplesmente um binário e não possui nenhuma outra dependência, o que significa que você não precisa lidar com um processo de instalação complicado apenas para usar uma ferramenta de automação.

Uma vez [instalado](installation.md), você só precisa só precisa escrever suas tarefas usando um esquema [YAML](http://yaml.org/) simples num arquivo chamado `Taskfile.yml`:

```yaml title="Taskfile.yml"
version: '3'

tasks:
  hello:
    cmds:
      - echo 'Hello World from Task!'
    silent: true
```

E invocá-lo ao rodar `task hello` do seu terminal.

O exemplo acima é apenas o começo. Você pode dar uma olhada no [guia de uso](/usage) para conferir a documentação completa do esquema e as funcionalidades do Task.

## Funcionalidades

- [Instalação fácil](installation.md): apenas baixe um único binário, adicione-o a `$PATH` e pronto! Ou você também pode instalá-lo usando [Homebrew](https://brew.sh/), [Snapcraft](https://snapcraft.io/) ou [Scoop](https://scoop.sh/) se você quiser.
- Available on CIs: by adding [this simple command](installation.md#install-script) to install on your CI script and you're ready to use Task as part of your CI pipeline;
- Verdadeiramente multiplataforma: enquanto a maioria das ferramentas de compilação só funcionam bem no Linux ou macOS, o Task também suporta Windows graças [a este interpretador de shell para Go](https://github.com/mvdan/sh).
- Great for code generation: you can easily [prevent a task from running](/usage#prevent-unnecessary-work) if a given set of files haven't changed since last run (based either on its timestamp or content).

## Patrocinadores de Ouro

<div class="gold-sponsors">

| [Appwrite](https://appwrite.io/?utm_source=taskfile.dev&utm_medium=website&utm_campaign=task_oss_fund)                       |
| ---------------------------------------------------------------------------------------------------------------------------- |
| [![Appwrite](/img/appwrite.svg)](https://appwrite.io/?utm_source=taskfile.dev&utm_medium=website&utm_campaign=task_oss_fund) |

</div>

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
