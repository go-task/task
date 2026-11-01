---
title: Secret variables
description:
  Mask sensitive variable values in Task's command logs, load secrets from
  external sources, and understand what masking does not protect.
section: Guide
docType: guide
outline: deep
---

# Secret variables

Set `secret: true` on a variable to replace its value with `*****` in Task's own
command logs. The command still receives the original value.

```yaml
version: '3'

vars:
  API_TOKEN:
    sh: echo "$MY_API_TOKEN"
    secret: true

tasks:
  request:
    cmds:
      - 'curl -H "Authorization: Bearer {{.API_TOKEN}}" https://api.example.com'
      # Logged as: curl -H "Authorization: Bearer *****" https://api.example.com
```

Set `MY_API_TOKEN` in the environment before running `task request`. Task's log
masks the token, while `curl` receives the real value. The example URL is a
placeholder; replace it with your API endpoint.

Secret variables can be declared globally or on a task. Use `value` instead of
`sh` for a literal or template value, and keep real credentials out of committed
Taskfiles.

## Load from a secret store {#loading-secrets}

Use a dynamic variable to read a secret from an external source:

```yaml
version: '3'

vars:
  API_TOKEN:
    sh: vault kv get -field=api_key secret/myapp
    secret: true

tasks:
  request:
    cmds:
      - 'curl -H "Authorization: Bearer {{.API_TOKEN}}" https://api.example.com'
```

To read an existing environment variable, replace the `sh` command with
`echo "$MY_API_TOKEN"`. Keep local credential files, such as `.env.local`, out
of version control.

For dynamic variable evaluation and caching, see
[When values are computed](./variables.md#when-values-are-computed).

## Mark derived secrets {#derived-values}

The `secret` flag does not propagate to variables that reference a secret. Mark
every variable carrying the sensitive value as secret:

```yaml
version: '3'

vars:
  API_TOKEN:
    sh: echo "$MY_API_TOKEN"
    secret: true
  AUTH_HEADER:
    value: 'Bearer {{.API_TOKEN}}'
    secret: true

tasks:
  request:
    cmds:
      - 'curl -H "Authorization: {{.AUTH_HEADER}}" https://api.example.com'
      # Logged as: curl -H "Authorization: *****" https://api.example.com
```

Without `secret: true` on `AUTH_HEADER`, Task's command log would expose the
derived value.

## Understand masking limits {#limits-of-masking}

Masking applies to Task's own command logs, including when those logs are
collected by a CI system. It does not redact:

- stdout or stderr produced by commands;
- process arguments visible to tools such as `ps`;
- values stored in shell history or committed files;
- derived variables that are not themselves marked as secret.

For example, a command that echoes a secret still prints its real value, even
though Task masks the command text it logs before execution.

The `secret` flag applies to `vars`, not `env`. Use your command's supported
credential mechanism to control how it receives secrets.
