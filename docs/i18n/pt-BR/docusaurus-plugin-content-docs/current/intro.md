---
slug: /
sidebar_position: 1
title: Página Inicial
---

# Task

<div align="center">
  <img id="logo" src="img/logo.svg" height="250px" width="250px" />
</div>

Task é uma ferramenta de automatização que foi criada para ser mais simples de usar do que outras similares, como por exemplo o [GNU Make][make].

Por ser escrito em [Go][go], o Task é simplesmente um binário e não possui nenhuma outra dependência, o que significa que você não precisa lidar com um processo de instalação complicado apenas para usar uma ferramenta de automação.

Uma vez [instalado](installation.md), você só precisa só precisa escrever suas tarefas usando um esquema [YAML][yaml] simples num arquivo chamado `Taskfile.yml`:

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

## Features

- [Instalação fácil](installation.md): apenas baixe um único binário, adicione-o a `$PATH` e pronto! Ou você também pode instalá-lo usando [Homebrew][homebrew], [Snapcraft][snapcraft] ou [Scoop][scoop] se você quiser.
- Disponível em CIs: adicionando [este script simples](installation.md#install-script) para instalá-lo no seu CI você estará pronto para usar o Task como parte do seu pipeline de CI;
- Verdadeiramente multiplataforma: enquanto a maioria das ferramentas de compilação só funcionam bem no Linux ou macOS, o Task também suporta Windows graças [a este interpretador de shell para Go][sh].
- Ótimo para a geração de código: você pode facilmente [impedir que uma tarefa execute](/usage#prevent-unnecessary-work) se um determinado conjunto de arquivos não tiver mudado desde a última execução (baseado na data de modificação ou conteúdo dos arquivos).

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[make]: https://www.gnu.org/software/make/
[go]: https://go.dev/
[yaml]: http://yaml.org/
[homebrew]: https://brew.sh/
[snapcraft]: https://snapcraft.io/
[scoop]: https://scoop.sh/
[sh]: https://github.com/mvdan/sh
