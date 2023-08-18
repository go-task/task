---
slug: /contributing/
sidebar_position: 11
---

# コントリビュート

Contributions to Task are very welcome, but we ask that you read this document before submitting a PR.

:::note

This document applies to the core [Task][task] repository _and_ [Task for Visual Studio Code][vscode-task].

:::

## 始める前に

- **Check existing work** - Is there an existing PR? Are there issues discussing the feature/change you want to make? Please make sure you consider/address these discussions in your work.
- **Backwards compatibility** - Will your change break existing Taskfiles? It is much more likely that your change will merged if it backwards compatible. Is there an approach you can take that maintains this compatibility? If not, consider opening an issue first so that API changes can be discussed before you invest your time into a PR.
- **Experiments** - If there is no way to make your change backward compatible then there is a procedure to introduce breaking changes into minor versions. We call these "\[experiments\]\[experiments\]". If you're intending to work on an experiment, then please read the \[experiments workflow\]\[experiments-workflow\] document carefully and submit a proposal first.

## 1. セットアップ

- **Go** - Taskは[Go][go]で書かれています。 We always support the latest two major Go versions, so make sure your version is recent enough.
- **Node.js** - [Node.js][nodejs] is used to host Task's documentation server and is required if you want to run this server locally. It is also required if you want to contribute to the Visual Studio Code extension.
- **Yarn** - [Yarn][yarn] is the Node.js package manager used by Task.

## 2. 変更を加える

- **Code style** - Try to maintain the existing code style where possible. Go code should be formatted by [`gofumpt`][gofumpt] and linted using [`golangci-lint`][golangci-lint]. Any Markdown or TypeScript files should be formatted and linted by [Prettier][prettier]. This style is enforced by our CI to ensure that we have a consistent style across the project. You can use the `task lint` command to lint the code locally and the `task lint:fix` command to automatically fix any issues that are found.
- **Documentation** - Ensure that you add/update any relevant documentation. See the [updating documentation](#updating-documentation) section below.
- **Tests** - Ensure that you add/update any relevant tests and that all tests are passing before submitting the PR. See the [writing tests](#writing-tests) section below.

### Running your changes

To run Task with working changes, you can use `go run ./cmd/task`. To run a development build of task against a test Taskfile in `testdata`, you can use `go run ./cmd/task --dir ./testdata/<my_test_dir> <task_name>`.

To run Task for Visual Studio Code, you can open the project in VSCode and hit F5 (or whatever you debug keybind is set to). This will open a new VSCode window with the extension running. Debugging this way is recommended as it will allow you to set breakpoints and step through the code. Otherwise, you can run `task package` which will generate a `.vsix` file that can be used to manually install the extension.

### ドキュメントの更新

Task uses [Docusaurus][docusaurus] to host a documentation server. The code for this is located in the core Task repository. This can be setup and run locally by using `task docs` (requires `nodejs` & `yarn`). All content is written in Markdown and is located in the `docs/docs` directory. All Markdown documents should have an 80 character line wrap limit (enforced by Prettier).

When making a change, consider whether a change to the [Usage Guide](./usage.md) is necessary. This document contains descriptions and examples of how to use Task features. If you're adding a new feature, try to find an appropriate place to add a new section. If you're updating an existing feature, ensure that the documentation and any examples are up-to-date. Ensure that any examples follow the [Taskfile Styleguide](./styleguide.md).

If you added a new field, command or flag, ensure that you add it to the [API Reference](./api_reference.md). New fields also need to be added to the [JSON Schema][json-schema]. The descriptions for fields in the API reference and the schema should match.

### Writing tests

A lot of Task's tests are held in the `task_test.go` file in the project root and this is where you'll most likely want to add new ones too. Most of these tests also have a subdirectory in the `testdata` directory where any Taskfiles/data required to run the tests are stored.

When making a changes, consider whether new tests are required. These tests should ensure that the functionality you are adding will continue to work in the future. Existing tests may also need updating if you have changed Task's behavior.

You may also consider adding unit tests for any new functions you have added. The unit tests should follow the Go convention of being location in a file named `*_test.go` in the same package as the code being tested.

## 3. Committing your code

Try to write meaningful commit messages and avoid having too many commits on the PR. Most PRs should likely have a single commit (although for bigger PRs it may be reasonable to split it in a few). Git squash and rebase is your friend!

If you're not sure how to format your commit message, check out [Conventional Commits][conventional-commits]. This style is not enforced, but it is a good way to make your commit messages more readable and consistent.

## 4. Submitting a PR

- **Describe your changes** - Ensure that you provide a comprehensive description of your changes.
- **Issue/PR links** - Link any previous work such as related issues or PRs. Please describe how your changes differ to/extend this work.
- **Examples** - Add any examples or screenshots that you think are useful to demonstrate the effect of your changes.
- **Draft PRs** - If your changes are incomplete, but you would like to discuss them, open the PR as a draft and add a comment to start a discussion. Using comments rather than the PR description allows the description to be updated later while preserving any discussions.

## FAQ

> コントリビュートしたいのですが、どうすればいいですか？

Take a look at the list of [open issues for Task][task-open-issues] or [Task for Visual Studio Code][vscode-task-open-issues]. We have a [good first issue][good-first-issue] label for simpler issues that are ideal for first time contributions.

All kinds of contributions are welcome, whether its a typo fix or a shiny new feature. You can also contribute by upvoting/commenting on issues, helping to answer questions or contributing to other [community projects](./community.md).

> I'm stuck, where can I get help?

If you have questions, feel free to ask them in the `#help` forum channel on our [Discord server][discord-server] or open a [Discussion][discussion] on GitHub.

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
