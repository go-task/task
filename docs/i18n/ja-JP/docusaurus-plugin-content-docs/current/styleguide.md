---
slug: /styleguide/
sidebar_position: 10
---

# スタイルガイド

これは`Taskfile.yml`の公式Taskスタイルガイドです。 このガイドには、Taskfileを整理し、他ユーザーにとっても理解しやすいように保つための基本的な手順が含まれています。

これには一般的なガイドラインが含まれていますが、必ずしも厳密に従う必要はありません。 必要であったり、あるいは違う方法を取りたい場合は自由にしてください。 また、このガイドの改善点について、IssueまたはPull Requestを開くことも自由です。

## `taskfile.yml`ではなく`Taskfile.yml`を使用する

```yaml
# bad
taskfile.yml


# good
Taskfile.yml
```

これはLinuxユーザーにとって特に重要です。 WindowsとmacOSは大文字と小文字を区別しないファイルシステム持っているので、公式にはサポートされていないにもかかわらず、`taskfile.yml`は正常に動作します。 Linuxでは`Taskfile.yml`だけが動作します。

## キーワードを正しい順序にする

- `version:`
- `includes:`
- 設定項目として、`output:`、`silent:`、`method:`、そして`run:`があります。
- `vars:`
- `env:`, `dotenv:`
- `tasks:`

## インデントにはスペース2つを使用する

これはYAMLファイルの最も一般的な慣習であり、Taskはそれに倣うものです。

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

## メインセクションをスペースで分ける

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

## タスク間にスペースを設ける

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

## 変数名は大文字で定義する

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

## テンプレート記法で変数名の前後に空白を設けない

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

この書き方は多くの人によってGoのテンプレート作成にも使われています。

## タスク名の単語をハイフンで区切る

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

## タスクの名前空間にコロンを使用する

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

これはインクルードされたタスクファイルを使用する場合に、自動的に行われます。

## 複雑な複数行のコマンドの使用は避け、外部スクリプトを使用する

```yaml
# bad
version: '3'

tasks:
  build:
    cmds:
      - |
        for i in $(seq 1 10); do
          echo $i
          echo "some other complex logic"
        done'

# good
version: '3'

tasks:
  build:
    cmds:
      - ./scripts/my_complex_script.sh
```
