---
slug: /contributing/
sidebar_position: 9
---

# Contributing

Contributions to Task are very welcome, but we ask that you read this document
before submitting a PR.

## Before you start

- **Check existing work** - Is there an existing PR? Are there issues discussing
the feature/change you want to make? Please make sure you consider/address these
discussions in your work.
- **Backwards compatibility** - Will your change break existing Taskfiles? It is
much more likely that your change will merged if it backwards compatible. Is
there an approach you can take that maintains this compatibility? If not,
consider opening an issue first so that API changes can be discussed before you
invest you time into a PR.

## 1. Setup

- **Go** - Task is written in [Go]. We always support the latest two major Go
  versions, so make sure your version is recent enough.
- **Node.js** - [Node.js] is used to host Task's documentation server and is
  required if you want to run this server locally.
- **Yarn** - [Yarn] is the Node.js package manager used by Task.

## 2. Making changes

- **Code style** - Try to maintain the existing code style where possible and
  ensure that code is formatted by `gofmt`. We use `golangci-lint` in our CI to
  enforce a consistent style and best-practise. There's a `lint` command in
  the Taskfile to run this locally.
- **Documentation** - Ensure that you add/update any relevant documentation. See
  the [updating documentation](#updating-documentation) section below.
- **Tests** - Ensure that you add/update any relevant tests and that all tests
  are passing before submitting the PR. See the [writing tests](#writing-tests)
  section below.

### Running your changes

To run Task with working changes, you can use `go run ./cmd/task`. To run a
development build of task against a test Taskfile in `testdata`, you can use `go
run ./cmd/task --dir ./testdata/<my_test_dir> <task_name>`.

### Updating documentation

Task uses [Docusaurus] to host a documentation server. This can be setup and run
locally by using `task docs:setup` and `task docs:start` respectively (requires
`nodejs` & `yarn`). All content is written in Markdown and is located in the
`docs/docs` directory. All Markdown documents should have an 80 character line
wrap limit.

When making a change, consider whether a change to the [Usage Guide](./usage.md)
is necessary. This document contains descriptions and examples of how to use
Task features. If you're adding a new feature, try to find an appropriate place
to add a new section. If you're updating an existing feature, ensure that the
documentation and any examples are up-to-date. Ensure that any examples follow
the [Taskfile Styleguide](./styleguide.md).

If you added a new field, command or flag, ensure that you add it to the [API
Reference](./api_reference.md). New fields also need to be added to the
[JSON Schema](../static/schema.json). The descriptions for fields in the API
reference and the schema should match.

### Writing tests

Most of Task's test are held in the `task_test.go` file in the project root and
this is where you'll most likely want to add new ones too. Most of these tests
also have a subdirectory in the `testdata` directory where any Taskfiles/data
required to run the tests are stored.

When making a changes, consider whether new tests are required. These tests
should ensure that the functionality you are adding will continue to work in the
future. Existing tests may also need updating if you have changed Task's
behaviour.

## 3. Committing your code

Try to write meaningful commit messages and avoid having too many commits on
the PR. Most PRs should likely have a single commit (although for bigger PRs it
may be reasonable to split it in a few). Git squash and rebase is your friend!

## 4. Submitting a PR

- **Describe your changes** - Ensure that you provide a comprehensive
  description of your changes.
- **Issue/PR links** - Link any previous work such as related issues or PRs.
  Please describe how your changes differ to/extend this work.
- **Examples** - Add any examples that you think are useful to demonstrate the
  effect of your changes.
- **Draft PRs** - If your changes are incomplete, but you would like to discuss
  them, open the PR as a draft and add a comment to start a discussion. Using
  comments rather than the PR description allows the description to be updated
  later while preserving any discussions.

## FAQ

> I want to contribute, where do I start?

Take a look at the list of [open issues]. We have a [good first issue] label for
simpler issues that are ideal for first time contributions.

All kinds of contributions are welcome, whether its a typo fix or a shiny new
feature. You can also contribute by upvoting/commenting on issues, helping to
answer questions or contributing to other [community projects](./community.md).

> I'm stuck, where can I get help?

If you have questions, feel free to ask them in the `#help` channel on our
[Discord server].

---

[Go]: https://go.dev
[install version 1.18+]: https://go.dev/doc/install
[Node.js]: https://nodejs.org/en/
[Yarn]: https://yarnpkg.com/
[Docusaurus]: https://docusaurus.io
[open issues]: https://github.com/go-task/task/issues
[good first issue]: https://github.com/go-task/task/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22
[Discord server]: https://discord.gg/6TY36E39UK
