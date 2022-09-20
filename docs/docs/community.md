---
slug: /community/
sidebar_position: 6
---

# Community

Some of the work to improve the Task ecosystem is done by the community, be
it installation methods or integrations with code editor. I (the author) am
thankful for everyone that helps me to improve the overall experience.

## Editor Integrations

### JSON Schema

[@KROSF](https://github.com/KROSF) worked on a JSON Schema [into this Gist](https://gist.github.com/KROSF/c5435acf590acd632f71bb720f685895),
which later was made officially available by [@Crandel](https://github.com/Crandel)
at [https://json.schemastore.org/taskfile.json](https://json.schemastore.org/taskfile.json).
Further improvements are possible by opening pull requests changing
[this file](https://github.com/SchemaStore/schemastore/blob/master/src/schemas/json/taskfile.json).
Some code editors, like Visual Studio Code, make use of Schema Store
automatically.

### Visual Studio Code extension

Additionally, there's also some work done by
[@paulvarache](https://github.com/paulvarache) in making an Visual Studio Code
extension, which has its code [here](https://github.com/paulvarache/vscode-taskfile)
and is published [here](https://marketplace.visualstudio.com/items?itemName=paulvarache.vscode-taskfile).

### Sublime Text 4 package

There is a convenience wrapper for initializing and running tasks from Sublime Text's command palette. The package is
developed by [@biozz](https://github.com/biozz), the source code is available [here](https://github.com/biozz/sublime-taskfile)
and it is published on Package Control [here](https://packagecontrol.io/packages/Taskfile).

### IntelliJ plugin

There's a JetBrains IntelliJ plugin done by
[@lechuckroh](https://github.com/lechuckroh), which has its code [here](https://github.com/lechuckroh/task-intellij-plugin)
and is published [here](https://plugins.jetbrains.com/plugin/17058-taskfile).

## Installation methods

Some installation methods are maintained by third party:

- [GitHub Actions](https://github.com/arduino/setup-task)
  by [@arduino](https://github.com/arduino)
- [AUR](https://aur.archlinux.org/packages/go-task-bin)
  by [@carlsmedstad](https://github.com/carlsmedstad)
- [Scoop](https://github.com/lukesampson/scoop-extras/blob/master/bucket/task.json)
- [Fedora](https://packages.fedoraproject.org/pkgs/golang-github-task/go-task/)
- [NixOS](https://github.com/NixOS/nixpkgs/blob/master/pkgs/development/tools/go-task/default.nix)

## More

Also, thanks for all the [code contributors](https://github.com/go-task/task/graphs/contributors),
[financial contributors](https://opencollective.com/task), all those who
[reported bugs](https://github.com/go-task/task/issues?q=is%3Aissue) and
[answered questions](https://github.com/go-task/task/discussions).

If you know something that is missing in this document, please submit a
pull request.
