set GO_TASK_PROGNAME task

function __task_get_tasks --description "Prints all available tasks with their description"
  # Read the list of tasks (and potential errors)
  $GO_TASK_PROGNAME --list-all 2>&1 | read -lz rawOutput

  # Return on non-zero exit code (for cases when there is no Taskfile found or etc.)
  if test $status -ne 0
    return
  end

  # Grab names and descriptions (if any) of the tasks
  set -l output (echo $rawOutput | sed -e '1d; s/\* \(.*\):\s*\(.*\)\s*(aliases.*/\1\t\2/' -e 's/\* \(.*\):\s*\(.*\)/\1\t\2/'| string split0)
  if test $output
    echo $output
  end
end

complete -c $GO_TASK_PROGNAME -d 'Runs the specified task(s). Falls back to the "default" task if no task name was specified, or lists all tasks if an unknown task name was
specified.' -xa "(__task_get_tasks)"

complete -c $GO_TASK_PROGNAME -s c -l color     -d 'colored output (default true)'
complete -c $GO_TASK_PROGNAME -s d -l dir       -d 'sets directory of execution'
complete -c $GO_TASK_PROGNAME      -l dry       -d 'compiles and prints tasks in the order that they would be run, without executing them'
complete -c $GO_TASK_PROGNAME -s f -l force     -d 'forces execution even when the task is up-to-date'
complete -c $GO_TASK_PROGNAME -s h -l help      -d 'shows Task usage'
complete -c $GO_TASK_PROGNAME -s i -l init      -d 'creates a new Taskfile.yml in the current folder'
complete -c $GO_TASK_PROGNAME -s l -l list      -d 'lists tasks with description of current Taskfile'
complete -c $GO_TASK_PROGNAME -s o -l output    -d 'sets output style: [interleaved|group|prefixed]' -xa "interleaved group prefixed"
complete -c $GO_TASK_PROGNAME -s p -l parallel  -d 'executes tasks provided on command line in parallel'
complete -c $GO_TASK_PROGNAME -s s -l silent    -d 'disables echoing'
complete -c $GO_TASK_PROGNAME      -l status    -d 'exits with non-zero exit code if any of the given tasks is not up-to-date'
complete -c $GO_TASK_PROGNAME      -l summary   -d 'show summary about a task'
complete -c $GO_TASK_PROGNAME -s t -l taskfile  -d 'choose which Taskfile to run. Defaults to "Taskfile.yml"'
complete -c $GO_TASK_PROGNAME -s v -l verbose   -d 'enables verbose mode'
complete -c $GO_TASK_PROGNAME      -l version   -d 'show Task version'
complete -c $GO_TASK_PROGNAME -s w -l watch     -d 'enables watch of the given task'
