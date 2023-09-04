---
slug: /integrations/
sidebar_position: 8
---

# 集成

## Visual Studio Code 扩展

Task 有一个 [针对 Visual Studio Code 的官方扩展](https://marketplace.visualstudio.com/items?itemName=task.vscode-task)。 这个项目的代码可以在 [这里](https://github.com/go-task/vscode-task) 找到。 要使用此扩展，您的系统上必须安装 Task v3.23.0+。

此扩展提供以下功能（以及更多）：

- 在侧边栏中查看 task。
- 从侧边栏和命令面板运行 task。
- 从侧边栏和命令面板转到定义。
- 运行上一个 task 命令。
- 多根工作区支持。
- 在当前工作空间中初始化一个 Taskfile。

要获得 Taskfile 的自动完成和验证，请参阅下面的 [Schema](#schema) 部分。

![Task for Visual Studio Code](https://github.com/go-task/vscode-task/blob/main/res/preview.png?raw=true)

## Schema

这最初是由 [@KROSF](https://github.com/KROSF) 在 [这个 Gist](https://gist.github.com/KROSF/c5435acf590acd632f71bb720f685895) 中创建的，现在在 [这个](https://github.com/go-task/task/blob/main/docs/static/schema.json) 文件中正式维护，并在 https://taskfile.dev/schema.json 上提供。 这个 schema 可用于验证 Taskfile 并在许多代码编辑器中提供自动完成功能：

### Visual Studio Code

要将 schema 集成到 VS Code 中，您需要安装 Red Hat 的 [YAML 扩展](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml)。 项目中的任何 `Taskfile.yml` 都应该被自动检测到，并且验证/自动完成应该可以工作。 如果这不起作用或者您想为具有不同名称的文件手动配置它，您可以将以下内容添加到您的 `settings.json`：

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

您还可以通过将以下注释添加到文件顶部来直接在 Taskfile 中配置 schema：

```yaml
# yaml-language-server: $schema=https://taskfile.dev/schema.json
version: '3'
```

您可以在 [YAML 语言服务器项目](https://github.com/redhat-developer/yaml-language-server) 中找到更多相关信息。

## 社区集成

除了我们的官方集成之外，还有一个很棒的开发人员社区，他们为 Task 创建了自己的集成：

- [Sublime Text Plugin](https://packagecontrol.io/packages/Taskfile) [[源码](https://github.com/biozz/sublime-taskfile)] 由 [@biozz](https://github.com/biozz)
- [IntelliJ Plugin](https://plugins.jetbrains.com/plugin/17058-taskfile) [[源码](https://github.com/lechuckroh/task-intellij-plugin)] 由 [@lechuckroh](https://github.com/lechuckroh)
- [mk](https://github.com/pycontribs/mk) 命令行工具本机识别 Taskfile。

如果你做了一些与 Task 集成的东西，请随意打开一个 PR 将它添加到这个列表中。
