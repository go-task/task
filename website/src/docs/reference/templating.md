---
title: Templating Reference
description:
  Comprehensive guide to Task's templating system with Go text/template, special
  variables, and available functions
outline: deep
---

# Templating Reference

Task's templating engine uses Go's
[text/template](https://pkg.go.dev/text/template) package to interpolate values.
This reference covers the main features and all available functions for creating
dynamic Taskfiles. Most of the provided functions come from the
[slim-sprig](https://sprig.taskfile.dev/) library.

## Basic Usage

Most string values in Task can be templated using double curly braces
<span v-pre>`{{` and `}}`</span>. Anything inside the braces is executed as a Go
template.

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

Task provides special variables that are always available in templates. These
override any user-defined variables with the same name.

### CLI

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

#### `CLI_ASSUME_YES`

- **Type**: `bool`
- **Description**: Whether `--yes` flag was set

### Task

#### `TASK`

- **Type**: `string`
- **Description**: Name of the current task

```yaml
tasks:
  build:
    cmds:
      - echo "Running task {{.TASK}}"
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
      - echo "Root {{.ROOT_DIR}}"
      - echo "Current {{.TASKFILE_DIR}}"
      - echo "Working {{.USER_WORKING_DIR}}"
```

### Status

#### `CHECKSUM`

- **Type**: `string`
- **Description**: Checksum of files in `sources` (only in `status` with
  `checksum` method)

#### `TIMESTAMP`

- **Type**: `time.Time`
- **Description**: Greatest timestamp of files in `sources` (only in `status`
  with `timestamp` method)

```yaml
tasks:
  build:
    method: checksum
    sources: ['**/*.go']
    status:
      - test "{{.CHECKSUM}}" = "$(cat .last-checksum)"
    cmds:
      - go build ./...
      - echo "{{.CHECKSUM}}" > .last-checksum
```

### Loop

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

### Defer

#### `EXIT_CODE`

- **Type**: `int`
- **Description**: Failed command exit code (only in `defer`, only when
  non-zero)

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

### System

#### `TASK_VERSION`

- **Type**: `string`
- **Description**: Current version of Task

```yaml
tasks:
  version:
    cmds:
      - echo "Using Task {{.TASK_VERSION}}"
```

## Available Functions

Task provides a comprehensive set of functions for templating. Functions can be chained using pipes (`|`) and combined for powerful templating capabilities.

### Logic and Control Flow

#### `and`, `or`, `not`

Boolean operations for conditional logic

```yaml
tasks:
  conditional:
    vars:
      DEBUG: true
      VERBOSE: false
      PRODUCTION: false
    cmds:
      - echo "{{if and .DEBUG .VERBOSE}}Debug mode with verbose{{end}}"
      - echo "{{if or .DEBUG .VERBOSE}}Some kind of debug{{end}}"
      - echo "{{if not .PRODUCTION}}Development build{{end}}"
```

#### `eq`, `ne`, `lt`, `le`, `gt`, `ge`

Comparison operations

```yaml
tasks:
  compare:
    vars:
      VERSION: 3
    cmds:
      - echo "{{if gt .VERSION 2}}Version 3 or higher{{end}}"
      - echo "{{if eq .VERSION 3}}Exactly version 3{{end}}"
```

### Data Access and Manipulation

#### `index`

Access array/map elements by index or key

```yaml
tasks:
  access:
    vars:
      SERVICES: [api, web, worker]
      CONFIG:
        map:
          database: postgres
          port: 5432
    cmds:
      - echo "First service {{index .SERVICES 0}}"
      - echo "Database {{index .CONFIG "database"}}"
```

#### `len`

Get length of arrays, maps, or strings

```yaml
tasks:
  length:
    vars:
      ITEMS: [a, b, c, d]
      TEXT: "Hello World"
    cmds:
      - echo "Found {{len .ITEMS}} items"
      - echo "Text has {{len .TEXT}} characters"
```

#### `slice`

Extract a portion of an array or string

```yaml
tasks:
  slice-demo:
    vars:
      ITEMS: [a, b, c, d, e]
    cmds:
      - echo "{{slice .ITEMS 1 3}}"     # [b c]
```

### String Functions

#### Basic String Operations

```yaml
tasks:
  string-basic:
    vars:
      MESSAGE: '  Hello World  '
      NAME: 'john doe'
      TEXT: "Hello World"
    cmds:
      - echo "{{.MESSAGE | trim}}"          # "Hello World"
      - echo "{{.NAME | title}}"            # "John Doe"
      - echo "{{.NAME | upper}}"            # "JOHN DOE"
      - echo "{{.MESSAGE | lower}}"         # "hello world"
      - echo "{{.NAME | trunc 4}}"          # "john"
      - echo "{{"test" | repeat 3}}"        # "testtesttest"
      - echo "{{.TEXT | substr 0 5}}"       # "Hello"
```

#### String Testing and Searching

```yaml
tasks:
  string-test:
    vars:
      FILENAME: 'app.tar.gz'
      EMAIL: 'user@example.com'
    cmds:
      - echo "{{.FILENAME | hasPrefix "app"}}"    # true
      - echo "{{.FILENAME | hasSuffix ".gz"}}"    # true
      - echo "{{.EMAIL | contains "@"}}"          # true
```

#### String Replacement and Formatting

```yaml
tasks:
  string-format:
    vars:
      TEXT: 'Hello, World!'
      UNSAFE: 'file with spaces.txt'
    cmds:
      - echo "{{.TEXT | replace "," ""}}"         # "Hello World!"
      - echo "{{.TEXT | quote}}"                  # "\"Hello, World!\""
      - echo "{{.UNSAFE | shellQuote}}"           # Shell-safe quoting
      - echo "{{.UNSAFE | q}}"                    # Short alias for shellQuote
```

#### Regular Expressions

```yaml
tasks:
  regex:
    vars:
      EMAIL: 'user@example.com'
      TEXT: 'abc123def456'
    cmds:
      - echo "{{regexMatch "@" .EMAIL}}"                    # true
      - echo "{{regexFind "[0-9]+" .TEXT}}"                # "123"
      - echo "{{regexFindAll "[0-9]+" .TEXT -1}}"          # ["123", "456"]
      - echo "{{regexReplaceAll "[0-9]+" .TEXT "X"}}"      # "abcXdefX"
```

### List Functions

#### List Access and Basic Operations

```yaml
tasks:
  list-basic:
    vars:
      ITEMS: ["apple", "banana", "cherry", "date"]
    cmds:
      - echo "First {{.ITEMS | first}}"          # "apple"
      - echo "Last {{.ITEMS | last}}"            # "date"
      - echo "Rest {{.ITEMS | rest}}"            # ["banana", "cherry", "date"]
      - echo "Initial {{.ITEMS | initial}}"      # ["apple", "banana", "cherry"]
      - echo "Length {{.ITEMS | len}}"           # 4
```

#### List Manipulation

```yaml
tasks:
  list-manipulate:
    vars:
      NUMBERS: [3, 1, 4, 1, 5, 9, 1]
      FRUITS: ["apple", "banana"]
    cmds:
      - echo "{{.NUMBERS | uniq}}"                       # [3, 1, 4, 5, 9]
      - echo "{{.NUMBERS | sortAlpha}}"                  # [1, 1, 1, 3, 4, 5, 9]
      - echo"'{{append .FRUITS "cherry"}}""              # ["apple", "banana", "cherry"]
      - echo "{{ without .NUMBERS 1}}"                   # [3, 4, 5, 9]
      - echo "{{.NUMBERS | has 5}}"                      # true
```

#### String Lists

```yaml
tasks:
  string-lists:
    vars:
      CSV: 'apple,banana,cherry'
      WORDS: ['hello', 'world', 'from', 'task']
      MULTILINE: |
        line1
        line2
        line3
    cmds:
      - echo "{{.CSV | splitList ","}}"           # ["apple", "banana", "cherry"]
      - echo "{{.WORDS | join " "}}"              # "hello world from task"
      - echo "{{.WORDS | sortAlpha}}"             # ["from", "hello", "task", "world"]
      - echo "{{.MULTILINE | splitLines}}"        # Split on newlines (Unix/Windows)
      - echo "{{.MULTILINE | catLines}}"          # Replace newlines with spaces
```

#### Shell Argument Parsing

```yaml
tasks:
  shell-args:
    vars:
      ARGS: 'file1.txt -v --output="result file.txt"'
    cmds:
      - |
        {{range .ARGS | splitArgs}}
        echo "Arg: {{.}}"
        {{end}}
```

### Math Functions

```yaml
tasks:
  math:
    vars:
      A: 10
      B: 3
      NUMBERS: [1, 5, 3, 9, 2]
    cmds:
      - echo "Addition {{add .A .B}}"            # 13
      - echo "Subtraction {{sub .A .B}}"         # 7
      - echo "Multiplication {{mul .A .B}}"      # 30
      - echo "Division {{div .A .B}}"            # 3
      - echo "Modulo {{mod .A .B}}"              # 1
      - echo "Maximum {{.NUMBERS | max}}"        # 9
      - echo "Minimum {{.NUMBERS | min}}"        # 1
      - echo "Random 1-99 {{randInt 1 100}}"     # Random number
      - echo "Random 0-999 {{randIntN 1000}}"    # Random number 0-999
```

### Date and Time Functions

```yaml
tasks:
  date-time:
    vars:
      BUILD_DATE: "2023-12-25"
    cmds:
      - echo "Now {{now | date "2006-01-02 15:04:05"}}"
      - echo {{ toDate "2006-01-02" .BUILD_DATE }}
      - echo "Build {{.BUILD_DATE | toDate "2006-01-02" | date "Jan 2, 2006"}}"
      - echo "Unix timestamp {{now | unixEpoch}}"
      - echo "Duration ago {{now | ago}}"
```

### System Functions

#### Platform Information

```yaml
tasks:
  platform:
    cmds:
      - echo "OS {{OS}}"                         # linux, darwin, windows, etc.
      - echo "Architecture {{ARCH}}"             # amd64, arm64, etc.
      - echo "CPU cores {{numCPU}}"              # Number of CPU cores
      - echo "Building for {{OS}}/{{ARCH}}"
```

#### Path Functions

```yaml
tasks:
  paths:
    vars:
      WIN_PATH: 'C:\Users\name\file.txt'
      OUTPUT_DIR: 'dist'
      BINARY_NAME: 'myapp'
    cmds:
      - echo "{{.WIN_PATH | toSlash}}"                          # Convert to forward slashes
      - echo "{{.WIN_PATH | fromSlash}}"                        # Convert to OS-specific slashes
      - echo "{{joinPath .OUTPUT_DIR .BINARY_NAME}}"            # Join path elements
      - echo "Relative {{relPath .ROOT_DIR .TASKFILE_DIR}}"    # Get relative path
```

### Data Structure Functions

#### Dictionary Operations

```yaml
tasks:
  dict:
    vars:
      CONFIG:
        map:
          database: postgres
          port: 5432
          ssl: true
    cmds:
      - echo "Database {{get .CONFIG "database"}}"
      - echo "Database {{"database" | get .CONFIG}}"
      - echo "Keys {{.CONFIG | keys}}"
      - echo "Keys {{keys .CONFIG }}"
      - echo "Has SSL {{hasKey .CONFIG "ssl"}}"
      - echo "{{dict "env" "prod" "debug" false}}"
```

#### Merging and Combining

```yaml
tasks:
  merge:
    vars:
      BASE_CONFIG:
        map:
          timeout: 30
          retries: 3
      USER_CONFIG:
        map:
          timeout: 60
          debug: true
    cmds:
      - echo "{{merge .BASE_CONFIG .USER_CONFIG | toJson}}"
```

### Default Values and Coalescing

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
      - echo "Is empty {{empty .ITEMS}}"                     # true
```

### Encoding and Serialization

#### JSON

```yaml
tasks:
  json:
    vars:
      DATA:
        map:
          name: 'Task'
          version: '3.0'
      JSON_STRING: '{"key": "value", "number": 42}'
    cmds:
      - echo "{{.DATA | toJson}}"
      - echo "{{.DATA | toPrettyJson}}"
      - echo "{{.JSON_STRING | fromJson }}"
```

#### YAML

```yaml
tasks:
  yaml:
    vars:
      CONFIG:
        map:
          database:
            host: localhost
            port: 5432
      YAML_STRING: |
        key: value
        items:
          - one
          - two
    cmds:
      - echo "{{.CONFIG | toYaml}}"
      - echo "{{.YAML_STRING | fromYaml}}"
```

#### Base64

```yaml
tasks:
  base64:
    vars:
      SECRET: 'my-secret-key'
    cmds:
      - echo "{{.SECRET | b64enc}}"               # Encode to base64
      - echo "{{"bXktc2VjcmV0LWtleQ==" | b64dec}}"   # Decode from base64
```

### Type Conversion

```yaml
tasks:
  convert:
    vars:
      NUM_STR: '42'
      FLOAT_STR: '3.14'
      BOOL_STR: 'true'
      ITEMS: [1, 2, 3]
    cmds:
      - echo "{{.NUM_STR | atoi | add 8}}"        # String to int: 50
      - echo "{{.FLOAT_STR | float64}}"           # String to float: 3.14
      - echo "{{.ITEMS | toStrings}}"             # Convert to strings: ["1", "2", "3"]
```

### Utility Functions

#### UUID Generation

```yaml
tasks:
  generate:
    vars:
      DEPLOYMENT_ID: "{{uuid}}"
    cmds:
      - echo "Deployment ID {{.DEPLOYMENT_ID}}"
```

#### Debugging

```yaml
tasks:
  debug:
    vars:
      COMPLEX_VAR:
        map:
          items: [1, 2, 3]
          nested:
            key: value
    cmds:
      - echo "{{spew .COMPLEX_VAR}}"              # Pretty-print for debugging
```

### Output Functions

#### Formatted Output

```yaml
tasks:
  output:
    vars:
      VERSION: "1.2.3"
      BUILD: 42
    cmds:
      - echo '{{print "Simple output"}}'
      - echo '{{printf "Version %s.%d" .VERSION .BUILD}}'
      - echo '{{println "With newline"}}'
```
