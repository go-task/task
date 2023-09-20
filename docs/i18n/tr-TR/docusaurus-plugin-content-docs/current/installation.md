---
slug: /installation/
sidebar_position: 2
---

# Kurulum

Task birçok kurulum yöntemi sunar. Aşağıdaki mevcut yöntemlere göz atın.

## Paket Yöneticileri

### Homebrew

Eğer macOS veya Linux kullanıyorsanız ve [Homebrew][homebrew] yüklüyse, Task'ı yüklemek şunu çalıştırmak kadar basittir:

```bash
brew install go-task/tap/go-task
```

The above Formula is [maintained by ourselves](https://github.com/go-task/homebrew-tap/blob/main/Formula/go-task.rb).

Son zamanlarda, Task [resmi Homebrew deposunda](https://formulae.brew.sh/formula/go-task) da kullanıma sunuldu, bu nedenle tercih ederseniz bu seçeneğe de sahipsiniz:

```bash
brew install go-task
```

### Tea

Eğer macOS veya Linux kullanıyorsanız ve [tea][tea] yüklüyse, Task'ı yüklemek şunu çalıştırmak kadar basittir:

```bash
tea task
```

veya tea'nin magic'ini etkinleştirdiyseniz:

```bash
task
```
Bu yükleme yöntemi topluluğa aittir. Task'ın yeni bir sürümünden sonra, tea'da kullanılabilir olana kadar biraz zaman ayırın.

### Snap

Task, [Snapcraft][snapcraft]'ta mevcuttur, ancak Linux'unuzun dağıtım, Snaps to Task için klasik sınırlandırmanın doğru çalışmasına izin vermelidir:

```bash
sudo snap install task --classic
```

### Chocolatey

Windows kullanıyorsanız ve [Chocolatey][choco] yüklüyse, Task'ı yüklemek şunu çalıştırmak kadar basittir:

```bash
choco install go-task
```

Bu yükleme yöntemi topluluğa aittir.

### Scoop

Windows kullanıyorsanız ve [Scoop][scoop] yüklüyse, Task'ı yüklemek şunu çalıştırmak kadar basittir:

```cmd
scoop install task
```

Bu yükleme yöntemi topluluğa aittir. Task'ın yeni bir sürümünden sonra, Scoop'ta kullanılabilir olana kadar biraz zaman ayırın.

### AUR

Arch Linux kullanıyorsanız `yay`, `pacaur` veya `yaourt` gibi favori paket yöneticinizi kullanarak Task'ı [AUR](https://aur.archlinux.org/packages/go-task-bin)'dan yükleyebilirsiniz:

```cmd
yay -S go-task-bin
```

Alternatif olarak, şu adresten yüklenen [bu paket](https://aur.archlinux.org/packages/go-task) var: [sürümler sayfasından](https://github.com/go-task/task/releases) derlenmiş dosyayı indirmek yerine kaynak kodu:

```cmd
yay -S go-task
```

Bu yükleme yöntemi topluluğa aittir.

### Fedora

Fedora kullanıyorsanız Task'ı `dnf` kullanarak [Fedora](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/)'nın resmi deposundan yükleyebilirsiniz:

```cmd
sudo dnf install go-task
```

Bu yükleme yöntemi topluluğa aittir. Task'ın yeni bir sürümünden sonra, [Fedora](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/)'da kullanılabilir olana kadar biraz zaman ayırın.

### Nix

NixOS kullanıyorsanız veya Nix yüklüyse Task'ı [nixpkgs](https://github.com/NixOS/nixpkgs)'den yükleyebilirsiniz:

```cmd
nix-env -iA nixpkgs.go-task
```

Bu yükleme yöntemi topluluğa aittir. Task'ın yeni bir sürümünden sonra, [nixpkgs](https://github.com/NixOS/nixpkgs)'te kullanılabilir olana kadar biraz zaman ayırın.

### npm

[Bu paketi](https://www.npmjs.com/package/@go-task/cli) yükleyerek Task'ı yüklemek için Node ve npm komutlarını da kullanabilirsiniz.

```bash
npm install -g @go-task/cli
```

### Winget

Windows kullanıyorsanız ve [winget](https://github.com/microsoft/winget-cli) paket yönetim aracını kurduysanız, [winget-pkgs](https://github.com/microsoft/winget-pkgs)'den Task'ı kurabilirsiniz.

```bash
winget install Task.Task
```

## Derlenmiş Dosyayı Yükleyin

### Derlenmiş Dosya

Derlenmiş dosyayı [GitHub'daki sürümler sayfasından][releases] indirebilir ve `$PATH`'nize ekleyebilirsiniz.

DEB ve RPM paketleri de mevcuttur.

`task_checksums.txt` dosyası, her dosya için SHA-256 doğrulamalarını içerir.

### Script'i Yükleyin

Ayrıca CI gibi senaryolarda çok yararlı olan bir [yükleme script][installscript]'imiz var. Bu komut dosyasının kolayca oluşturulmasını sağladığı için [GoDownloader][godownloader]'a çok teşekkürler.

Varsayılan olarak, çalışma dizinine göre `./bin` dizinine kurulur:

```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d
```

Kurulum dizinini `-b` parametresiyle geçersiz kılmak mümkündür. Linux'ta, geçerli kullanıcı için `~/.local/bin` ve `~/bin ` yüklemek veya tüm kullanıcılar için yüklemek için `/usr/local/bin` yaygın seçeneklerdir:

```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
```

:::caution

macOS ve Windows'ta `~/.local/bin` ve `~/bin` varsayılan olarak `$PATH`'e eklenmez.

:::

### GitHub Actions

Task'ı GitHub Actions'a yüklemek istiyorsanız, Arduino ekibi tarafından bu [action](https://github.com/arduino/setup-task)'u kullanmayı deneyebilirsiniz:

```yaml
- name: Install Task
  uses: arduino/setup-task@v1
  with:
    version: 3.x
    repo-token: ${{ secrets.GITHUB_TOKEN }}
```

Bu yükleme yöntemi topluluğa aittir.

## Kaynaktan Oluştur

### Go Modülleri

[Go][go]'nun desteklenen bir sürümünün düzgün şekilde yüklendiğinden ve ayarlandığından emin olun. Go'nun gerekli minimum sürümünü [go.mod](https://github.com/go-task/task/blob/main/go.mod#L3) dosyasında bulabilirsiniz.

Aşağıdakileri çalıştırarak en son sürümü global olarak yükleyebilirsiniz:

```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

Veya başka bir dizine kurabilirsiniz:

```bash
env GOBIN=/bin go install github.com/go-task/task/v3/cmd/task@latest
```

:::tip

CI ortamları için bunun yerine, yalnızca en son yayınlanan derlenmiş dosyayı indireceğinden daha hızlı ve daha kararlı olan [yükleme script](#install-script)'ini kullanmanızı öneririz.

:::

## Tamamlamaları Kurma

Shell'inize karşılık gelen otomatik tamamlama dosyasını indirin.

[Tüm tamamlamalar, Task'ın deposunda mevcuttur](https://github.com/go-task/task/tree/main/completion).

### Bash

Öncelikle, paket yöneticinizi kullanarak bash-completion'ı kurduğunuzdan emin olun.

Tamamlama dosyasını çalıştırılabilir yapın:

```
chmod +x path/to/task.bash
```

Ardından, bunu `~/.bash_profile` ekleyin:

```shell
source path/to/task.bash
```

### ZSH

`_task` dosyasını `$FPATH` içinde bir yere koyun:

```shell
mv path/to/_task /usr/local/share/zsh/site-functions/_task
```

`~/.zshrc` dosyanızda aşağıdakilerin bulunduğundan emin olun:

```shell
autoload -U compinit
compinit -i
```

ZSH'nin sürüm 5.7 veya üstü önerilir.

### Fish

`task.fish` tamamlama script'ini taşıyın:

```shell
mv path/to/task.fish ~/.config/fish/completions/task.fish
```

### PowerShell

Profil script'inizi aşağıdakilerle açın:

```
mkdir -Path (Split-Path -Parent $profile) -ErrorAction SilentlyContinue
notepad $profile
```

Aşağıdaki satırı ekleyin ve dosyayı kaydedin:

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
