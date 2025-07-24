---
title: Templating Reference
description: Comprehensive guide to Task's templating system with Go text/template, special variables, and available functions
outline: deep
---

# Templating Reference

Task's templating engine uses Go's [text/template](https://pkg.go.dev/text/template) package to interpolate values. This reference covers the main features and all available functions for creating dynamic Taskfiles.

## Basic Usage

Most string values in Task can be templated using double curly braces <span v-pre>`{{` and `}}`</span>. Anything inside the braces is executed as a Go template.

### Simple Variable Interpolation

```yaml
version: '3'

tasks:
  hello:
    vars:
      MESSAGE: 'Hello, World!'
    cmds:
      - 'echo {{.MESSAGE}}'
```

**Output:**
```
Hello, World!
```

### Conditional Logic

```yaml
version: '3'

tasks:
  maybe-happy:
    vars:
      SMILE: ':\)'
      FROWN: ':\('
      HAPPY: true
    cmds:
      - 'echo {{if .HAPPY}}{{.SMILE}}{{else}}{{.FROWN}}{{end}}'
```

**Output:**
```
:)
```

### Function Calls and Pipes

```yaml
version: '3'

tasks:
  uniq:
    vars:
      NUMBERS: '0,1,1,1,2,2,3'
    cmds:
      - 'echo {{splitList "," .NUMBERS | uniq | join ", "}}'
```

**Output:**
```
0, 1, 2, 3
```

### Control Flow with Loops

```yaml
version: '3'

tasks:
  loop:
    vars:
      NUMBERS: [0, 1, 1, 1, 2, 2, 3]
    cmds:
      - |
        {{range $index, $num := .NUMBERS}}
        {{if gt $num 1}}{{break}}{{end}}
        echo {{$index}}: {{$num}}
        {{end}}
```

**Output:**
```
0: 0
1: 1
2: 1
3: 1
```

## Special Variables

Task provides special variables that are always available in templates. These override any user-defined variables with the same name.

### CLI Context

#### `CLI_ARGS`
- **Type**: `string`
- **Description**: All extra arguments passed after `--` as a string

```yaml
tasks:
  test:
    cmds:
      - go test {{.CLI_ARGS}}
```

```bash
task test -- -v -race
# Runs: go test -v -race
```

#### `CLI_ARGS_LIST`
- **Type**: `[]string`
- **Description**: All extra arguments passed after `--` as a shell parsed list

```yaml
tasks:
  docker-run:
    cmds:
      - docker run {{range .CLI_ARGS_LIST}}{{.}} {{end}}myapp
```

#### `CLI_FORCE`
- **Type**: `bool`
- **Description**: Whether `--force` or `--force-all` flags were set

```yaml
tasks:
  deploy:
    cmds:
      - |
        {{if .CLI_FORCE}}
        echo "Force deployment enabled"
        {{end}}
        ./deploy.sh
```

#### `CLI_SILENT`
- **Type**: `bool`
- **Description**: Whether `--silent` flag was set

#### `CLI_VERBOSE`
- **Type**: `bool`
- **Description**: Whether `--verbose` flag was set

#### `CLI_OFFLINE`
- **Type**: `bool`
- **Description**: Whether `--offline` flag was set

### Task Context

#### `TASK`
- **Type**: `string`
- **Description**: Name of the current task

```yaml
tasks:
  build:
    cmds:
      - echo "Running task: {{.TASK}}"
```

#### `ALIAS`
- **Type**: `string`
- **Description**: Alias used for the current task, otherwise matches `TASK`

#### `TASK_EXE`
- **Type**: `string`
- **Description**: Task executable name or path

```yaml
tasks:
  self-update:
    cmds:
      - echo "Updating {{.TASK_EXE}}"
```

### File Paths

#### `ROOT_TASKFILE`
- **Type**: `string`
- **Description**: Absolute path of the root Taskfile

#### `ROOT_DIR`
- **Type**: `string`
- **Description**: Absolute path of the root Taskfile directory

#### `TASKFILE`
- **Type**: `string`
- **Description**: Absolute path of the current (included) Taskfile

#### `TASKFILE_DIR`
- **Type**: `string`
- **Description**: Absolute path of the current Taskfile directory

#### `TASK_DIR`
- **Type**: `string`
- **Description**: Absolute path where the task is executed

#### `USER_WORKING_DIR`
- **Type**: `string`
- **Description**: Absolute path where `task` was called from

```yaml
tasks:
  info:
    cmds:
      - echo "Root: {{.ROOT_DIR}}"
      - echo "Current: {{.TASKFILE_DIR}}"
      - echo "Working: {{.USER_WORKING_DIR}}"
```

### Status Context

#### `CHECKSUM`
- **Type**: `string`
- **Description**: Checksum of files in `sources` (only in `status` with `checksum` method)

#### `TIMESTAMP`
- **Type**: `time.Time`
- **Description**: Greatest timestamp of files in `sources` (only in `status` with `timestamp` method)

```yaml
tasks:
  build:
    method: checksum
    sources: ["**/*.go"]
    status:
      - test "{{.CHECKSUM}}" = "$(cat .last-checksum)"
    cmds:
      - go build ./...
      - echo "{{.CHECKSUM}}" > .last-checksum
```

### Loop Context

#### `ITEM`
- **Type**: `any`
- **Description**: Current iteration value when using `for` property

```yaml
tasks:
  greet:
    cmds:
      - for: [alice, bob, charlie]
        cmd: echo "Hello {{.ITEM}}"
```

Can be renamed using `as`:

```yaml
tasks:
  greet:
    cmds:
      - for:
          var: NAMES
          as: NAME
        cmd: echo "Hello {{.NAME}}"
```

### Defer Context

#### `EXIT_CODE`
- **Type**: `int`
- **Description**: Failed command exit code (only in `defer`, only when non-zero)

```yaml
tasks:
  deploy:
    cmds:
      - ./deploy.sh
      - defer: |
          {{if .EXIT_CODE}}
          echo "Deployment failed with code {{.EXIT_CODE}}"
          ./rollback.sh
          {{end}}
```

### System Context

#### `TASK_VERSION`
- **Type**: `string`
- **Description**: Current version of Task

```yaml
tasks:
  version:
    cmds:
      - echo "Using Task {{.TASK_VERSION}}"
```

## Built-in Functions

These functions are provided by Go's [text/template](https://pkg.go.dev/text/template#hdr-Functions) package.

### Logic Functions

#### `and`
Boolean AND operation
```yaml
cmds:
  - echo "{{if and .DEBUG .VERBOSE}}Debug mode{{end}}"
```

#### `or`
Boolean OR operation
```yaml
cmds:
  - echo "{{if or .DEV .STAGING}}Non-production{{end}}"
```

#### `not`
Boolean negation
```yaml
cmds:
  - echo "{{if not .PRODUCTION}}Development build{{end}}"
```

### Data Access

#### `index`
Access array/map elements
```yaml
vars:
  SERVICES: [api, web, worker]
cmds:
  - echo "First service: {{index .SERVICES 0}}"
```

#### `len`
Get length of arrays, maps, or strings
```yaml
vars:
  ITEMS: [a, b, c, d]
cmds:
  - echo "Found {{len .ITEMS}} items"
```

#### `slice`
Get slice of array/string
```yaml
vars:
  ITEMS: [a, b, c, d, e]
cmds:
  - echo "{{slice .ITEMS 1 3}}" # [b c]
```

### Output Functions

#### `print`, `printf`, `println`
Formatted output functions
```yaml
cmds:
  - echo "{{printf "Version: %s.%d" .VERSION .BUILD}}"
```

## Slim-Sprig Functions

Task includes functions from [slim-sprig](https://go-task.github.io/slim-sprig/) for enhanced templating capabilities.

### String Functions

#### Basic String Operations

```yaml
tasks:
  string-demo:
    vars:
      MESSAGE: "  Hello World  "
      NAME: "john doe"
    cmds:
      - echo "{{.MESSAGE | trim}}"           # "Hello World"
      - echo "{{.NAME | title}}"             # "John Doe"
      - echo "{{.NAME | upper}}"             # "JOHN DOE"
      - echo "{{.MESSAGE | lower}}"          # "hello world"
```

#### String Testing

```yaml
tasks:
  check:
    vars:
      FILENAME: "app.tar.gz"
    cmds:
      - |
        {{if .FILENAME | hasPrefix "app"}}
        echo "Application file detected"
        {{end}}
      - |
        {{if .FILENAME | hasSuffix ".gz"}}
        echo "Compressed file detected"
        {{end}}
```

#### String Manipulation

```yaml
tasks:
  process:
    vars:
      TEXT: "Hello, World!"
    cmds:
      - echo "{{.TEXT | replace "," ""}}"     # "Hello World!"
      - echo "{{.TEXT | quote}}"             # "\"Hello, World!\""
      - echo "{{"test" | repeat 3}}"         # "testtesttest"
      - echo "{{.TEXT | trunc 5}}"           # "Hello"
```

#### Regular Expressions

```yaml
tasks:
  regex-demo:
    vars:
      EMAIL: "user@example.com"
      TEXT: "abc123def456"
    cmds:
      - echo "{{regexMatch "@" .EMAIL}}"                    # true
      - echo "{{regexFind "[0-9]+" .TEXT}}"                 # "123"
      - echo "{{regexFindAll "[0-9]+" .TEXT -1}}"           # ["123", "456"]
      - echo "{{regexReplaceAll "[0-9]+" .TEXT "X"}}"       # "abcXdefX"
```

### List Functions

#### List Creation and Access

```yaml
tasks:
  list-demo:
    vars:
      ITEMS: ["apple", "banana", "cherry", "date"]
    cmds:
      - echo "First: {{.ITEMS | first}}"     # "apple"
      - echo "Last: {{.ITEMS | last}}"       # "date"
      - echo "Rest: {{.ITEMS | rest}}"       # ["banana", "cherry", "date"]
      - echo "Initial: {{.ITEMS | initial}}" # ["apple", "banana", "cherry"]
```

#### List Manipulation

```yaml
tasks:
  manipulate:
    vars:
      NUMBERS: [3, 1, 4, 1, 5, 9, 1]
      FRUITS: ["apple", "banana"]
    cmds:
      - echo "{{.NUMBERS | uniq}}"                    # [3, 1, 4, 5, 9]
      - echo "{{.NUMBERS | sortAlpha}}"               # [1, 1, 1, 3, 4, 5, 9]
      - echo "{{.FRUITS | append "cherry"}}"          # ["apple", "banana", "cherry"]
      - echo "{{.NUMBERS | without 1}}"               # [3, 4, 5, 9]
```

#### String Lists

```yaml
tasks:
  string-lists:
    vars:
      CSV: "apple,banana,cherry"
      WORDS: ["hello", "world", "from", "task"]
    cmds:
      - echo "{{.CSV | splitList ","}}"          # ["apple", "banana", "cherry"]
      - echo "{{.WORDS | join " "}}"             # "hello world from task"
      - echo "{{.WORDS | sortAlpha}}"            # ["from", "hello", "task", "world"]
```

### Math Functions

```yaml
tasks:
  math-demo:
    vars:
      A: 10
      B: 3
      NUMBERS: [1, 5, 3, 9, 2]
    cmds:
      - echo "{{add .A .B}}"           # 13
      - echo "{{sub .A .B}}"           # 7
      - echo "{{mul .A .B}}"           # 30
      - echo "{{div .A .B}}"           # 3
      - echo "{{mod .A .B}}"           # 1
      - echo "{{.NUMBERS | max}}"      # 9
      - echo "{{.NUMBERS | min}}"      # 1
      - echo "{{randInt 1 100}}"       # Random number 1-99
```

### Date Functions

```yaml
tasks:
  date-demo:
    vars:
      BUILD_DATE: "2023-12-25T10:30:00Z"
    cmds:
      - echo "Now: {{now | date "2006-01-02 15:04:05"}}"
      - echo "Build: {{.BUILD_DATE | toDate | date "Jan 2, 2006"}}"
      - echo "Unix: {{now | unixEpoch}}"
      - echo "Duration: {{now | ago}}"
```

### Dictionary Functions

```yaml
tasks:
  dict-demo:
    vars:
      CONFIG:
        database: postgres
        port: 5432
        ssl: true
    cmds:
      - echo "DB: {{.CONFIG | get "database"}}"
      - echo "Keys: {{.CONFIG | keys}}"
      - echo "Has SSL: {{.CONFIG | hasKey "ssl"}}"
      - echo "{{dict "env" "prod" "debug" false}}"
```

### Default Functions

```yaml
tasks:
  defaults:
    vars:
      API_URL: ""
      DEBUG: false
      ITEMS: []
    cmds:
      - echo "{{.API_URL | default "http://localhost:8080"}}"
      - echo "{{.DEBUG | default true}}"
      - echo "{{.MISSING_VAR | default "fallback"}}"
      - echo "{{coalesce .API_URL .FALLBACK_URL "default"}}"
      - echo "Empty: {{empty .ITEMS}}"  # true
```

### Encoding Functions

```yaml
tasks:
  encoding:
    vars:
      DATA:
        name: "Task"
        version: "3.0"
    cmds:
      - echo "{{.DATA | toJson}}"
      - echo "{{.DATA | toPrettyJson}}"
      - echo "{{"hello" | b64enc}}"     # aGVsbG8=
      - echo "{{"aGVsbG8=" | b64dec}}"  # hello
```

### Type Conversion

```yaml
tasks:
  convert:
    vars:
      NUM_STR: "42"
      FLOAT_STR: "3.14"
      ITEMS: [1, 2, 3]
    cmds:
      - echo "{{.NUM_STR | atoi | add 8}}"      # 50
      - echo "{{.FLOAT_STR | float64}}"         # 3.14
      - echo "{{.ITEMS | toStrings}}"           # ["1", "2", "3"]
```

## Task-Specific Functions

Task provides additional functions for common operations.

### System Functions

#### `OS`
Get the operating system
```yaml
tasks:
  build:
    cmds:
      - |
        {{if eq OS "windows"}}
        go build -o app.exe
        {{else}}
        go build -o app
        {{end}}
```

#### `ARCH`
Get the system architecture
```yaml
tasks:
  info:
    cmds:
      - echo "Building for {{OS}}/{{ARCH}}"
```

#### `numCPU`
Get number of CPU cores
```yaml
tasks:
  test:
    cmds:
      - go test -parallel {{numCPU}} ./...
```

### Path Functions

#### `toSlash` / `fromSlash`
Convert path separators
```yaml
tasks:
  paths:
    vars:
      WIN_PATH: 'C:\Users\name\file.txt'
    cmds:
      - echo "{{.WIN_PATH | toSlash}}"    # C:/Users/name/file.txt (on Windows)
```

#### `joinPath`
Join path elements
```yaml
tasks:
  build:
    vars:
      OUTPUT_DIR: dist
      BINARY_NAME: myapp
    cmds:
      - go build -o {{joinPath .OUTPUT_DIR .BINARY_NAME}}
```

#### `relPath`
Get relative path
```yaml
tasks:
  info:
    cmds:
      - echo "Relative: {{relPath .ROOT_DIR .TASKFILE_DIR}}"
```

### String Processing

#### `splitLines`
Split on newlines (Unix and Windows)
```yaml
tasks:
  process:
    vars:
      MULTILINE: |
        line1
        line2
        line3
    cmds:
      - |
        {{range .MULTILINE | splitLines}}
        echo "Line: {{.}}"
        {{end}}
```

#### `catLines`
Replace newlines with spaces
```yaml
tasks:
  flatten:
    vars:
      MULTILINE: |
        hello
        world
    cmds:
      - echo "{{.MULTILINE | catLines}}"  # "hello world"
```

#### `shellQuote` (alias: `q`)
Quote for shell safety
```yaml
tasks:
  safe:
    vars:
      FILENAME: "file with spaces.txt"
    cmds:
      - ls -la {{.FILENAME | shellQuote}}
      - cat {{.FILENAME | q}}  # Short alias
```

#### `splitArgs`
Parse shell arguments
```yaml
tasks:
  parse:
    vars:
      ARGS: 'file1.txt -v --output="result file.txt"'
    cmds:
      - |
        {{range .ARGS | splitArgs}}
        echo "Arg: {{.}}"
        {{end}}
```

### Data Functions

#### `merge`
Merge maps
```yaml
tasks:
  config:
    vars:
      BASE_CONFIG:
        timeout: 30
        retries: 3
      USER_CONFIG:
        timeout: 60
        debug: true
    cmds:
      - echo "{{merge .BASE_CONFIG .USER_CONFIG | toJson}}"
```

#### `spew`
Debug variable contents
```yaml
tasks:
  debug:
    vars:
      COMPLEX_VAR:
        items: [1, 2, 3]
        nested:
          key: value
    cmds:
      - echo "{{spew .COMPLEX_VAR}}"
```

### YAML Functions

#### `fromYaml` / `toYaml`
YAML encoding/decoding
```yaml
tasks:
  yaml-demo:
    vars:
      CONFIG:
        database:
          host: localhost
          port: 5432
    cmds:
      - echo "{{.CONFIG | toYaml}}"
      - echo "{{.YAML_STRING | fromYaml | get "key"}}"
```

### Utility Functions

#### `uuid`
Generate UUID
```yaml
tasks:
  deploy:
    vars:
      DEPLOYMENT_ID: "{{uuid}}"
    cmds:
      - echo "Deployment ID: {{.DEPLOYMENT_ID}}"
```

#### `randIntN`
Generate random integer
```yaml
tasks:
  test:
    vars:
      RANDOM_PORT: "{{randIntN 9000 | add 1000}}"  # 1000-9999
    cmds:
      - echo "Using port: {{.RANDOM_PORT}}"
```

## Advanced Examples

### Dynamic Task Generation

```yaml
version: '3'

vars:
  SERVICES: [api, web, worker, scheduler]
  ENVIRONMENTS: [dev, staging, prod]

tasks:
  deploy-all:
    desc: Deploy all services to all environments
    deps:
      - for: '{{.SERVICES}}'
        task: deploy-service
        vars:
          SERVICE: '{{.ITEM}}'

  deploy-service:
    desc: Deploy a service to all environments
    requires:
      vars: [SERVICE]
    deps:
      - for: '{{.ENVIRONMENTS}}'
        task: deploy
        vars:
          SERVICE: '{{.SERVICE}}'
          ENV: '{{.ITEM}}'

  deploy:
    desc: Deploy service to specific environment
    requires:
      vars: [SERVICE, ENV]
    cmds:
      - echo "Deploying {{.SERVICE}} to {{.ENV}}"
      - |
        {{if eq .ENV "prod"}}
        echo "Production deployment - extra validation"
        ./validate-prod.sh {{.SERVICE}}
        {{end}}
      - ./deploy.sh {{.SERVICE}} {{.ENV}}
```

### Configuration Management

```yaml
version: '3'

vars:
  BASE_CONFIG:
    timeout: 30
    retries: 3
    logging: true

  DEV_CONFIG:
    debug: true
    timeout: 10

  PROD_CONFIG:
    debug: false
    timeout: 60
    ssl: true

tasks:
  start:
    vars:
      ENVIRONMENT: '{{.ENVIRONMENT | default "dev"}}'
      CONFIG: |
        {{if eq .ENVIRONMENT "prod"}}
        {{merge .BASE_CONFIG .PROD_CONFIG | toYaml}}
        {{else}}
        {{merge .BASE_CONFIG .DEV_CONFIG | toYaml}}
        {{end}}
    cmds:
      - echo "Starting in {{.ENVIRONMENT}} mode"
      - echo "{{.CONFIG}}" > config.yaml
      - ./app --config config.yaml
```

### Matrix Build with Conditional Logic

```yaml
version: '3'

vars:
  PLATFORMS:
    - os: linux
      arch: amd64
      cgo: "1"
    - os: linux
      arch: arm64
      cgo: "0"
    - os: windows
      arch: amd64
      cgo: "1"
    - os: darwin
      arch: amd64
      cgo: "1"
    - os: darwin
      arch: arm64
      cgo: "0"

tasks:
  build-all:
    desc: Build for all platforms
    cmds:
      - |
        {{range .PLATFORMS}}
        echo "Building for {{.os}}/{{.arch}} (CGO={{.cgo}})"
        GOOS={{.os}} GOARCH={{.arch}} CGO_ENABLED={{.cgo}} go build \
          -o dist/myapp-{{.os}}-{{.arch}}{{if eq .os "windows"}}.exe{{end}} \
          ./cmd/myapp
        {{end}}

  package:
    desc: Create platform-specific packages
    deps: [build-all]
    cmds:
      - |
        {{range .PLATFORMS}}
        {{$ext := ""}}
        {{if eq .os "windows"}}{{$ext = ".exe"}}{{end}}
        {{$archive := printf "myapp-%s-%s" .os .arch}}

        {{if eq .os "windows"}}
        echo "Creating Windows package: {{$archive}}.zip"
        zip -j dist/{{$archive}}.zip dist/myapp-{{.os}}-{{.arch}}{{$ext}}
        {{else}}
        echo "Creating Unix package: {{$archive}}.tar.gz"
        tar -czf dist/{{$archive}}.tar.gz -C dist myapp-{{.os}}-{{.arch}}{{$ext}}
        {{end}}
        {{end}}
```
