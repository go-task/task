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

[v3.4X.0][release] ships a new completion engine that replaces all five with a
single source of truth. It is opt-in for now. Let me show you what it fixes.

## The same TAB, five different answers

Ask Task to complete the value of `--output` and the answer depends on where you
are typing:

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

Change directory and things get worse:

```shell
$ task --dir ./sub <TAB>
```

Zsh, Fish and PowerShell answer with the tasks of the _current_ directory, since
none of them looked at `--dir` before asking Task for a task list. Bash and
Nushell get it right.

Behind these symptoms was one structural problem: the logic lived in the
scripts. Every Task flag was hand-copied into five files, with five sets of
descriptions that slowly diverged, so adding a flag meant patching five scripts
and forgetting at least one. And to learn about your tasks, the scripts parsed
the output of `--list-all`, which is meant to be read by a human. Here is what
the Fish completion had to do:

```fish
sed -e '1d; s/\* \(.*\):[[:space:]]\{2,\}\(.*\)[[:space:]]\{2,\}(\(aliases.*\))/\1\t\2\t\3/'
```

Reformatting a line of `task --list-all` was a breaking change nobody could see
coming.

## One engine instead of five scripts

The scripts no longer decide anything. They hand the command line over to a new
hidden `task __complete` command and print back whatever it returns. All the
knowledge about your Taskfile lives in the binary now, in one place.

The protocol is the one [cobra][cobra] uses: one line per suggestion, followed
by a directive telling the shell what to do with the result, such as "do not add
a space after this" or "keep my order". Reusing it means we inherit years of
shell edge cases someone else already ran into, and a future move to cobra stays
cheap.

What is left in each wrapper is the part that genuinely differs between shells:
how to read the words under the cursor, and how to feed suggestions back. No
flag list, no `sed`, no regex over human output.

## What this unlocks

### Completing required variables

This is the one I wanted for a long time. A static script could never know that
a task has [required variables][requires], let alone which values they accept.
The engine does:

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
one give you `VAR=` with the cursor ready for typing. The order of your
`requires` block is preserved instead of being alphabetically sorted by the
shell.

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

That is the same resolution the interactive prompt uses, which I wrote about in
[a previous post][prompt-post]. Completion and prompting now agree on what a
valid value is.

### Everything else, everywhere

Since there is only one implementation left, a fix lands in every shell at once.
Flag values, aliases and flag descriptions are identical whichever shell you
type into. `--dir`, `--taskfile` and `--global` are honoured before the Taskfile
is even read, so completing inside another directory works. Completing
`--taskfile` filters on `.yml` and `.yaml`.

Flags gated behind an [experiment][experiments] are another good example. They
are read from Task's own flag set at the moment you press TAB, so they show up
exactly when the experiment is enabled. Before that, the shells either forked a
`task --experiments | grep` on every keystroke, or, in Nushell, offered them
unconditionally because the command signature was static.

## Completions should never surprise you

Completion runs on a keystroke, so it is allowed to do very little:

- No network access, even when your Taskfile has remote `includes:`. An uncached
  remote Taskfile used to freeze the shell until the 10 second timeout expired.
- Nothing is read from stdin, so no prompt can ever block your terminal.
- No `sh:` variable is evaluated. Pressing TAB never runs a command from your
  Taskfile.
- A broken or missing Taskfile still leaves you with flag completion.

## Try it today

The engine is opt-in. Swap `--completion` for `--new-completion` in whatever you
have in your shell config:

::: code-group

```shell [bash]
# ~/.bashrc
eval "$(task --new-completion bash)"
```

```shell [zsh]
# ~/.zshrc
eval "$(task --new-completion zsh)"
```

```shell [fish]
# ~/.config/fish/config.fish
task --new-completion fish | source
```

```powershell [powershell]
# $PROFILE\Microsoft.PowerShell_profile.ps1
Invoke-Expression (&task --new-completion powershell | Out-String)
```

:::

`--completion` keeps serving the old scripts, so nothing changes for you until
you ask for it. Once we are confident the new engine behaves well everywhere, it
will become what `--completion` returns. The [installation docs][install] cover
Nushell and the Zsh `verbose` and `show-aliases` zstyles, which still work.

## Feedback

Please tell us how it behaves in your shell. That is exactly where the remaining
surprises are, and a report costs us much less than a bug found after we flip
the default. You can find us on our [Discord server][discord] or [open an
issue][gh-issue].

[release]: https://github.com/go-task/task/releases/tag/v3.4X.0
[cobra]: https://github.com/spf13/cobra
[requires]: /docs/reference/schema#requires
[prompt-post]: /blog/if-and-variable-prompt
[experiments]: /docs/experiments/
[install]: /docs/installation#trying-the-new-completion-engine-experimental
[discord]: https://discord.com/invite/6TY36E39UK
[gh-issue]: https://github.com/go-task/task/issues
