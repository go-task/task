---
slug: /api/
sidebar_position: 4
toc_min_heading_level: 2
toc_max_heading_level: 5
---

# Referência da API

## Linha de comando

O comando "task" tem a seguinte sintaxe:

```bash
task [--flags] [tasks...] [-- CLI_ARGS...]
```

:::tip

If `--` is given, all remaining arguments will be assigned to a special `CLI_ARGS` variable

:::

| Abreviação | Modificador                 | Tipo     | Predefinição                                                        | Descrição                                                                                                                                                                                                                                                       |
| ---------- | --------------------------- | -------- | ------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `-c`       | `--color`                   | `bool`   | `true`                                                              | Saída colorida. Habilitado por padrão. Defina o modificador como `false` ou use `NO_COLOR=1` para desativar.                                                                                                                                                    |
| `-C`       | `--concurrency`             | `int`    | `0`                                                                 | Limitar número de tarefas a serem executadas simultaneamente. Zero significa ilimitado.                                                                                                                                                                         |
| `-d`       | `--dir`                     | `string` | Pasta atual                                                         | Define a pasta de execução.                                                                                                                                                                                                                                     |
| `-n`       | `--dry`                     | `bool`   | `false`                                                             | Compila e imprime as tarefas na ordem em que elas seriam executadas, sem executá-las.                                                                                                                                                                           |
| `-x`       | `--exit-code`               | `bool`   | `false`                                                             | Faz com que o código de saída do comando sendo executado seja repassado pelo Task.                                                                                                                                                                              |
| `-f`       | `--force`                   | `bool`   | `false`                                                             | Força a execução mesmo quando a tarefa está atualizada.                                                                                                                                                                                                         |
| `-g`       | `--global`                  | `bool`   | `false`                                                             | Executa o Taskfile global, de `$HOME/Taskfile.{yml,yaml}`.                                                                                                                                                                                                      |
| `-h`       | `--help`                    | `bool`   | `false`                                                             | Mostra a ajuda do Task.                                                                                                                                                                                                                                         |
| `-i`       | `--init`                    | `bool`   | `false`                                                             | Cria um novo Taskfile.yml na pasta atual.                                                                                                                                                                                                                       |
| `-I`       | `--interval`                | `string` | `5s`                                                                | Define um intervalo de tempo diferente ao usar `--watch`, o padrão sendo 5 segundos. Este valor deve ser um [Go Duration](https://pkg.go.dev/time#ParseDuration) válido.                                                                                        |
| `-l`       | `--list`                    | `bool`   | `false`                                                             | Lista as tarefas com descrição do Taskfile atual.                                                                                                                                                                                                               |
| `-a`       | `--list-all`                | `bool`   | `false`                                                             | Lista todas as tarefas, com ou sem descrição.                                                                                                                                                                                                                   |
|            | `--sort`                    | `string` | `default`                                                           | Muda order das terafas quando listadas.<br />`default` - Ordem alfabética com as tarefas fo Taskfile raíz listadas por primeiro<br />`alphanumeric` - Alfabética<br />`none` - Sem ordenação (mantém a mesma ordem de declaração no Taskfile) |
|            | `--json`                    | `bool`   | `false`                                                             | Imprime a saída em [JSON](#json-output).                                                                                                                                                                                                                        |
| `-o`       | `--output`                  | `string` | O padrão é o que está definido no Taskfile, ou então `intervealed`. | Configura o estilo de saída: [`interleaved`/`group`/`prefixed`].                                                                                                                                                                                                |
|            | `--output-group-begin`      | `string` |                                                                     | Formato de mensagem a imprimir antes da saída agrupada de uma tarefa.                                                                                                                                                                                           |
|            | `--output-group-end`        | `string` |                                                                     | Formato de mensagem a imprimir depois da saída agrupada de uma tarefa.                                                                                                                                                                                          |
|            | `--output-group-error-only` | `bool`   | `false`                                                             | Oculta saída dos comandos que terminarem sem erro.                                                                                                                                                                                                              |
| `-p`       | `--parallel`                | `bool`   | `false`                                                             | Executa as tarefas fornecidas na linha de comando em paralelo.                                                                                                                                                                                                  |
| `-s`       | `--silent`                  | `bool`   | `false`                                                             | Desabilita impressão.                                                                                                                                                                                                                                           |
| `-y`       | `--yes`                     | `bool`   | `false`                                                             | Assuma "sim" como resposta a todos os prompts.                                                                                                                                                                                                                  |
|            | `--status`                  | `bool`   | `false`                                                             | Sai com código de saída diferente de zero se alguma das tarefas especificadas não estiver atualizada.                                                                                                                                                           |
|            | `--summary`                 | `bool`   | `false`                                                             | Mostrar resumo sobre uma tarefa.                                                                                                                                                                                                                                |
| `-t`       | `--taskfile`                | `string` | `Taskfile.yml` ou `Taskfile.yaml`                                   |                                                                                                                                                                                                                                                                 |
| `-v`       | `--verbose`                 | `bool`   | `false`                                                             | Habilita modo verboso.                                                                                                                                                                                                                                          |
|            | `--version`                 | `bool`   | `false`                                                             | Mostrar versão do Task.                                                                                                                                                                                                                                         |
| `-w`       | `--watch`                   | `bool`   | `false`                                                             | Habilita o monitoramento de tarefas.                                                                                                                                                                                                                            |

## Códigos de saída

O Task às vezes fecha com códigos de saída específicos. Estes códigos são divididos em três grupos com os seguintes intervalos:

- General errors (0-99)
- Taskfile errors (100-199)
- Task errors (200-299)

Uma lista completa dos códigos de saída e suas descrições podem ser encontradas abaixo:

| Código | Descrição                                                   |
| ------ | ----------------------------------------------------------- |
| 0      | Sucesso                                                     |
| 1      | Um erro desconhecido ocorreu                                |
| 100    | Nenhum Arquivo foi encontrado                               |
| 101    | Um arquivo Taskfile já existe ao tentar inicializar um      |
| 102    | O arquivo Taskfile é inválido ou não pode ser analisado     |
| 103    | A remote Taskfile could not be downlaoded                   |
| 104    | A remote Taskfile was not trusted by the user               |
| 105    | A remote Taskfile was could not be fetched securely         |
| 106    | No cache was found for a remote Taskfile in offline mode    |
| 107    | No schema version was defined in the Taskfile               |
| 200    | A tarefa especificada não pôde ser encontrada               |
| 201    | Ocorreu um erro ao executar um comando dentro de uma tarefa |
| 202    | O usuário tentou invocar uma tarefa que é interna           |
| 203    | Há várias tarefas com o mesmo nome ou apelido               |
| 204    | Uma tarefa foi chamada muitas vezes                         |
| 205    | A tarefa foi cancelada pelo usuário                         |
| 206    | A task was not executed due to missing required variables   |

Esses códigos também podem ser encontrados no repositório em [`errors/errors.go`](https://github.com/go-task/task/blob/main/errors/errors.go).

:::info

Quando o Task é executado com o modificador `-x`/`--exit-code`, o código de saída de todos os comandos falhados será passado para o usuário.

:::

## Saída em JSON

Quando estiver usando o modificador `--json` em combinação com o modificador `--list` ou `--list-all`, a saída será um objeto JSON com a seguinte estrutura:

```json
{
  "tasks": [
    {
      "name": "",
      "desc": "",
      "summary": "",
      "up_to_date": false,
      "location": {
        "line": 54,
        "column": 3,
        "taskfile": "/path/to/Taskfile.yml"
      }
    }
    // ...
  ],
  "location": "/path/to/Taskfile.yml"
}
```

## Variáveis Especiais

Há algumas variáveis especiais que são acessíveis via template:

| Variável           | Descrição                                                                                                                                                                     |
| ------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `CLI_ARGS`         | Contém todos os argumentos extras passados depois de `--` quando invocando o Task via linha de comando.                                                                       |
| `TASK`             | O nome da tarefa atual.                                                                                                                                                       |
| `ROOT_DIR`         | O caminho absoluto para o Taskfile raíz.                                                                                                                                      |
| `TASKFILE_DIR`     | O caminho absoluto para o Taskfile incluído.                                                                                                                                  |
| `USER_WORKING_DIR` | O caminho absoluto a partir do qual o comando `task` foi invocado.                                                                                                            |
| `CHECKSUM`         | O "checksum" dos arquivos listados em `sources`. Apenas disponível dentro do atributo `status` e se o método estiver configurado como `checksum`.                             |
| `TIMESTAMP`        | The date object of the greatest timestamp of the files listed in `sources`. Apenas disponível dentro do atributo `status` e se o método estiver configurado como `timestamp`. |
| `TASK_VERSION`     | A versão atual do Task.                                                                                                                                                       |
| `ITEM`             | The value of the current iteration when using the `for` property. Can be changed to a different variable name using `as:`.                                                    |

## ENV

Some environment variables can be overridden to adjust Task behavior.

| ENV                  | Padrão  | Descrição                                                                                                                                   |
| -------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `TASK_TEMP_DIR`      | `.task` | Caminho da pasta temporária. Pode ser um caminho relativo ao projeto como `tmp/task` ou um caminho absoluto como `/tmp/.task` ou `~/.task`. |
| `TASK_COLOR_RESET`   | `0`     | Cor utilizada para branco.                                                                                                                  |
| `TASK_COLOR_BLUE`    | `34`    | Cor utilizada para azul.                                                                                                                    |
| `TASK_COLOR_GREEN`   | `32`    | Cor utilizada para verde.                                                                                                                   |
| `TASK_COLOR_CYAN`    | `36`    | Cor utilizada para ciano.                                                                                                                   |
| `TASK_COLOR_YELLOW`  | `33`    | Cor utilizada para amarelo.                                                                                                                 |
| `TASK_COLOR_MAGENTA` | `35`    | Cor utilizada para magenta.                                                                                                                 |
| `TASK_COLOR_RED`     | `31`    | Cor utilizada para vermelho.                                                                                                                |
| `FORCE_COLOR`        |         | Forçar saída colorida no terminal.                                                                                                          |

## Esquema do Taskfile

| Atributo   | Tipo                               | Padrão        | Descrição                                                                                                                                                                                    |
| ---------- | ---------------------------------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `version`  | `string`                           |               | Versão do Taskfile. A versão mais atual é a `3`.                                                                                                                                             |
| `output`   | `string`                           | `interleaved` | Mode de saída. Opções disponíveis: `interleaved`, `group` e `prefixed`.                                                                                                                      |
| `method`   | `string`                           | `checksum`    | O método padrão deste Taskfile. Can be overridden in a task by task basis. Opções disponíveis: `checksum` (conteúdo dos arquivos), `timestamp` (data/hora de modificação) e `none` (nenhum). |
| `includes` | [`map[string]Include`](#include)   |               | Taskfiles adicionais a serem incluídos.                                                                                                                                                      |
| `vars`     | [`map[string]Variable`](#variable) |               | Um conjunto de variáveis globais.                                                                                                                                                            |
| `env`      | [`map[string]Variable`](#variable) |               | Um conjunto de variáveis de ambiente globais.                                                                                                                                                |
| `tasks`    | [`map[string]Task`](#task)         |               | Um conjunto de tarefas.                                                                                                                                                                      |
| `silent`   | `bool`                             | `false`       | Opção padrão para `silent` para este Taskfile. If `false`, can be overridden with `true` in a task by task basis.                                                                            |
| `dotenv`   | `[]string`                         |               | Uma lista de arquivos `.env` para serem incluídos.                                                                                                                                           |
| `run`      | `string`                           | `always`      | Opção padrão para `run` para este Taskfile. Opções disponíveis: `always` (sempre), `once` (uma vez) e `when_changed` (quando mudou).                                                         |
| `interval` | `string`                           | `5s`          | Configura um intervalo de tempo diferente para `--watch`, sendo que o padrão é de 5 segundos. Essa string deve ser um [Go Duration](https://pkg.go.dev/time#ParseDuration) válido.           |
| `set`      | `[]string`                         |               | Configura opções para o builtin [`set`](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html).                                                                            |
| `shopt`    | `[]string`                         |               | Configura opções para o builtin [`shopt`](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html).                                                                        |

### Include

| Atributo   | Tipo                  | Padrão                        | Descrição                                                                                                                                                                                                                                                |
| ---------- | --------------------- | ----------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `taskfile` | `string`              |                               | The path for the Taskfile or directory to be included. If a directory, Task will look for files named `Taskfile.yml` or `Taskfile.yaml` inside that directory. If a relative path, resolved relative to the directory containing the including Taskfile. |
| `dir`      | `string`              | The parent Taskfile directory | The working directory of the included tasks when run.                                                                                                                                                                                                    |
| `optional` | `bool`                | `false`                       | If `true`, no errors will be thrown if the specified file does not exist.                                                                                                                                                                                |
| `internal` | `bool`                | `false`                       | Stops any task in the included Taskfile from being callable on the command line. These commands will also be omitted from the output when used with `--list`.                                                                                            |
| `aliases`  | `[]string`            |                               | Alternative names for the namespace of the included Taskfile.                                                                                                                                                                                            |
| `vars`     | `map[string]Variable` |                               | A set of variables to apply to the included Taskfile.                                                                                                                                                                                                    |

:::info

Informing only a string like below is equivalent to setting that value to the `taskfile` attribute.

```yaml
includes:
  foo: ./path
```

:::

### Variable

| Atributo | Tipo     | Padrão | Descrição                                                                |
| -------- | -------- | ------ | ------------------------------------------------------------------------ |
| _itself_ | `string` |        | A static value that will be set to the variable.                         |
| `sh`     | `string` |        | A shell command. The output (`STDOUT`) will be assigned to the variable. |

:::info

Static and dynamic variables have different syntaxes, like below:

```yaml
vars:
  STATIC: static
  DYNAMIC:
    sh: echo "dynamic"
```

:::

### Task

| Atributo        | Tipo                               | Padrão                                                | Descrição                                                                                                                                                                                                                                                                                                |
| --------------- | ---------------------------------- | ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cmds`          | [`[]Command`](#command)            |                                                       | A list of shell commands to be executed.                                                                                                                                                                                                                                                                 |
| `deps`          | [`[]Dependency`](#dependency)      |                                                       | A list of dependencies of this task. Tasks defined here will run in parallel before this task.                                                                                                                                                                                                           |
| `label`         | `string`                           |                                                       | Overrides the name of the task in the output when a task is run. Supports variables.                                                                                                                                                                                                                     |
| `desc`          | `string`                           |                                                       | A short description of the task. This is displayed when calling `task --list`.                                                                                                                                                                                                                           |
| `prompt`        | `string`                           |                                                       | A prompt that will be presented before a task is run. Declining will cancel running the current and any subsequent tasks.                                                                                                                                                                                |
| `summary`       | `string`                           |                                                       | A longer description of the task. This is displayed when calling `task --summary [task]`.                                                                                                                                                                                                                |
| `aliases`       | `[]string`                         |                                                       | A list of alternative names by which the task can be called.                                                                                                                                                                                                                                             |
| `sources`       | `[]string`                         |                                                       | A list of sources to check before running this task. Relevant for `checksum` and `timestamp` methods. Can be file paths or star globs.                                                                                                                                                                   |
| `generates`     | `[]string`                         |                                                       | A list of files meant to be generated by this task. Relevant for `timestamp` method. Can be file paths or star globs.                                                                                                                                                                                    |
| `status`        | `[]string`                         |                                                       | A list of commands to check if this task should run. The task is skipped otherwise. This overrides `method`, `sources` and `generates`.                                                                                                                                                                  |
| `requires`      | `[]string`                         |                                                       | A list of variables which should be set if this task is to run, if any of these variables are unset the task will error and not run.                                                                                                                                                                     |
| `preconditions` | [`[]Precondition`](#precondition)  |                                                       | A list of commands to check if this task should run. If a condition is not met, the task will error.                                                                                                                                                                                                     |
| `requires`      | [`Requires`](#requires)            |                                                       | A list of required variables which should be set if this task is to run, if any variables listed are unset the task will error and not run.                                                                                                                                                              |
| `dir`           | `string`                           |                                                       | The directory in which this task should run. Defaults to the current working directory.                                                                                                                                                                                                                  |
| `vars`          | [`map[string]Variable`](#variable) |                                                       | A set of variables that can be used in the task.                                                                                                                                                                                                                                                         |
| `env`           | [`map[string]Variable`](#variable) |                                                       | A set of environment variables that will be made available to shell commands.                                                                                                                                                                                                                            |
| `dotenv`        | `[]string`                         |                                                       | A list of `.env` file paths to be parsed.                                                                                                                                                                                                                                                                |
| `silent`        | `bool`                             | `false`                                               | Hides task name and command from output. The command's output will still be redirected to `STDOUT` and `STDERR`. When combined with the `--list` flag, task descriptions will be hidden.                                                                                                                 |
| `interactive`   | `bool`                             | `false`                                               | Tells task that the command is interactive.                                                                                                                                                                                                                                                              |
| `internal`      | `bool`                             | `false`                                               | Stops a task from being callable on the command line. It will also be omitted from the output when used with `--list`.                                                                                                                                                                                   |
| `method`        | `string`                           | `checksum`                                            | Defines which method is used to check the task is up-to-date. `timestamp` will compare the timestamp of the sources and generates files. `checksum` will check the checksum (You probably want to ignore the .task folder in your .gitignore file). `none` skips any validation and always run the task. |
| `prefix`        | `string`                           |                                                       | Defines a string to prefix the output of tasks running in parallel. Only used when the output mode is `prefixed`.                                                                                                                                                                                        |
| `ignore_error`  | `bool`                             | `false`                                               | Continue execution if errors happen while executing commands.                                                                                                                                                                                                                                            |
| `run`           | `string`                           | The one declared globally in the Taskfile or `always` | Specifies whether the task should run again or not if called more than once. Available options: `always`, `once` and `when_changed`.                                                                                                                                                                     |
| `platforms`     | `[]string`                         | All platforms                                         | Specifies which platforms the task should be run on. [Valid GOOS and GOARCH values allowed](https://github.com/golang/go/blob/main/src/go/build/syslist.go). Task will be skipped otherwise.                                                                                                             |
| `set`           | `[]string`                         |                                                       | Specify options for the [`set` builtin](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html).                                                                                                                                                                                        |
| `shopt`         | `[]string`                         |                                                       | Specify option for the [`shopt` builtin](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html).                                                                                                                                                                                     |

:::info

These alternative syntaxes are available. They will set the given values to `cmds` and everything else will be set to their default values:

```yaml
tasks:
  foo: echo "foo"

  foobar:
    - echo "foo"
    - echo "bar"

  baz:
    cmd: echo "baz"
```

:::

#### Command

| Atributo       | Tipo                               | Padrão        | Descrição                                                                                                                                                                                          |
| -------------- | ---------------------------------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cmd`          | `string`                           |               | The shell command to be executed.                                                                                                                                                                  |
| `task`         | `string`                           |               | Set this to trigger execution of another task instead of running a command. This cannot be set together with `cmd`.                                                                                |
| `for`          | [`For`](#for)                      |               | Runs the command once for each given value.                                                                                                                                                        |
| `silent`       | `bool`                             | `false`       | Skips some output for this command. Note that STDOUT and STDERR of the commands will still be redirected.                                                                                          |
| `vars`         | [`map[string]Variable`](#variable) |               | Optional additional variables to be passed to the referenced task. Only relevant when setting `task` instead of `cmd`.                                                                             |
| `ignore_error` | `bool`                             | `false`       | Continue execution if errors happen while executing the command.                                                                                                                                   |
| `defer`        | `string`                           |               | Alternative to `cmd`, but schedules the command to be executed at the end of this task instead of immediately. This cannot be used together with `cmd`.                                            |
| `platforms`    | `[]string`                         | All platforms | Specifies which platforms the command should be run on. [Valid GOOS and GOARCH values allowed](https://github.com/golang/go/blob/main/src/go/build/syslist.go). Command will be skipped otherwise. |
| `set`          | `[]string`                         |               | Specify options for the [`set` builtin](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html).                                                                                  |
| `shopt`        | `[]string`                         |               | Specify option for the [`shopt` builtin](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html).                                                                               |

:::info

If given as a a string, the value will be assigned to `cmd`:

```yaml
tasks:
  foo:
    cmds:
      - echo "foo"
      - echo "bar"
```

:::

#### Dependency

| Atributo | Tipo                               | Padrão  | Descrição                                                                                                        |
| -------- | ---------------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------- |
| `task`   | `string`                           |         | The task to be execute as a dependency.                                                                          |
| `vars`   | [`map[string]Variable`](#variable) |         | Optional additional variables to be passed to this task.                                                         |
| `silent` | `bool`                             | `false` | Hides task name and command from output. The command's output will still be redirected to `STDOUT` and `STDERR`. |

:::tip

If you don't want to set additional variables, it's enough to declare the dependency as a list of strings (they will be assigned to `task`):

```yaml
tasks:
  foo:
    deps: [foo, bar]
```

:::

#### For

The `for` parameter can be defined as a string, a list of strings or a map. If it is defined as a string, you can give it any of the following values:

- `source` - Will run the command for each source file defined on the task. (Glob patterns will be resolved, so `*.go` will run for every Go file that matches).

If it is defined as a list of strings, the command will be run for each value.

Finally, the `for` parameter can be defined as a map when you want to use a variable to define the values to loop over:

| Atributo | Tipo     | Padrão           | Descrição                                    |
| -------- | -------- | ---------------- | -------------------------------------------- |
| `var`    | `string` |                  | The name of the variable to use as an input. |
| `split`  | `string` | (any whitespace) | What string the variable should be split on. |
| `as`     | `string` | `ITEM`           | The name of the iterator variable.           |

#### Precondition

| Attribute | Type     | Default | Description                                                                                                  |
| --------- | -------- | ------- | ------------------------------------------------------------------------------------------------------------ |
| `sh`      | `string` |         | Command to be executed. If a non-zero exit code is returned, the task errors without executing its commands. |
| `msg`     | `string` |         | Optional message to print if the precondition isn't met.                                                     |

:::tip

If you don't want to set a different message, you can declare a precondition like this and the value will be assigned to `sh`:

```yaml
tasks:
  foo:
    precondition: test -f Taskfile.yml
```

:::

#### Requires

| Attribute | Type       | Default | Description                                                                                        |
| --------- | ---------- | ------- | -------------------------------------------------------------------------------------------------- |
| `vars`    | `[]string` |         | List of variable or environment variable names that must be set if this task is to execute and run |
