---
slug: /installation/
sidebar_position: 2
---

# Instalação

Há muitas maneiras possíveis de se instalar o Task. Confira os métodos disponíveis abaixo.

## Gerenciador de Pacotes

### Homebrew

Se você estiver no macOS ou Linux e tiver o [Homebrew][homebrew] instalado, instalar o Task é tão simples quanto rodar:

```bash
brew install go-task/tap/go-task
```

The above Formula is [maintained by ourselves](https://github.com/go-task/homebrew-tap/blob/main/Formula/go-task.rb).

Recentemente, o Task também foi disponibilizado [no repositório oficial do Homebrew](https://formulae.brew.sh/formula/go-task), então você também tem essa opção, se preferir:

```bash
brew install go-task
```

### Tea

If you're on macOS or Linux and have [tea][tea] installed, getting Task is as simple as running:

```bash
tea task
```

or, if you have tea’s magic enabled:

```bash
task
```
Este método de instalação é mantido pela comunidade. After a new release of Task, they are automatically released by tea in a minimum of time.

### Snap

O Task está disponível no [Snapcraft][snapcraft], mas tenha em mente que a sua distribuição Linux deve suportar confinamento clássico (*classic confinement*) para Snaps para o Task funcionar corretamente:

```bash
sudo snap install task --classic
```

### Chocolatey

Se você estiver no Windows e tiver o [Chocolatey][choco] instalado, instalar o Task é tão simples quanto rodar:

```bash
choco install go-task
```

Este método de instalação é mantido pela comunidade.

### Scoop

Se você está no Windows e tem o [Scoop][scoop] instalado, instalar o Task é tão simples quanto rodar:

```cmd
scoop install task
```

This installation method is community owned. Após o lançamento de uma nova versão do Task, pode levar algum tempo até que esteja disponível no Scoop.

### AUR

Se você estiver no Arch Linux, você pode instalar o Task a partir do [AUR](https://aur.archlinux.org/packages/go-task-bin) usando o seu gerenciador de pacotes favorito, como `yay`, `pacauro` ou `yaourt`:

```cmd
yay -S go-task-bin
```

Alternativamente, há [este pacote](https://aur.archlinux.org/packages/go-task) que instala do código fonte ao invés de baixar o binário [do GitHub](https://github.com/go-task/task/releases):

```cmd
yay -S go-task
```

Este método de instalação é mantido pela comunidade.

### Fedora

Se você estiver no Fedora Linux, você pode instalar o Task do [repositório oficial do Fedora](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/) usando `dnf`:

```cmd
sudo dnf install go-task
```

Este método de instalação é mantido pela comunidade. Após o lançamento de uma nova versão do Task, pode levar algum tempo até que ela esteja disponível no [Fedora](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/).

### Nix

Se você estiver no NixOS ou tiver o Nix instalado, você pode instalar o Task a partir do [nixpkgs](https://github.com/NixOS/nixpkgs):

```cmd
nix-env -iA nixpkgs.go-task
```

Este método de instalação é mantido pela comunidade. Após o lançamento de uma nova versão do Task, pode levar algum tempo até que ela esteja disponível em [nixpkgs](https://github.com/NixOS/nixpkgs).

### npm

Você também pode usar o Node e o npm para instalar o Task instalando [este pacote](https://www.npmjs.com/package/@go-task/cli).

```bash
npm install -g @go-task/cli
```

### Winget

Se você estiver usando o Windows e instalando a ferramenta de gerenciamento de pacotes [winget](https://github.com/microsoft/winget-cli), você pode instalar o Task a partir de [winget-pkgs](https://github.com/microsoft/winget-pkgs).

```bash
winget install Task.Task
```

## Baixe o Binário

### Binário

Você pode baixar o binário da [página de versões no GitHub][releases] e adicionar a sua variável de ambiente `$PATH`.

Os pacotes DEB e RPM também estão disponíveis.

O arquivo `task_checksums.txt` contém a *checksum* SHA-256 para cada arquivo.

### Script de instalação

Também temos um [script de instalação][installscript] que é muito útil em cenários, como CI. Muito obrigado ao [GoDownloader][godownloader] por permitir a geração fácil deste script.

Por padrão, o binário será baixado no diretório `./bin` em relação ao diretório atual:

```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d
```

É possível substituir o diretório de instalação com o parâmetro `-b`. No Linux, escolhas comuns são `~/.local/bin` e `~/bin` para instalar para o usuário ou `/usr/local/bin` para instalar para todos os usuários:

```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
```

:::caution

No macOS e Windows, `~/.local/bin` e `~/bin` não são adicionados ao `$PATH` por padrão.

:::

### GitHub Actions

Se você quiser instalar o Task no GitHub Actions você pode tentar usar [esta *action*](https://github.com/arduino/setup-task) pela equipe do Arduino:

```yaml
- name: Install Task
  uses: arduino/setup-task@v1
  with:
    version: 3.x
    repo-token: ${{ secrets.GITHUB_TOKEN }}
```

This installation method is community owned.

## Compilar do código-fonte

### Go Modules

Certifique-se de que você tem uma versão suportada do [Go][go] corretamente instalado e configurado. Você pode encontrar a versão mínima necessária do Go no arquivo [go.mod](https://github.com/go-task/task/blob/main/go.mod#L3).

Você pode então instalar a última versão globalmente ao rodar:

```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

Ou você pode instalar em outro diretório rodando:

```bash
env GOBIN=/bin go install github.com/go-task/task/v3/cmd/task@latest
```

:::tip

Para ambientes com CI, recomendamos usar o [script de instalação](#install-script), que é mais rápido e mais estável, já que ele apenas baixará o último binário lançado.

:::

## Configurar *auto-complete* de terminal

Baixe o arquivo de *auto-completion* correspondente ao seu terminal.

[Todos os scripts de *completion* estão disponíveis no repositório do Task](https://github.com/go-task/task/tree/main/completion).

### Bash

Primeiro, certifique-se de que você instalou o *bash-completion* usando seu gerenciador de pacotes.

Torne o arquivo de *completion* executável:

```
chmod +x path/to/task.bash
```

Depois, adicione isto ao seu `~/.bash_profile`:

```shell
source path/to/task.bash
```

### ZSH

Coloque o arquivo `_task` em algum lugar no seu `$FPATH`:

```shell
mv path/to/_task /usr/local/share/zsh/site-functions/_task
```

Certifique-se de que o seguinte esteja presente em seu `~/.zshrc`:

```shell
autoload -U compinit
compinit -i
```

Recomenda-se ZSH versão 5.7 ou posterior.

### Fish

Mova o script de *completion* `task.fish`:

```shell
mv path/to/task.fish ~/.config/fish/completions/task.fish
```

### PowerShell

Abra seu *profile script* rodando:

```
mkdir -Path (Split-Path -Parent $profile) -ErrorAction SilentlyContinue
notepad $profile
```

Adicione a seguinte linha e salve o arquivo:

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
