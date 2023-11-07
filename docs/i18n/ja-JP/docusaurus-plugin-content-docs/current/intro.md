---
slug: /
sidebar_position: 1
title: ホーム
---

# Task

<div align="center">
  <img id="logo" src="img/logo.svg" height="250px" width="250px" />
</div>

Taskは[GNU Make][make]のようにシンプルで簡単に使えるタスクランナー・ビルドツールです。

[Go][go]で書かれているため、Taskはシングルバイナリで他の依存関係がありません。つまり、複雑なインストールセットアップがありません。

一度[インストール](installation.md)したら、シンプルな[YAML][yaml]スキーマを利用して、`Taskfile.yml`というファイルにビルドタスクを記述するだけでいいです。

```yaml title="Taskfile.yml"
version: '3'

tasks:
  hello:
    cmds:
      - echo 'Hello World from Task!'
    silent: true
```

記述後はターミナル上で`task hello`と実行することでそのタスクが実行されます。

上記の例は始まりに過ぎません。 全てのスキーマやTaskの機能については、[usage](/usage)ガイドを確認するといいでしょう。

## 特徴

- [簡単なインストール方法](installation.md): シングルバイナリをダウンロードして、`$PATH`に追加するだけで完了です！ または[Homebrew][homebrew]、[Snapcraft][snapcraft]、[Scoop][scoop]を使ってインストールすることができます。
- Clで使用可能: [シンプルなコマンド](installation.md#install-script)でCIスクリプトに追加することでCIパイプラインでTaskを使うことができます。
- 真のクロスプラットフォーム: ほとんどのビルドツールはLinuxまたはmacOSだけで使用可能ですが、Taskは[Goのシェルインタープリタ][sh]を使うことでWindowsもサポートしています。
- コード生成に適している: 特定のファイル群が最後に実行されてから変更されていない場合(タイムスタンプや内容に基づき)、簡単に[タスクの実行を防ぐ](/usage#prevent-unnecessary-work)ことができます。

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[make]: https://www.gnu.org/software/make/
[go]: https://go.dev/
[yaml]: http://yaml.org/
[homebrew]: https://brew.sh/
[snapcraft]: https://snapcraft.io/
[scoop]: https://scoop.sh/
[sh]: https://github.com/mvdan/sh
