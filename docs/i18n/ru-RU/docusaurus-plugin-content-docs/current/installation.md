---
slug: /installation/
sidebar_position: 2
---

# Установка

Task имеет множество способов установки. Просмотрите доступные методы ниже.

## Менеджеры пакетов

### Homebrew

Если вы используете macOS или Linux и установили [Homebrew][homebrew], то для установки достаточно выполнить:

```bash
brew install go-task/tap/go-task
```

The above Formula is [maintained by ourselves](https://github.com/go-task/homebrew-tap/blob/main/Formula/go-task.rb).

Недавно Task стал доступен [в официальном репозитории Homebrew](https://formulae.brew.sh/formula/go-task), поэтому вы можете использовать:

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
Этот метод установки находится под управлением сообщества. After a new release of Task, they are automatically released by tea in a minimum of time.

### Snap

Task доступен в [Snapcraft][snapcraft], но имейте в виду, что ваш Linux дистрибутив должен иметь классическое ограничение для Snaps, чтобы Task работал правильно:

```bash
sudo snap install task --classic
```

### Chocolatey

Если вы используете Windows и у вас установлен [Chocolatey][choco], установка Task сводится к запуску:

```bash
choco install go-task
```

Этот метод установки находится под управлением сообщества.

### Scoop

Если вы используете Windows и у вас установлен [Scoop][scoop], установка Task сводится к запуску:

```cmd
scoop install task
```

This installation method is community owned. После нового релиза Task может потребоваться некоторое время, пока он станет доступен в Scoop.

### AUR

Если вы используете Arch Linux, вы можете установить Task из [AUR](https://aur.archlinux.org/packages/go-task-bin) с помощью вашего любимого менеджера пакетов, такого как `yay`, `pacaur` или `yaourt`:

```cmd
yay -S go-task-bin
```

В качестве альтернативы, можно использовать [пакет](https://aur.archlinux.org/packages/go-task), который устанавливается из исходного кода, а не загружает бинарный файл со [страницы релизов](https://github.com/go-task/task/releases):

```cmd
yay -S go-task
```

Этот метод установки находится под управлением сообщества.

### Fedora

Если вы используете Fedora Linux, вы можете установить Task из официального репозитория [Fedora](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/), используя `dnf`:

```cmd
sudo dnf install go-task
```

Этот метод установки находится под управлением сообщества. После нового релиза Task может потребоваться некоторое время, пока он станет доступен в [Fedora](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/).

### Nix

Если вы используете NixOS или у вас установлен Nix, вы можете установить Task из [nixpkgs](https://github.com/NixOS/nixpkgs):

```cmd
nix-env -iA nixpkgs.go-task
```

Этот метод установки находится под управлением сообщества. После нового релиза Task может потребоваться некоторое время, пока он станет доступен в [nixpkgs](https://github.com/NixOS/nixpkgs).

### npm

Для установки Task вы также можете использовать Node и npm, установив [этот пакет](https://www.npmjs.com/package/@go-task/cli).

```bash
npm install -g @go-task/cli
```

### Winget

Если вы используете Windows и установили менеджер пакетов [winget](https://github.com/microsoft/winget-cli), вы можете установить Task из [winget-pkgs](https://github.com/microsoft/winget-pkgs).

```bash
winget install Task.Task
```

## Установка бинарных файлов

### Бинарные

Вы можете установить бинарные файлы со [страницы релизов на GitHub][releases] и добавить их в ваш `$PATH`.

Также доступны DEB и RPM пакеты.

Файл `task_checksums.txt` содержит контрольные суммы SHA-256 для каждого файла.

### Скрипт для установки

У нас также есть [скрипт для установки][installscript], который очень полезен в некоторых случаях, таких как CI. Благодарим [GoDownloader][godownloader] за то, что он облегчает генерацию этого скрипта.

По умолчанию он устанавливается в каталог `./bin` относительно рабочего каталога:

```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d
```

Вы можете переопределить каталог установки с помощью параметра `-b`. В Linux распространенными вариантами являются `~/.local/bin` и `~/bin`, чтобы установить для текущего пользователя, или `/usr/local/bin`, чтобы установить для всех пользователей:

```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
```

:::caution

В macOS и Windows, `~/.local/bin` и `~/bin` по умолчанию не добавляются в `$PATH`.

:::

### GitHub Actions

Если вы хотите установить Task в GitHub Actions, вы можете попробовать использовать [это действие](https://github.com/arduino/setup-task) от команды Arduino:

```yaml
- name: Install Task
  uses: arduino/setup-task@v1
  with:
    version: 3.x
    repo-token: ${{ secrets.GITHUB_TOKEN }}
```

This installation method is community owned.

## Сборка из исходного кода

### Go Modules

Убедитесь, что у вас правильно установлена и настроена поддерживаемая версия [Go][go]. Вы можете найти минимальную требуемую версию Go в [go.mod](https://github.com/go-task/task/blob/main/go.mod#L3) файле.

Затем вы можете установить последнюю версию глобально, запустив:

```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

Или вы можете установить в другую директорию:

```bash
env GOBIN=/bin go install github.com/go-task/task/v3/cmd/task@latest
```

:::tip

Для окружения CI мы рекомендуем вместо этого использовать [скрипт установки](#install-script), который быстрее и стабильнее, так как он просто загрузит последний выпущенный бинарный файл.

:::

## Завершение установки

Загрузите файл автодополнения, соответствующий вашей оболочке.

[Все дополнения доступны в репозитории Task](https://github.com/go-task/task/tree/main/completion).

### Bash

Сначала убедитесь, что вы установили bash-completion с помощью вашего менеджера пакетов.

Сделайте файл автодополнения исполняемым:

```
chmod +x path/to/task.bash
```

Затем, добавьте это в свой `~/.bash_profile`:

```shell
source path/to/task.bash
```

### ZSH

Поместите файл `_task` куда-нибудь в ваш `$FPATH`:

```shell
mv path/to/_task /usr/local/share/zsh/site-functions/_task
```

Убедитесь, что в `~/.zshrc` присутствует следующее:

```shell
autoload -U compinit
compinit -i
```

Рекомендуется использовать ZSH версии 5.7 или выше.

### Fish

Переместите скрипт автодополнения `task.fish`:

```shell
mv path/to/task.fish ~/.config/fish/completions/task.fish
```

### PowerShell

Откройте сценарии вашего профиля с помощью:

```
mkdir -Path (Split-Path -Parent $profile) -ErrorAction SilentlyContinue
notepad $profile
```

Добавьте строку и сохраните файл:

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
