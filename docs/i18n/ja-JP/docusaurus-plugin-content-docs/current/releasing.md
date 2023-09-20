---
slug: /releasing/
sidebar_position: 13
---

# リリース

Taskのリリースプロセスは[GoReleaser][goreleaser]の助けを借りて行われます。 ローカルでリリースプロセスをテストするには、Taskfileの`test-release`タスクを呼び出します。

[GitHub Actions](https://github.com/go-task/task/actions)は、新しいGitタグが`main`ブランチにプッシュされると、自動的に成果物(実行ファイルおよびDEBとRPMパッケージ)をリリースするはずです。

v3.15.0以降、特定のタグをチェックアウトし、上記のGitHub Actionsで定義されたGoバージョンを使用して`goreleaser build`を呼び出すことで、実行ファイルをローカルでも再現して検証することができます。

# Homebrew

Goreleaser will automatically push a new commit to the [Formula/go-task.rb][gotaskrb] file in the [Homebrew tap][homebrewtap] repository to release the new version.

# npm

npmにリリースするには、[`package.json`][packagejson]でバージョンを更新し、`task npm:publish`を実行してプッシュします。

# Snapcraft

[snapパッケージ][snappackage]をリリースするには、マニュアルのステップが必要です:

- Updating the current version on [snapcraft.yaml][snapcraftyaml].
- [Snapcraftダッシュボード][snapcraftdashboard]で、`amd64`、`armhf`、`arm64`の新しい成果物を全てstableチャンネルに移動させる。

# winget

wingetもマニュアルのステップが必要です。 `task test-release`をローカルで実行すると、マニフェストファイルが`dist/winget/manifests/t/Task/Task/v{version}`に生成されます。 [マニフェストディレクトリをこのフォーク](https://github.com/go-task/winget-pkgs/tree/master/manifests/t/Task/Task)にアップロードし、[このリポジトリ](https://github.com/microsoft/winget-pkgs)にプルリクエストを作ってください。

# Scoop

ScoopはWindowsオペレーティングシステム用のコマンドラインパッケージマネージャーです。 Scoopパッケージマニフェストはコミュニティによって管理されています。 Scoopの所有者は通常、[このファイル](https://github.com/ScoopInstaller/Main/blob/master/bucket/task.json)を編集することでバージョンを更新します。 Taskのバージョンが古くなっていると思われる場合は、Issueを作成してお知らせください。

# Nix

Nixはコミュニティが所有するインストール方法です。 Nixパッケージのメンテナは通常、[このファイル](https://github.com/NixOS/nixpkgs/blob/nixos-unstable/pkgs/development/tools/go-task/default.nix)を編集してバージョンを更新します。 Taskのバージョンが古くなっていると思われる場合は、Issueを作成してお知らせください。

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[goreleaser]: https://goreleaser.com/
[homebrewtap]: https://github.com/go-task/homebrew-tap
[gotaskrb]: https://github.com/go-task/homebrew-tap/blob/main/Formula/go-task.rb
[packagejson]: https://github.com/go-task/task/blob/main/package.json#L3
[snappackage]: https://github.com/go-task/snap
[snapcraftyaml]: https://github.com/go-task/snap/blob/main/snap/snapcraft.yaml#L2
[snapcraftdashboard]: https://snapcraft.io/task/releases
