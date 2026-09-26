---
title: One Completion Engine for Every Shell
description:
  A new completion engine that gives every shell the same suggestions, and that
  can finally complete your required variables.
author: vmaerten
date: 2026-08-11
tags: ['new-features', 'completion']
outline: deep
editLink: false
---

# One Completion Engine for Every Shell

<AuthorCard :author="$frontmatter.author" />

Task has shipped shell completions for a long time. They were written as one
script per shell: a Bash script, a Zsh script, a Fish script, one for PowerShell
and one for Nushell. Each of them had to work out on its own what your Taskfile
contains and what flags Task accepts. Over the years, they drifted.

<!-- more -->

[Task v3.54.0][release] brings a shared completion engine to Bash, Zsh, Fish,
PowerShell and Nushell. It is the default behind `task --completion <shell>`.
Let me show you what it fixes.

## The same TAB, five different answers

With the old scripts, asking Task to complete the value of `--output` could
produce different answers depending on the shell:

```shell
# bash, zsh, fish
$ task --output <TAB>
interleaved  group  prefixed

# powershell
PS> task --output <TAB>
build  deploy
```

PowerShell had no idea `--output` takes a value, so it fell back to listing task
names where a value was expected.

The differences also affected which Taskfile was read:

```shell
$ task --dir ./sub <TAB>
```

The old Zsh, Fish and PowerShell scripts suggested tasks from the _current_
directory, since none of them passed `--dir` along when asking Task for a task
list. Bash and Nushell handled it correctly.

Behind these symptoms was one structural problem: the logic lived in the
scripts. Several scripts maintained their own lists of flags and descriptions,
so adding a flag meant updating each of them. Some also parsed the
human-readable output of `--list-all` to learn about your tasks. Here is part of
what the Fish completion had to do:

```fish
sed -e '1d; s/\* \(.*\):[[:space:]]\{2,\}\(.*\)[[:space:]]\{2,\}(\(aliases.*\))/\1\t\2\t\3/'
```

Even reformatting a line of `task --list-all` could break those completions.

## One engine instead of five scripts

The new scripts hand the command line over to a hidden `task __complete`
command. The binary returns suggestions and instructions for the shell, such as
whether to offer files or preserve the order of the results.

Each wrapper still handles the details of its shell: reading the words under the
cursor, quoting values and displaying suggestions. Task names, required
variables and flags now come from one implementation in the binary.

## What this unlocks

### Completing required variables

This is the one I wanted for a long time. The old scripts did not suggest a
task's [required variables][requires] or their allowed values. The engine does:

```yaml
version: '3'

tasks:
  deploy:
    desc: Deploy the app
    requires:
      vars:
        - name: ENVIRONMENT
          enum: [dev, staging, prod]
        - REGION
    cmds:
      - ./deploy.sh
```

```shell
$ task deploy <TAB>
ENVIRONMENT=dev  ENVIRONMENT=staging  ENVIRONMENT=prod  REGION=
```

Variables with an `enum` give you the allowed values directly. Variables without
one give you `VAR=`. The engine returns these suggestions in the order of your
`requires` block.

Assignments already on the command line are left out of the suggestions:

```shell
$ task deploy ENVIRONMENT=prod <TAB>
REGION=
```

Once both variables are supplied, completion offers task names again so you can
add another task to the same command. When several tasks are already named, the
engine combines their required variables without suggesting a name twice.

Enums defined by reference work too:

```yaml
version: '3'

vars:
  ALLOWED_ENVS: [dev, staging, prod]

tasks:
  deploy:
    requires:
      vars:
        - name: ENVIRONMENT
          enum:
            ref: .ALLOWED_ENVS
```

This uses the same enum resolver as the [interactive prompt][prompt-post], with
one important limit: completion does not run `sh:` variables. If an enum needs a
shell command to produce its values, completion offers `ENVIRONMENT=` instead.

### Everything else, everywhere

All five shells now get their task names, aliases, flags and flag values from
the engine. `--dir`, `--taskfile` and `--global` are honoured when loading the
Taskfile, so completing inside another directory works. Completing `--taskfile`
offers `.yml` and `.yaml` files, plus directories you can navigate into.

Flags gated behind an [experiment][experiments] are another good example. They
are read from Task's own flag set at the moment you press TAB, so they show up
when the experiment is enabled. The wrappers no longer need separate flag lists
or their own checks of `task --experiments`.

The suggestions are shared, but the shells still control how they are inserted
and displayed. For example, PowerShell can add a space after `VAR=`, older Bash
versions may sort the results, and Nushell does not append a trailing space.

## Completions should never surprise you

Completion runs on a keystroke, so it is allowed to do very little:

- No remote Taskfile downloads. Completion uses local files and cached remote
  Taskfiles. If an uncached include prevents the Taskfile from loading, task
  suggestions are unavailable until it has been fetched by a normal Task run.
- Nothing is read from stdin. A Taskfile supplied with `--taskfile -` is skipped
  during completion.
- No `sh:` variable is evaluated. Pressing TAB never runs a command from your
  Taskfile.
- A broken or missing Taskfile still leaves you with flag completion.

## Load the new completions

Starting with Task v3.54.0, `--completion` generates the new wrappers. If your
shell configuration already runs that command at startup, restart your shell
after upgrading. If you saved a completion script to a file, regenerate it with
the updated binary.

Here are the startup commands for each shell:

::: code-group

```shell [bash]
# ~/.bashrc
eval "$(task --completion bash)"
```

```shell [zsh]
# ~/.zshrc
eval "$(task --completion zsh)"
```

```shell [fish]
# ~/.config/fish/config.fish
task --completion fish | source
```

```powershell [powershell]
# Add to your PowerShell profile ($PROFILE)
Invoke-Expression (&task --completion powershell | Out-String)
```

```nu [nushell]
# Add to config.nu; completions are loaded in the next shell
mkdir ($nu.data-dir | path join "vendor/autoload")
task --completion nu | save --force ($nu.data-dir | path join "vendor/autoload/task-completions.nu")
```

:::

The [installation docs][install] cover saving scripts to completion directories,
Nushell's external completer and the Zsh `verbose` and `show-aliases` settings.

## Feedback

Please tell us how it behaves in your shell. If a suggestion is missing or
inserted incorrectly, include your Task version, shell version and a small
Taskfile that reproduces it. You can find us on our [Discord server][discord] or
[open an issue][gh-issue].

[release]: https://github.com/go-task/task/releases/tag/v3.54.0
[requires]: /docs/reference/schema#requires
[prompt-post]: /blog/if-and-variable-prompt
[experiments]: /docs/experiments/
[install]: /docs/installation#setup-completions
[discord]: https://discord.com/invite/6TY36E39UK
[gh-issue]: https://github.com/go-task/task/issues
