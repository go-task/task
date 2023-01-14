---
slug: /community/
sidebar_position: 8
---

# Community

Some of the work to improve the Task ecosystem is done by the community, be
it installation methods or integrations with code editor. I (the author) am
thankful for everyone that helps me to improve the overall experience.

## Translations

[@DeronW](https://github.com/DeronW) maintains  the
[Chinese translation](https://task-zh.readthedocs.io/zh_CN/latest/) of the
website [on this repository](https://github.com/DeronW/task).

## Editor Integrations

### JSON Schema

Initial work on the schema was made by [@KROSF](https://github.com/KROSF)
on [this Gist](https://gist.github.com/KROSF/c5435acf590acd632f71bb720f685895).
The schema is currently available at
https://taskfile.dev/schema.json and linked at https://json.schemastore.org/taskfile.json
so it is be used automatically many code editors, like VSCode.
Contributions can be done by editing [this file](https://github.com/go-task/task/blob/master/docs/static/schema.json).

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
