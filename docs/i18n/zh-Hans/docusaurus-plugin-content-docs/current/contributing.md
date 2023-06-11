---
slug: /contributing/
sidebar_position: 11
---

# 贡献

非常欢迎对 Task 的贡献，但我们要求您在提交 PR 之前阅读本文档。

:::note

本文档适用于 [Task][task] 核心存储库 _和_ [Task for Visual Studio Code][vscode-task]。

:::

## 开始之前

- **检查已有工作** - 是否已经存在 PR？ 是否存在 Issue 正在讨论您要进行的功能/更改？ 请确保你的工作中确实考虑了这些相关的讨论内容。
- **向后兼容** - 你的变更是否破坏了已经存在的 Taskfile？ 向后兼容的变更会更容易被合并进去。 您是否可以采取一种方法来保持这种兼容性？ 如果没有，请考虑先提出一个 Issue，以便在您投入时间进行 PR 之前讨论 API 的更改。
- **Experiments** - If there is no way to make your change backward compatible then there is a procedure to introduce breaking changes into minor versions. We call these "\[experiments\]\[experiments\]". If you're intending to work on an experiment, then please read the \[experiments workflow\]\[experiments-workflow\] document carefully and submit a proposal first.

## 1. 设置

- **Go** - Task 使用 [Go][go] 编写。 我们始终支持最新的两个主要 Go 版本，因此请确保您的版本足够新。
- **Node.js** - [Node.js][nodejs] 用于托管 Task 的文档服务器，如果您想在本地运行此服务器，则需要它。 如果您想为 Visual Studio Code 扩展做贡献，也需要它。
- **Yarn** - [Yarn][yarn] 是 Task 使用的 Node.js 包管理器。

## 2. 进行变更

- **代码风格** - 尽量保持现有的代码风格。 Go 代码应该由 [`gofumpt`][gofumpt] 格式化并使用 [`golangci-lint`][golangci-lint] 进行检查。 任何 Markdown 或 TypeScript 文件都应该由 [Prettier][prettier] 格式化和检查。 这种风格由我们的 CI 强制执行，以确保我们在整个项目中拥有一致的风格。 您可以使用 `task lint` 命令在本地检查代码，并使用 `task lint:fix` 命令自动修复发现的任何问题。
- **文档** - 确保添加/更新了相关文档。 请参阅下面的 [更新文档](#更新文档) 部分。
- **测试** - 确保添加/更新了相关测试，并且在提交 PR 前已通过所有测试。 请参阅下面的 [编写测试](#编写测试) 部分。

### 运行您的变更

要运行带有工作变更的任务，您可以使用 `go run ./cmd/task`。 要针对 `testdata` 中的测试任务文件运行任务的开发构建，您可以使用 `go
run ./cmd/task --dir ./testdata/<my_test_dir> <task_name>`。

要运行 Task for Visual Studio Code，您可以在 VSCode 中打开项目并按 F5（或任何您设置绑定的调试键）。 这将打开一个新的 VSCode 窗口，扩展正在运行。 建议以这种方式进行调试，因为它允许您设置断点并单步执行代码。 或者，您可以运行 `task package`，这将生成一个可用于手动安装扩展的 `.vsix` 文件。

### 更新文档

Task 使用 [Docusaurus][docusaurus] 来托管文档服务器。 此代码位于 Task 核心存储库中。 这可以通过使用 `task docs`（需要 `nodejs` 和 `yarn`）在本地设置和运行。 所有内容均使用 Markdown 编写，位于 `docs/docs` 目录中。 所有 Markdown 文档都应有 80 个字符的换行限制（由 Prettier 强制执行）。

进行变更时，请考虑是否有必要更改 [使用指南](./usage.md)。 本文档包含有关如何使用任务功能的说明和示例。 如果您要添加新功能，请尝试找到合适的位置来添加新部分。 如果您要更新现有功能，请确保文档和所有示例都是最新的。 确保任何示例都遵循 [Taskfile 风格指南](./styleguide.md)。

如果您添加了新字段、命令或标志，请确保将其添加到 [API 参考](./api_reference.md) 中。 还需要将新字段添加到 [JSON Schema][json-schema] 中。 API 参考和 schema 中的字段描述应该匹配。

### 编写测试

许多 Task 的测试都保存在项目根目录下的 `task_test.go` 文件中，这也是您最有可能想要添加新测试的地方。 大多数这些测试在 `testdata` 目录中也有一个子目录，其中存储了运行测试所需的任何 Taskfiles/数据。

进行更改时，请考虑是否需要添加新的测试。 这些测试应确保您添加的功能在未来持续工作。 如果您更改了 Task 的行为，则现有测试也可能需要更新。

您还可以考虑为您添加的任何新功能添加单元测试。 单元测试应遵循 Go 约定，即位于与被测试代码相同的包中名为 `*_test.go` 的文件中。

## 3. 提交代码

尝试编写有意义的提交消息并避免在 PR 上有太多提交。 大多数 PR 应该有一个单一的提交（尽管对于更大的 PR 将它分成几个可能是合理的）。 Git squash(并和) 和 rebase(变基) 是你的好伙伴!

如果您不确定如何格式化提交消息，请查看 [约定式提交][conventional-commits]。 这种风格不是强制的，但它是使您的提交消息更具可读性和一致性的好方法。

## 4. 提交 PR

- **描述变更** - 确保您提供对更改的全面描述。
- **Issue/PR 链接** - 链接到之前相关的 Issue 或 PR。 请描述当前工作与之前的不同之处。
- **示例** - 添加您认为有助于展示更改效果的任何示例或屏幕截图。
- **PR 草案** - 如果变更还未完成，但您想讨论它们，请将 PR 作为草稿打开并添加评论以开始讨论。 使用评论而不是 PR 描述允许稍后更新描述，同时保留讨论。

## 常见问题

> 我想贡献，我从哪里开始？

查看 [Task][task-open-issues] 或 [Task for Visual Studio Code][vscode-task-open-issues] 的未解决问题列表。 我们有一个 [good first issue][good-first-issue] 标签，用于更简单的问题，非常适合首次贡献。

欢迎各种贡献，无论是拼写错误修复还是很小的新功能。 您还可以通过对 Issue 进行投票/评论、帮助回答问题或帮助 [其他社区项目](./community.md) 来做出贡献。

> 我被困住了，我在哪里可以获得帮助？

如果您有任何疑问，请随时在我们的 [Discord 服务器][discord-server] 上的 `#help` 论坛频道中提问，或在 GitHub 上打开 [讨论][discussion]。

---

<!-- prettier-ignore-start -->

<!-- prettier-ignore-end -->
[task]: https://github.com/go-task/task
[vscode-task]: https://github.com/go-task/vscode-task
[go]: https://go.dev
[gofumpt]: https://github.com/mvdan/gofumpt
[golangci-lint]: https://golangci-lint.run
[prettier]: https://prettier.io
[nodejs]: https://nodejs.org/en/
[yarn]: https://yarnpkg.com/
[docusaurus]: https://docusaurus.io
[json-schema]: https://github.com/go-task/task/blob/main/docs/static/schema.json
[task-open-issues]: https://github.com/go-task/task/issues
[vscode-task-open-issues]: https://github.com/go-task/vscode-task/issues
[good-first-issue]: https://github.com/go-task/task/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22
[discord-server]: https://discord.gg/6TY36E39UK
[discussion]: https://github.com/go-task/task/discussions
[conventional-commits]: https://www.conventionalcommits.org
