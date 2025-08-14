---
title: Schema taskrc Reference
description: Complete reference for the taskrc schema
outline: deep
---

# Schema taskrc Reference

The `.taskrc` file is the configuration file used by Task. It can have either a
`.yml` or `.yaml` extension and can be located in two places:

- **Local** – at the root of the project.
- **Global** – in the user's HOME directory.

When both local and global `.taskrc.yml` files are present, their contents are
merged. If the same key exists in both files, the value from the local file
takes precedence over the global one.

## Schema

| Attribute     | Type                                | Default | Description                                          |
| ------------- | ----------------------------------- | ------- | ---------------------------------------------------- |
| `version`     | `string`                            |         | Version of the Taskfile. The current version is `3`. |
| `experiments` | [`map[string]number`](#experiments) |         | Experiments to enable or disable                     |

## Experiments

| Attribute          | Type     | Default | Description                                              |
| ------------------ | -------- | ------- | -------------------------------------------------------- |
| `REMOTE_TASKFILES` | `number` |         | Enable (1) or disable (0) the remote taskfile experiment |
| `ENV_PRECEDENCE`   | `number` |         | Enable (1) or disable (0) the env precedence experiment. |
| `GENTLE_FORCE`     | `number` |         | Enable (1) or disable (0) the gentle force experiment.   |
