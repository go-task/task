---
slug: /installation/
sidebar_position: 2
---

# インストール方法

Taskは多くのインストール方法を提供します。 以下にある利用可能な方法を確認してください。

## パッケージマネージャー

### Homebrew

macOSやLinuxを使っていて、[Homebrew][homebrew]をインストールしている場合は、以下を実行することで簡単にTaskをインストールできます:

```bash
brew install go-task/tap/go-task
```

上記のFormulaは、[私たちによってメンテナンスされています](https://github.com/go-task/homebrew-tap/blob/main/Formula/go-task.rb)。

最近、Taskは[公式のHomebrewリポジトリ](https://formulae.brew.sh/formula/go-task)でも利用可能になったので、以下の方法でもインストールできます:

```bash
brew install go-task
```

### Tea

macOSやLinuxを使っていて、[tea][tea]をインストールしている場合は、以下を実行することで簡単にTaskをインストールできます:

```bash
tea task
```

teaの魔法が有効になっている場合:

```bash
task
```
このインストール方法はコミュニティが所有しています。 Taskの新しいリリースが出来た後、自動的にteaが最小限の時間でリリースします。

### Snap

Taskは[ Snapcraft][snapcraft]で提供されていますが、Snapを適切に動作させるためには、あなたのLinux distributionがクラシックな制限を許可することを念頭に置いておいてください:

```bash
sudo snap install task --classic
```

### Chocolatey

Windowsを使っていて、[Chocolatey][choco]をインストールしていれば、以下を実行することで簡単にTaskをインストールできます:

```bash
choco install go-task
```

このインストール方法はコミュニティーが所有しています。

### Scoop

Windowsを使っていて、[Scoop][scoop]をインストールしていれば、以下を実行することで簡単にTaskをインストールできます:

```cmd
scoop install task
```

このインストール方法はコミュニティーが所有しています。 Taskの新しいリリースが出来た後、Scoopで利用可能になるには時間がかかるかもしれません。

### AUR

Arch Linuxを使っていれば、あなたの好きなパッケージマネージャ(`yay`、`pacaur`、または`yaourt`など)を使って[AUR](https://aur.archlinux.org/packages/go-task-bin)からTaskをインストールできます:

```cmd
yay -S go-task-bin
```

あるいは、[リリースページ](https://github.com/go-task/task/releases)からバイナリをダウンロードする代わりに、ソースコードからインストールする[パッケージ](https://aur.archlinux.org/packages/go-task)もあります：

```cmd
yay -S go-task
```

このインストール方法はコミュニティーが所有しています。

### Fedora

Fedora Linuxを使っている場合、`dnf`を使って、公式の[Fedora](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/)リポジトリからTaskをインストールできます:

```cmd
sudo dnf install go-task
```

このインストール方法はコミュニティーが所有しています。 Taskの新しいリリースが出来た後、[Fedora](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/)で利用可能になるには時間がかかるかもしれません。

### Nix

NixOSを使っている場合、またはNixをインストールしている場合、[nixpkgs](https://github.com/NixOS/nixpkgs)からTaskをインストールできます。

```cmd
nix-env -iA nixpkgs.go-task
```

このインストール方法はコミュニティーが所有しています。 Taskの新しいリリースが出来た後、[nixpkgs](https://github.com/NixOS/nixpkgs)で利用可能になるには時間がかかるかもしれません。

### npm

Nodeとnpmを使って[このパッケージ](https://www.npmjs.com/package/@go-task/cli)をインストールすることでTaskをインストールすることもできます。

```bash
npm install -g @go-task/cli
```

### Winget

Windowsを使っていて、[winget](https://github.com/microsoft/winget-cli)パッケージマネジメントツールをインストールしていれば、[winget-pkgs](https://github.com/microsoft/winget-pkgs)からTaskをインストールできます。

```bash
winget install Task.Task
```

## バイナリの取得

### バイナリ

[GitHubのリリースページ][releases]からバイナリをダウンロードして`$PATH`に追加することでインストールできます。

DEBとRPMパッケージも利用可能です。

`task_checksums.txt`ファイルには、各バイナリファイルのSHA-256チェックサムが記載されています。

### スクリプトを使ったインストール

[install script][installscript]もあり、CIのような場面で非常に有用です。 [GoDownloader][godownloader]のおかげで、スクリプトを簡単に生成することができました。

デフォルトでは作業ディレクトリに相対的な`./bin`ディレクトリにインストールされます:

```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d
```

`-b`パラメータでインストールディレクトリを上書きすることができます。 Linuxでは、現在のユーザー向けには`~/.local/bin`や`~/bin`、すべてのユーザー向けには`/usr/local/bin`をインストール先として選択することが一般的です:

```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
```

:::caution

macOSとWindowsでは`~/.local/bin`と`~/bin`は、`$PATH`にデフォルトで追加されていません。

:::

### GitHub Actions

GitHub ActionsでTaskをインストールしたい場合、Arduinoチームの[action](https://github.com/arduino/setup-task)を使用してみてください:

```yaml
- name: Install Task
  uses: arduino/setup-task@v1
  with:
    version: 3.x
    repo-token: ${{ secrets.GITHUB_TOKEN }}
```

このインストール方法はコミュニティーが所有しています。

## ソースコードからビルド

### Goモジュール

[Go][go]のサポート対象のバージョンが適切にインストールおよびセットアップされていることを確認してください。 Goの必要な最小バージョンは[go.mod](https://github.com/go-task/task/blob/main/go.mod#L3)ファイルから確認できます。

以下を実行することで最新のリリースをグローバルにインストールできます:

```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

別のディレクトリにインストールすることもできます:

```bash
env GOBIN=/bin go install github.com/go-task/task/v3/cmd/task@latest
```

:::tip

CI環境では、最新リリースのバイナリを早く、安定して提供する[install script](#install-script)を使うことをおすすめします。

:::

## 補完のセットアップ

使用しているシェルに対応した自動補完ファイルをダウンロードしてください。

[シェルに対応した自動補完ファイルはTaskリポジトリにあります](https://github.com/go-task/task/tree/main/completion)。

### Bash

まず、パッケージマネージャを使用してbash-completionがインストールされていることを確認します。

補完ファイルを実行可能にします:

```
chmod +x path/to/task.bash
```

その後、以下の行を`~/.bash_profile`に追加してください:

```shell
source path/to/task.bash
```

### ZSH

`_task`ファイルを`$FPATH`のどこかに置きます:

```shell
mv path/to/_task /usr/local/share/zsh/site-functions/_task
```

`~/.zshrc`に以下の内容が含まれていることを確認してください:

```shell
autoload -U compinit
compinit -i
```

ZSHバージョンは5.7以降をおすすめします。

### Fish

` task.fish`補完スクリプトを移動させます:

```shell
mv path/to/task.fish ~/.config/fish/completions/task.fish
```

### PowerShell

プロファイルスクリプトを開きます:

```
mkdir -Path (Split-Path -Parent $profile) -ErrorAction SilentlyContinue
notepad $profile
```

以下の行を追加してファイルを保存します:

```shell
Invoke-Expression -Command path/to/task.ps1
```

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[go]: https://golang.org/
[snapcraft]: https://snapcraft.io/task
[homebrew]: https://brew.sh/
[installscript]: https://github.com/go-task/task/blob/main/install-task.sh
[releases]: https://github.com/go-task/task/releases
[godownloader]: https://github.com/goreleaser/godownloader
[choco]: https://chocolatey.org/
[scoop]: https://scoop.sh/
[tea]: https://tea.xyz/
