---
title: Taskfile Schema Reference
description: A reference for the Taskfile schema
outline: deep
---

# Taskfile Schema Reference

This page documents all available properties and types for the Taskfile schema
version 3, based on the
[official JSON schema](https://taskfile.dev/schema.json).

## Root Schema

The root Taskfile schema defines the structure of your main `Taskfile.yml`.

### `version`

- **Type**: `string` or `number`
- **Required**: Yes
- **Valid values**: `"3"`, `3`, or any valid semver string
- **Description**: Version of the Taskfile schema

```yaml
version: '3'
```

### `output`

- **Type**: `string` or `object`
- **Default**: `interleaved`
- **Options**: `interleaved`, `group`, `prefixed`
- **Description**: Controls how task output is displayed

```yaml
# Simple string format
output: group

# Advanced object format
output:
  group:
    begin: "::group::{{.TASK}}"
    end: "::endgroup::"
    error_only: false
```

### `method`

- **Type**: `string`
- **Default**: `checksum`
- **Options**: `checksum`, `timestamp`, `none`
- **Description**: Default method for checking if tasks are up-to-date

```yaml
method: timestamp
```

### [`includes`](#include)

- **Type**: `map[string]Include`
- **Description**: Include other Taskfiles

```yaml
includes:
  # Simple string format
  docs: ./Taskfile.yml

  # Full object format
  backend:
    taskfile: ./backend
    dir: ./backend
    optional: false
    flatten: false
    internal: false
    aliases: [api]
    excludes: [internal-task]
    vars:
      SERVICE_NAME: backend
    checksum: abc123...
```

### [`vars`](#variable)

- **Type**: `map[string]Variable`
- **Description**: Global variables available to all tasks

```yaml
vars:
  # Simple values
  APP_NAME: myapp
  VERSION: 1.0.0
  DEBUG: true
  PORT: 8080
  FEATURES: [auth, logging]

  # Dynamic variables
  COMMIT_HASH:
    sh: git rev-parse HEAD

  # Variable references
  BUILD_VERSION:
    ref: .VERSION

  # Map variables
  CONFIG:
    map:
      database: postgres
      cache: redis
```

### `env`

- **Type**: `map[string]Variable`
- **Description**: Global environment variables

```yaml
env:
  NODE_ENV: production
  DATABASE_URL:
    sh: echo $DATABASE_URL
```

### [`tasks`](#task)

- **Type**: `map[string]Task`
- **Description**: Task definitions

```yaml
tasks:
  # Simple string format
  hello: echo "Hello World"

  # Array format
  build:
    - go mod tidy
    - go build ./...

  # Full object format
  deploy:
    desc: Deploy the application
    cmds:
      - ./scripts/deploy.sh
```

### `silent`

- **Type**: `bool`
- **Default**: `false`
- **Description**: Suppress task name and command output by default

```yaml
silent: true
```

### `dotenv`

- **Type**: `[]string`
- **Description**: Load environment variables from .env files

```yaml
dotenv:
  - .env
  - .env.local
```

### `run`

- **Type**: `string`
- **Default**: `always`
- **Options**: `always`, `once`, `when_changed`
- **Description**: Default execution behavior for tasks

```yaml
run: once
```

### `interval`

- **Type**: `string`
- **Default**: `100ms`
- **Pattern**: `^[0-9]+(?:m|s|ms)$`
- **Description**: Watch interval for file changes

```yaml
interval: 1s
```

### `set`

- **Type**: `[]string`
- **Options**: `allexport`, `a`, `errexit`, `e`, `noexec`, `n`, `noglob`, `f`,
  `nounset`, `u`, `xtrace`, `x`, `pipefail`
- **Description**: POSIX shell options for all commands

```yaml
set: [errexit, nounset, pipefail]
```

### `shopt`

- **Type**: `[]string`
- **Options**: `expand_aliases`, `globstar`, `nullglob`
- **Description**: Bash shell options for all commands

```yaml
shopt: [globstar]
```

## Include

Configuration for including external Taskfiles.

### `taskfile`

- **Type**: `string`
- **Required**: Yes
- **Description**: Path to the Taskfile or directory to include

```yaml
includes:
  backend: ./backend/Taskfile.yml
  # Shorthand for above
  frontend: ./frontend
```

### `dir`

- **Type**: `string`
- **Description**: Working directory for included tasks

```yaml
includes:
  api:
    taskfile: ./api
    dir: ./api
```

### `optional`

- **Type**: `bool`
- **Default**: `false`
- **Description**: Don't error if the included file doesn't exist

```yaml
includes:
  optional-tasks:
    taskfile: ./optional.yml
    optional: true
```

### `flatten`

- **Type**: `bool`
- **Default**: `false`
- **Description**: Include tasks without namespace prefix

```yaml
includes:
  common:
    taskfile: ./common.yml
    flatten: true
```

### `internal`

- **Type**: `bool`
- **Default**: `false`
- **Description**: Hide included tasks from command line and `--list`

```yaml
includes:
  internal:
    taskfile: ./internal.yml
    internal: true
```

### `aliases`

- **Type**: `[]string`
- **Description**: Alternative names for the namespace

```yaml
includes:
  database:
    taskfile: ./db.yml
    aliases: [db, data]
```

### `excludes`

- **Type**: `[]string`
- **Description**: Tasks to exclude from inclusion

```yaml
includes:
  shared:
    taskfile: ./shared.yml
    excludes: [internal-setup, debug-only]
```

### `vars`

- **Type**: `map[string]Variable`
- **Description**: Variables to pass to the included Taskfile

```yaml
includes:
  deploy:
    taskfile: ./deploy.yml
    vars:
      ENVIRONMENT: production
```

### `checksum`

- **Type**: `string`
- **Description**: Expected checksum of the included file

```yaml
includes:
  remote:
    taskfile: https://example.com/tasks.yml
    checksum: c153e97e0b3a998a7ed2e61064c6ddaddd0de0c525feefd6bba8569827d8efe9
```

## Variable

Variables support multiple types and can be static values, dynamic commands,
references, or maps.

### Static Variables

```yaml
vars:
  # String
  APP_NAME: myapp
  # Number
  PORT: 8080
  # Boolean
  DEBUG: true
  # Array
  FEATURES: [auth, logging, metrics]
  # Null
  OPTIONAL_VAR: null
```

### Dynamic Variables (`sh`)

```yaml
vars:
  COMMIT_HASH:
    sh: git rev-parse HEAD
  BUILD_TIME:
    sh: date -u +"%Y-%m-%dT%H:%M:%SZ"
```

### Variable References (`ref`)

```yaml
vars:
  BASE_VERSION: 1.0.0
  FULL_VERSION:
    ref: .BASE_VERSION
```

### Map Variables (`map`)

```yaml
vars:
  CONFIG:
    map:
      database:
        host: localhost
        port: 5432
      cache:
        type: redis
        ttl: 3600
```

### Variable Ordering

Variables can reference previously defined variables:

```yaml
vars:
  GREETING: Hello
  TARGET: World
  MESSAGE: '{{.GREETING}} {{.TARGET}}!'
```

## Task

Individual task configuration with multiple syntax options.

### Simple Task Formats

```yaml
tasks:
  # String command
  hello: echo "Hello World"

  # Array of commands
  build:
    - go mod tidy
    - go build ./...

  # Object with cmd shorthand
  test:
    cmd: go test ./...
```

### Task Properties

#### `cmds`

- **Type**: `[]Command`
- **Description**: Commands to execute

```yaml
tasks:
  build:
    cmds:
      - go build ./...
      - echo "Build complete"
```

#### `cmd`

- **Type**: `string`
- **Description**: Single command (alternative to `cmds`)

```yaml
tasks:
  test:
    cmd: go test ./...
```

#### `deps`

- **Type**: `[]Dependency`
- **Description**: Tasks to run before this task

```yaml
tasks:
  # Simple dependencies
  deploy:
    deps: [build, test]
    cmds:
      - ./deploy.sh

  # Dependencies with variables
  advanced-deploy:
    deps:
      - task: build
        vars:
          ENVIRONMENT: production
      - task: test
        vars:
          COVERAGE: true
    cmds:
      - ./deploy.sh

  # Silent dependencies
  main:
    deps:
      - task: setup
        silent: true
    cmds:
      - echo "Main task"

  # Loop dependencies
  test-all:
    deps:
      - for: [unit, integration, e2e]
        task: test
        vars:
          TEST_TYPE: '{{.ITEM}}'
    cmds:
      - echo "All tests completed"
```

#### `desc`

- **Type**: `string`
- **Description**: Short description shown in `--list`

```yaml
tasks:
  test:
    desc: Run unit tests
    cmds:
      - go test ./...
```

#### `summary`

- **Type**: `string`
- **Description**: Detailed description shown in `--summary`

```yaml
tasks:
  deploy:
    desc: Deploy to production
    summary: |
      Deploy the application to production environment.
      This includes building, testing, and uploading artifacts.
```

#### `prompt`

- **Type**: `string` or `[]string`
- **Description**: Prompts shown before task execution

```yaml
tasks:
  # Single prompt
  deploy:
    prompt: "Deploy to production?"
    cmds:
      - ./deploy.sh

  # Multiple prompts
  deploy-multi:
    prompt:
      - "Are you sure?"
      - "This will affect live users!"
    cmds:
      - ./deploy.sh
```

#### `aliases`

- **Type**: `[]string`
- **Description**: Alternative names for the task

```yaml
tasks:
  build:
    aliases: [compile, make]
    cmds:
      - go build ./...
```

#### `sources`

- **Type**: `[]string` or `[]Glob`
- **Description**: Source files to monitor for changes

```yaml
tasks:
  build:
    sources:
      - '**/*.go'
      - go.mod
      # With exclusions
      - exclude: '**/*_test.go'
    cmds:
      - go build ./...
```

#### `generates`

- **Type**: `[]string` or `[]Glob`
- **Description**: Files generated by this task

```yaml
tasks:
  build:
    sources: ['**/*.go']
    generates:
      - './app'
      - exclude: '*.debug'
    cmds:
      - go build -o app ./cmd
```

#### `status`

- **Type**: `[]string`
- **Description**: Commands to check if task should run

```yaml
tasks:
  install-deps:
    status:
      - test -f node_modules/.installed
    cmds:
      - npm install
      - touch node_modules/.installed
```

#### `preconditions`

- **Type**: `[]Precondition`
- **Description**: Conditions that must be met before running

```yaml
tasks:
  # Simple precondition (shorthand)
  build:
    preconditions:
      - test -d ./src
    cmds:
      - go build ./...

  # Preconditions with custom messages
  deploy:
    preconditions:
      - sh: test -n "$API_KEY"
        msg: 'API_KEY environment variable is required'
      - sh: test -f ./app
        msg: "Application binary not found. Run 'task build' first."
    cmds:
      - ./deploy.sh
```

### `dir`

- **Type**: `string`
- **Description**: The directory in which this task should run
- **Default**: If the task is in the root Taskfile, the default `dir` is
  `ROOT_DIR`. For included Taskfiles, the default `dir` is the value specified in
  their respective `includes.*.dir` field (if any).

```yaml
tasks:
  current-dir:
    dir: '{{.USER_WORKING_DIR}}'
    cmd: pwd
```

#### `requires`

- **Type**: `Requires`
- **Description**: Required variables with optional enums

```yaml
tasks:
  # Simple requirements
  deploy:
    requires:
      vars: [API_KEY, ENVIRONMENT]
    cmds:
      - ./deploy.sh

  # Requirements with enum validation
  advanced-deploy:
    requires:
      vars:
        - API_KEY
        - name: ENVIRONMENT
          enum: [development, staging, production]
        - name: LOG_LEVEL
          enum: [debug, info, warn, error]
    cmds:
      - echo "Deploying to {{.ENVIRONMENT}} with log level {{.LOG_LEVEL}}"
      - ./deploy.sh
```

#### `watch`

- **Type**: `bool`
- **Default**: `false`
- **Description**: Automatically run task in watch mode

```yaml
tasks:
  dev:
    watch: true
    cmds:
      - npm run dev
```

#### `platforms`

- **Type**: `[]string`
- **Description**: Platforms where this task should run

```yaml
tasks:
  windows-build:
    platforms: [windows]
    cmds:
      - go build -o app.exe ./cmd

  unix-build:
    platforms: [linux, darwin]
    cmds:
      - go build -o app ./cmd
```

## Command

Individual command configuration within a task.

### Basic Commands

```yaml
tasks:
  example:
    cmds:
      - echo "Simple command"
      - ls -la
```

### Command Object

```yaml
tasks:
  example:
    cmds:
      - cmd: echo "Hello World"
        silent: true
        ignore_error: false
        platforms: [linux, darwin]
        set: [errexit]
        shopt: [globstar]
```

### Task References

```yaml
tasks:
  example:
    cmds:
      - task: other-task
        vars:
          PARAM: value
        silent: false
```

### Deferred Commands

```yaml
tasks:
  with-cleanup:
    cmds:
      - echo "Starting work"
      # Deferred command string
      - defer: echo "Cleaning up"
      # Deferred task reference
      - defer:
          task: cleanup-task
          vars:
            CLEANUP_MODE: full
```

### For Loops

#### Loop Over List

```yaml
tasks:
  greet-all:
    cmds:
      - for: [alice, bob, charlie]
        cmd: echo "Hello {{.ITEM}}"
```

#### Loop Over Sources/Generates

```yaml
tasks:
  process-files:
    sources: ['*.txt']
    cmds:
      - for: sources
        cmd: wc -l {{.ITEM}}
      - for: generates
        cmd: gzip {{.ITEM}}
```

#### Loop Over Variable

```yaml
tasks:
  process-items:
    vars:
      ITEMS: 'item1,item2,item3'
    cmds:
      - for:
          var: ITEMS
          split: ','
          as: CURRENT
        cmd: echo "Processing {{.CURRENT}}"
```

#### Loop Over Matrix

```yaml
tasks:
  test-matrix:
    cmds:
      - for:
          matrix:
            OS: [linux, windows, darwin]
            ARCH: [amd64, arm64]
        cmd: echo "Testing {{.ITEM.OS}}/{{.ITEM.ARCH}}"
```

#### Loop in Dependencies

```yaml
tasks:
  build-all:
    deps:
      - for: [frontend, backend, worker]
        task: build
        vars:
          SERVICE: '{{.ITEM}}'
```

## Shell Options

### Set Options

Available `set` options for POSIX shell features:

- `allexport` / `a` - Export all variables
- `errexit` / `e` - Exit on error
- `noexec` / `n` - Read commands but don't execute
- `noglob` / `f` - Disable pathname expansion
- `nounset` / `u` - Error on undefined variables
- `xtrace` / `x` - Print commands before execution
- `pipefail` - Pipe failures propagate

```yaml
# Global level
set: [errexit, nounset, pipefail]

tasks:
  debug:
    # Task level
    set: [xtrace]
    cmds:
      - cmd: echo "This will be traced"
        # Command level
        set: [noexec]
```

### Shopt Options

Available `shopt` options for Bash features:

- `expand_aliases` - Enable alias expansion
- `globstar` - Enable `**` recursive globbing
- `nullglob` - Null glob expansion

```yaml
# Global level
shopt: [globstar]

tasks:
  find-files:
    # Task level
    shopt: [nullglob]
    cmds:
      - cmd: ls **/*.go
        # Command level
        shopt: [globstar]
```
