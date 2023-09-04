---
slug: /integrations/
sidebar_position: 8
---

# Integrations

## Visual Studio Code Extension

Task has an [official extension for Visual Studio Code](https://marketplace.visualstudio.com/items?itemName=task.vscode-task). The code for this project can be found [here](https://github.com/go-task/vscode-task). To use this extension, you must have Task v3.23.0+ installed on your system.

This extension provides the following features (and more):

- View tasks in the sidebar.
- Run tasks from the sidebar and command palette.
- Go to definition from the sidebar and command palette.
- Run last task command.
- Multi-root workspace support.
- Initialize a Taskfile in the current workspace.

To get autocompletion and validation for your Taskfile, see the [Schema](#schema) section below.

![Task for Visual Studio Code](https://github.com/go-task/vscode-task/blob/main/res/preview.png?raw=true)

## Schema

This was initially created by [@KROSF](https://github.com/KROSF) in [this Gist](https://gist.github.com/KROSF/c5435acf590acd632f71bb720f685895) and is now officially maintained in [this file](https://github.com/go-task/task/blob/main/docs/static/schema.json) and made available at https://taskfile.dev/schema.json. This schema can be used to validate Taskfiles and provide autocompletion in many code editors:

### Visual Studio Code

To integrate the schema into VS Code, you need to install the [YAML extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml) by Red Hat. Any `Taskfile.yml` in your project should automatically be detected and validation/autocompletion should work. If this doesn't work or you want to manually configure it for files with a different name, you can add the following to your `settings.json`:

```json
// settings.json
{
  "yaml.schemas": {
    "https://taskfile.dev/schema.json": [
      "**/Taskfile.yml",
      "./path/to/any/other/taskfile.yml"
    ]
  }
}
```

You can also configure the schema directly inside of a Taskfile by adding the following comment to the top of the file:

```yaml
# yaml-language-server: $schema=https://taskfile.dev/schema.json
version: '3'
```

You can find more information on this in the [YAML language server project](https://github.com/redhat-developer/yaml-language-server).

## Community Integrations

In addition to our official integrations, there is an amazing community of developers who have created their own integrations for Task:

- [Sublime Text Plugin](https://packagecontrol.io/packages/Taskfile) [[source](https://github.com/biozz/sublime-taskfile)] by [@biozz](https://github.com/biozz)
- [IntelliJ Plugin](https://plugins.jetbrains.com/plugin/17058-taskfile) [[source](https://github.com/lechuckroh/task-intellij-plugin)] by [@lechuckroh](https://github.com/lechuckroh)
- [mk](https://github.com/pycontribs/mk) command line tool recognizes Taskfiles natively.

If you have made something that integrates with Task, please feel free to open a PR to add it to this list.
