---
name: Pull Request
about: Open a pull request.
body:
  - type: markdown
    attributes:
      value: |
        Thanks for your pull request, we really appreciate contributions!

        Please understand that it may take some time to be reviewed.

        Also, make sure to follow the [Contribution Guide](https://taskfile.dev/contributing/).
  - type: textarea
    id: description
    attributes:
      label: Description
      description: |
        Describe the PR you're opening.
      placeholder: You description here.
    validations:
      required: true
---
