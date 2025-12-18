set -l GO_TASK_PROGNAME (if set -q GO_TASK_PROGNAME; echo $GO_TASK_PROGNAME; else if set -q TASK_EXE; echo $TASK_EXE; else; echo task; end)

# Cache variables for experiments (global)
set -g __task_experiments_cache ""
set -g __task_experiments_cache_time 0

# Helper function to get experiments with 1-second cache
function __task_get_experiments
    set -l now (date +%s)
    set -l ttl 1  # Cache for 1 second only

    # Return cached value if still valid
    if test (math "$now - $__task_experiments_cache_time") -lt $ttl
        printf '%s\n' $__task_experiments_cache
        return
    end

    # Refresh cache
    set -g __task_experiments_cache (task --experiments 2>/dev/null)
    set -g __task_experiments_cache_time $now
    printf '%s\n' $__task_experiments_cache
end

# Helper function to check if an experiment is enabled
function __task_is_experiment_enabled
    set -l experiment $argv[1]
    __task_get_experiments | string match -qr "^\* $experiment:.*on"
end

function __task_get_tasks --description "Prints all available tasks with their description" --inherit-variable GO_TASK_PROGNAME
  # Check if the global task is requested
  set -l global_task false
  commandline --current-process | read --tokenize --list --local cmd_args
  for arg in $cmd_args
    if test "_$arg" = "_--"
      break # ignore arguments to be passed to the task
    end
    if test "_$arg" = "_--global" -o "_$arg" = "_-g"
      set global_task true
      break
    end
  end

  # Read the list of tasks (and potential errors)
  if $global_task
    $GO_TASK_PROGNAME --global --list-all
  else
    $GO_TASK_PROGNAME --list-all
  end 2>&1 | read -lz rawOutput

  # Return on non-zero exit code (for cases when there is no Taskfile found or etc.)
  if test $status -ne 0
    return
  end

  # Grab names and descriptions (if any) of the tasks
  set -l output (echo $rawOutput | sed -e '1d; s/\* \(.*\):\s\{2,\}\(.*\)\s\{2,\}(\(aliases.*\))/\1\t\2\t\3/' -e 's/\* \(.*\):\s\{2,\}\(.*\)/\1\t\2/'| string split0)
  if test $output
    echo $output
  end
end

complete -c $GO_TASK_PROGNAME \
  -d 'Runs the specified task(s). Falls back to the "default" task if no task name was specified, or lists all tasks if an unknown task name was specified.' \
  -xa "(__task_get_tasks)" \
  -n "not __fish_seen_subcommand_from --"

# Standard flags
complete -c $GO_TASK_PROGNAME -s a -l list-all                  -d 'list all tasks'
complete -c $GO_TASK_PROGNAME -s c -l color                     -d 'colored output (default true)'
complete -c $GO_TASK_PROGNAME -s C -l concurrency               -d 'limit number of concurrent tasks'
complete -c $GO_TASK_PROGNAME      -l completion                -d 'generate shell completion script' -xa "bash zsh fish powershell"
complete -c $GO_TASK_PROGNAME -s d -l dir                       -d 'set directory of execution'
complete -c $GO_TASK_PROGNAME      -l disable-fuzzy             -d 'disable fuzzy matching for task names'
complete -c $GO_TASK_PROGNAME -s n -l dry                       -d 'compile and print tasks without executing'
complete -c $GO_TASK_PROGNAME -s x -l exit-code                 -d 'pass-through exit code of task command'
complete -c $GO_TASK_PROGNAME      -l experiments               -d 'list available experiments'
complete -c $GO_TASK_PROGNAME -s F -l failfast                  -d 'when running tasks in parallel, stop all tasks if one fails'
complete -c $GO_TASK_PROGNAME -s f -l force                     -d 'force execution even when up-to-date'
complete -c $GO_TASK_PROGNAME -s g -l global                    -d 'run global Taskfile from home directory'
complete -c $GO_TASK_PROGNAME -s h -l help                      -d 'show help'
complete -c $GO_TASK_PROGNAME -s i -l init                      -d 'create new Taskfile'
complete -c $GO_TASK_PROGNAME      -l insecure                  -d 'allow insecure Taskfile downloads'
complete -c $GO_TASK_PROGNAME -s I -l interval                  -d 'interval to watch for changes'
complete -c $GO_TASK_PROGNAME -s j -l json                      -d 'format task list as JSON'
complete -c $GO_TASK_PROGNAME -s l -l list                      -d 'list tasks with descriptions'
complete -c $GO_TASK_PROGNAME      -l nested                    -d 'nest namespaces when listing as JSON'
complete -c $GO_TASK_PROGNAME      -l no-status                 -d 'ignore status when listing as JSON'
complete -c $GO_TASK_PROGNAME -s o -l output                    -d 'set output style' -xa "interleaved group prefixed"
complete -c $GO_TASK_PROGNAME      -l output-group-begin        -d 'message template before grouped output'
complete -c $GO_TASK_PROGNAME      -l output-group-end          -d 'message template after grouped output'
complete -c $GO_TASK_PROGNAME      -l output-group-error-only   -d 'hide output from successful tasks'
complete -c $GO_TASK_PROGNAME -s p -l parallel                  -d 'execute tasks in parallel'
complete -c $GO_TASK_PROGNAME -s s -l silent                    -d 'disable echoing'
complete -c $GO_TASK_PROGNAME      -l sort                      -d 'set task sorting order' -xa "default alphanumeric none"
complete -c $GO_TASK_PROGNAME      -l status                    -d 'exit non-zero if tasks not up-to-date'
complete -c $GO_TASK_PROGNAME      -l summary                   -d 'show task summary'
complete -c $GO_TASK_PROGNAME -s t -l taskfile                  -d 'choose Taskfile to run'
complete -c $GO_TASK_PROGNAME -s v -l verbose                   -d 'verbose output'
complete -c $GO_TASK_PROGNAME      -l version                   -d 'show version'
complete -c $GO_TASK_PROGNAME -s w -l watch                     -d 'watch mode, re-run on changes'
complete -c $GO_TASK_PROGNAME -s y -l yes                       -d 'assume yes to all prompts'

# Experimental flags (dynamically checked at completion time via -n condition)
# GentleForce experiment
complete -c $GO_TASK_PROGNAME -n "__task_is_experiment_enabled GENTLE_FORCE" -l force-all -d 'force execution of task and all dependencies'

# RemoteTaskfiles experiment - Options
complete -c $GO_TASK_PROGNAME -n "__task_is_experiment_enabled REMOTE_TASKFILES" -l offline          -d 'use only local or cached Taskfiles'
complete -c $GO_TASK_PROGNAME -n "__task_is_experiment_enabled REMOTE_TASKFILES" -l timeout          -d 'timeout for remote Taskfile downloads'
complete -c $GO_TASK_PROGNAME -n "__task_is_experiment_enabled REMOTE_TASKFILES" -l expiry           -d 'cache expiry duration'
complete -c $GO_TASK_PROGNAME -n "__task_is_experiment_enabled REMOTE_TASKFILES" -l remote-cache-dir -d 'directory to cache remote Taskfiles' -xa "(__fish_complete_directories)"

# RemoteTaskfiles experiment - Operations
complete -c $GO_TASK_PROGNAME -n "__task_is_experiment_enabled REMOTE_TASKFILES" -l download    -d 'download remote Taskfile'
complete -c $GO_TASK_PROGNAME -n "__task_is_experiment_enabled REMOTE_TASKFILES" -l clear-cache -d 'clear remote Taskfile cache'
