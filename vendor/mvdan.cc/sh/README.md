# sh

[![GoDoc](https://godoc.org/mvdan.cc/sh?status.svg)](https://godoc.org/mvdan.cc/sh)
[![Build Status](https://travis-ci.org/mvdan/sh.svg?branch=master)](https://travis-ci.org/mvdan/sh)
[![Coverage Status](https://coveralls.io/repos/github/mvdan/sh/badge.svg?branch=master)](https://coveralls.io/github/mvdan/sh)

A shell parser, formatter and interpreter. Supports [POSIX Shell],
[Bash] and [mksh]. Requires Go 1.8 or later.

**Please note that the import paths have been moved from
`github.com/mvdan/sh/...` to `mvdan.cc/sh/...` for 2.0.** This will help
future-proof the project by making it depend less on GitHub.

### shfmt

	go get -u mvdan.cc/sh/cmd/shfmt

`shfmt` formats shell programs. It can use tabs or any number of spaces
to indent. See [canonical.sh](syntax/canonical.sh) for a quick look at
its default style.

You can feed it standard input, any number of files or any number of
directories to recurse into. When recursing, it will operate on `.sh`
and `.bash` files and ignore files starting with a period. It will also
operate on files with no extension and a shell shebang.

	shfmt -l -w script.sh

Use `-i N` to indent with a number of spaces instead of tabs. There are
other formatting options - see `shfmt -h`.

Packages are available for [Arch], [Homebrew], [NixOS] and [Void].

#### Advantages over `bash -n`

`bash -n` can be useful to check for syntax errors in shell scripts.
However, `shfmt >/dev/null` can do a better job as it checks for invalid
UTF-8 and does all parsing statically, including checking POSIX Shell
validity:

```
 $ echo '${foo:1 2}' | bash -n
 $ echo '${foo:1 2}' | shfmt
1:9: not a valid arithmetic operator: 2
 $ echo 'foo=(1 2)' | bash --posix -n
 $ echo 'foo=(1 2)' | shfmt -p
1:5: arrays are a bash feature
```

### gosh

	go get -u mvdan.cc/sh/cmd/gosh

Experimental non-interactive shell that uses `interp`. Work in progress,
so don't expect stability just yet.

### Fuzzing

This project makes use of [go-fuzz] to find crashes and hangs in both
the parser and the printer. To get started, run:

	git checkout fuzz
	./fuzz

### Caveats

* Bash index expressions must be an arithmetic expression or a quoted
  string. This is because the static parser can't know whether the array
  is an associative array (string keys) since that depends on having
  called or not `declare -A`.

```
 $ echo '${array[spaced string]}' | shfmt
1:16: not a valid arithmetic operator: string
```

* `$((` and `((` ambiguity is not suported. Backtracking would greatly
  complicate the parser and make stream support - `io.Reader` -
  impossible. In practice, the POSIX spec recommends to [space the
  operands][posix-ambiguity] if `$( (` is meant.

```
 $ echo '$((foo); (bar))' | shfmt
1:1: reached ) without matching $(( with ))
```

* Some builtins like `export` and `let` are parsed as keywords. This is
  to let the static parser parse them completely and build their AST
  better than just a slice of arguments.

### Related projects

* [format-shell] - Atom plugin for `shfmt`
* [shell-format] - VS Code plugin for `shfmt`
* [dockerised-shfmt] - A docker image of `shfmt`
* [vim-shfmt] - Vim plugin for `shfmt`

[posix shell]: http://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html
[bash]: https://www.gnu.org/software/bash/
[mksh]: https://www.mirbsd.org/mksh.htm
[examples]: https://godoc.org/mvdan.cc/sh/syntax#pkg-examples
[arch]: https://aur.archlinux.org/packages/shfmt/
[homebrew]: https://github.com/Homebrew/homebrew-core/blob/HEAD/Formula/shfmt.rb
[nixos]: https://github.com/NixOS/nixpkgs/blob/HEAD/pkgs/tools/text/shfmt/default.nix
[void]: https://github.com/voidlinux/void-packages/blob/HEAD/srcpkgs/shfmt/template
[go-fuzz]: https://github.com/dvyukov/go-fuzz
[posix-ambiguity]: http://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html#tag_18_06_03
[format-shell]: https://atom.io/packages/format-shell
[shell-format]: https://marketplace.visualstudio.com/items?itemName=foxundermoon.shell-format
[dockerised-shfmt]: https://hub.docker.com/r/jamesmstone/shfmt/
[vim-shfmt]: https://github.com/z0mbix/vim-shfmt
