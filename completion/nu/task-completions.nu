# Nushell completions for Task (https://taskfile.dev).
#
# Nushell cannot source a script from stdin, so save this file where Nushell
# picks it up automatically:
#   mkdir ($nu.data-dir | path join "vendor/autoload")
#   task --completion nu | save --force ($nu.data-dir | path join "vendor/autoload/task-completions.nu")
#
# The file must not be named task.nu: Nushell refuses to export a known external
# named like its module, which would break `use task-completions.nu *`.

# Name or path of the Task executable, like the other completion scripts.
# The *completed* command is always `task`: an `extern` declaration requires a
# literal name. For a renamed executable, alias it instead: `alias go-task = task`.
def "nu-complete task-exe" [] {
  $env.TASK_EXE? | default "task"
}

def "nu-complete task-words" [context: string] {
  $context | split row --regex '\s+' | where {|word| $word != "" }
}

# Strips the quotes the user may have typed around a value and expands `~`,
# which Nushell does not do for a value coming from a variable.
def "nu-complete task-value" [value: string] {
  let unquoted = ($value | str trim --char '"' | str trim --char "'")
  if ($unquoted | str starts-with "~") {
    $unquoted | path expand --no-symlink
  } else {
    $unquoted
  }
}

# Rebuilds the flags deciding *which* Taskfile is read, so the task list follows
# the `-t/--taskfile`, `-d/--dir` and `-g/--global` already on the command line.
def "nu-complete task-scope" [words: list<string>] {
  mut scope: list<string> = []
  mut pending = ""

  for word in ($words | skip 1) {
    if $pending != "" {
      $scope = ($scope | append [$pending, (nu-complete task-value $word)])
      $pending = ""
      continue
    }

    let parts = ($word | split row "=")
    let name = ($parts | first)
    let inline = if ($parts | length) > 1 { $parts | skip 1 | str join "=" } else { null }

    if $name in ["-g", "--global"] {
      $scope = ($scope | append "--global")
    } else if $name in ["-t", "--taskfile", "-d", "--dir"] {
      let long = if $name in ["-t", "--taskfile"] { "--taskfile" } else { "--dir" }
      if $inline != null {
        $scope = ($scope | append [$long, (nu-complete task-value $inline)])
      } else {
        $pending = $long
      }
    }
  }

  $scope
}

# Lists the tasks of the targeted Taskfile. `--no-status` keeps completion fast:
# without it Task fingerprints every task's sources on each keystroke. Returns an
# empty list when Task exits non-zero (no Taskfile, invalid Taskfile).
def "nu-complete task-list" [words: list<string>] {
  let exe = (nu-complete task-exe)
  let args = [...(nu-complete task-scope $words) "--list-all" "--json" "--no-status"]
  let result = (try { do { ^$exe ...$args } | complete } catch { null })

  if ($result | is-empty) or $result.exit_code != 0 {
    return []
  }

  try { $result.stdout | from json | get tasks } catch { [] }
}

def "nu-complete task" [context: string] {
  let words = (nu-complete task-words $context)

  # Words after `--` are forwarded to the task as CLI_ARGS: stop offering task
  # names and let Nushell fall back to its own file completion.
  if "--" in $words {
    return null
  }

  let completions = (
    nu-complete task-list $words
    | each {|item|
      # `task` is the invocable name; `name` may be a display-only label.
      let name = ($item.task | str trim --right --char ':')
      let desc = ($item.desc? | default "")
      let aliases = (
        $item.aliases?
        | default []
        | each {|alias| {
          value: ($alias | str trim --right --char ':')
          description: (if ($desc | is-empty) { $"alias of ($name)" } else { $"($desc) \(alias of ($name)\)" })
        } }
      )
      [{ value: $name, description: $desc }] | append $aliases
    }
    | flatten
  )

  # `sort: false` keeps the order Task chose, which honours --sort and .taskrc.
  { options: { sort: false }, completions: $completions }
}

def "nu-complete task-shells" [] {
  ["bash", "zsh", "fish", "powershell", "nu"]
}

def "nu-complete task-output" [] {
  ["interleaved", "group", "prefixed"]
}

def "nu-complete task-sort" [] {
  ["default", "alphanumeric", "none"]
}

# Runs the specified task(s). Falls back to the "default" task if no task name
# was specified, or lists all tasks if an unknown task name was specified.
#
# An `extern` signature is static, so the experimental flag at the bottom is
# always offered; Task rejects it when the experiment is off. Run
# `task --experiments` to see which experiments are enabled.
export extern "task" [
  ...tasks: string@"nu-complete task"             # task(s) to run
  --list(-l)                                      # list tasks with a description
  --list-all(-a)                                  # list all tasks, with or without a description
  --json(-j)                                      # format the task list as JSON
  --no-status                                     # ignore status when listing tasks as JSON
  --nested                                        # nest namespaces when listing tasks as JSON
  --sort: string@"nu-complete task-sort"          # change the order of the tasks when listed
  --init(-i)                                      # create a new Taskfile.yml in the current folder
  --completion: string@"nu-complete task-shells"  # generate a shell completion script
  --taskfile(-t): glob                            # choose which Taskfile to run
  --dir(-d): directory                            # set the directory in which Task will execute
  --global(-g)                                    # run the global Taskfile from $HOME
  --temp-dir: directory                           # directory used to store Task temporary files
  --force(-f)                                     # force execution even when the task is up-to-date
  --status                                        # exit with a non-zero code if tasks are not up-to-date
  --dry(-n)                                       # compile and print the tasks without executing them
  --summary                                       # show the summary of a task instead of running it
  --watch(-w)                                     # watch the given tasks and re-run them on changes
  --interval(-I): string                          # interval to watch for changes, e.g. 500ms
  --parallel(-p)                                  # run the tasks given on the command line in parallel
  --concurrency(-C): int                          # limit the number of tasks run concurrently
  --failfast(-F)                                  # when running in parallel, stop everything if one task fails
  --exit-code(-x)                                 # pass through the exit code of the task command
  --interactive                                   # prompt for missing required variables
  --yes(-y)                                       # assume "yes" as the answer to all prompts
  --output(-o): string@"nu-complete task-output"  # set the output style
  --output-group-begin: string                    # message template printed before a task's grouped output
  --output-group-end: string                      # message template printed after a task's grouped output
  --output-group-error-only                       # swallow the output of successful tasks
  --output-group-by-task                          # group all task output together
  --color(-c)                                     # colored output, enabled by default
  --silent(-s)                                    # disable echoing
  --verbose(-v)                                   # enable verbose mode
  --disable-fuzzy                                 # disable fuzzy matching for task names
  --download                                      # download a cached version of a remote Taskfile
  --offline                                       # only use local or cached Taskfiles
  --clear-cache                                   # clear the remote Taskfile cache
  --trusted-hosts: string                         # trusted hosts for remote Taskfiles (comma-separated)
  --timeout: string                               # timeout for downloading remote Taskfiles
  --expiry: string                                # expiry duration for cached remote Taskfiles
  --remote-cache-dir: directory                   # directory used to cache remote Taskfiles
  --cacert: path                                  # custom CA certificate for HTTPS connections
  --cert: path                                    # client certificate for HTTPS connections
  --cert-key: path                                # client certificate key for HTTPS connections
  --insecure                                      # allow Taskfiles to be downloaded over insecure connections
  --experiments                                   # list the available experiments and whether they are enabled
  --version                                       # show the Task version
  --help(-h)                                      # show Task usage

  --force-all                                     # [GENTLE_FORCE] force the called task and all its dependencies
]
