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

Once [installed](/installation), you just need to describe your build tasks using a simple [YAML][yaml] schema in a file called `Taskfile.yml`:

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

- [Easy installation](/installation): just download a single binary, add to `$PATH` and you're done! または[Homebrew][homebrew]、[Snapcraft][snapcraft]、[Scoop][scoop]を使ってインストールすることができます。
- Available on CIs: by adding [this simple command](/installation#install-script) to install on your CI script and you're ready to use Task as part of your CI pipeline;
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
