---
slug: /faq/
sidebar_position: 15
---

# FAQ

このページはTaskに関するよくある質問についてまとめたものです。

## タスクがシェル環境を更新しないのはなぜですか？

これはシェルの仕組みの制限です。 Taskは現在のシェルのサブプロセスとして実行されるため、それを起動したシェルの環境を変更することができません。 この制限は他のタスクランナーやビルドツールでも同様です。

これを回避する一般的な方法は、シェルが解析できる出力を生成するタスクを作成することです。 例えば、シェルの環境変数を設定するには以下のようなタスクを書くことができます:

```yaml
my-shell-env:
  cmds:
    - echo "export FOO=foo"
    - echo "export BAR=bar"
```

そして、`eval $(task my-shell-env)`を実行することで、変数`$FOO`と`$BAR`がシェルで使用可能になります。

## タスクのコマンドでシェルを再利用できません

Taskはそれぞれのコマンドを異なるシェルプロセスとして実行するため、一つのコマンドが他のコマンドには影響されることはありません。 例えば、以下は上手く動きません:

```yaml
version: '3'

tasks:
  foo:
    cmds:
      - a=foo
      - echo $a
      # outputs ""
```

これを上手く動かすためには複数行コマンドを利用します:

```yaml
version: '3'

tasks:
  foo:
    cmds:
      - |
        a=foo
        echo $a
      # outputs "foo"
```

もっと複雑なコマンドの場合は、ファイルを用意してそれを呼び出すようにすることを推奨します:

```yaml
version: '3'

tasks:
  foo:
    cmds:
      - ./foo-printer.bash
```

```bash
#!/bin/bash
a=foo
echo $a
```

## 'x'組み込みコマンドがWindowsで動作しません

Windowsのデフォルトシェル(`cmd`と`powershell`)は、組み込みシェルとして`rm`や`cp`がありません。 つまり、これらのコマンドは動作しません。 Taskfileをクロスプラットフォームにしたい場合は、次のいずれかの方法で制限を回避する必要があります:

- `{{OS}}`関数を使用して、OS固有のスクリプトを実行する。
- `{{if eq OS "windows"}}powershell {{end}}<my_cmd>`のようにWindowsかを判別して、Powershellを実行する。
- Windowsでこれらのコマンドを組み込みでサポートするシェルを使ってください。例えば[Git Bash][git-bash]または[WSL][wsl]などがあります。

私たちはTaskのこの部分に改善したいと思っており、以下のIssueがそれを追跡するものです。 建設的なコメントやコントリビュートは大歓迎です！

- [#197](https://github.com/go-task/task/issues/197)
- [mvdan/sh#93](https://github.com/mvdan/sh/issues/93)
- [mvdan/sh#97](https://github.com/mvdan/sh/issues/97)

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[git-bash]: https://gitforwindows.org/
[wsl]: https://learn.microsoft.com/en-us/windows/wsl/install
