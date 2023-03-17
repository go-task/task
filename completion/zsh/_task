#compdef task

local context state state_descr line
typeset -A opt_args

_GO_TASK_COMPLETION_LIST_OPTION="${GO_TASK_COMPLETION_LIST_OPTION:---list-all}"

# Listing commands from Taskfile.yml
function __task_list() {
    local -a scripts cmd
    local -i enabled=0
    local taskfile item task desc

    cmd=(task)
    taskfile="${(v)opt_args[(i)-t|--taskfile]}"

    if [[ -n "$taskfile" && -f "$taskfile" ]]; then
        enabled=1
        cmd+=(--taskfile "$taskfile")
    else
        for taskfile in Taskfile{,.dist}.{yaml,yml}; do
            if [[ -f "$taskfile" ]]; then
                enabled=1
                break
            fi
        done
    fi

    (( enabled )) || return 0

    scripts=()
    for item in "${(@)${(f)$("${cmd[@]}" $_GO_TASK_COMPLETION_LIST_OPTION)}[2,-1]#\* }"; do
        task="${item%%:[[:space:]]*}"
        desc="${item##[^[:space:]]##[[:space:]]##}"
        scripts+=( "${task//:/\\:}:$desc" )
    done
    _describe 'Task to run' scripts
}

_arguments \
    '(-C --concurrency)'{-C,--concurrency}'[limit number of concurrent tasks]: ' \
    '(-p --parallel)'{-p,--parallel}'[run command-line tasks in parallel]' \
    '(-f --force)'{-f,--force}'[run even if task is up-to-date]' \
    '(-c --color)'{-c,--color}'[colored output]' \
    '(-d --dir)'{-d,--dir}'[dir to run in]:execution dir:_dirs' \
    '(--dry)--dry[dry-run mode, compile and print tasks only]' \
    '(-o --output)'{-o,--output}'[set output style]:style:(interleaved group prefixed)' \
    '(--output-group-begin)--output-group-begin[message template before grouped output]:template text: ' \
    '(--output-group-end)--output-group-end[message template after grouped output]:template text: ' \
    '(-s --silent)'{-s,--silent}'[disable echoing]' \
    '(--status)--status[exit non-zero if supplied tasks not up-to-date]' \
    '(--summary)--summary[show summary\: field from tasks instead of running them]' \
    '(-t --taskfile)'{-t,--taskfile}'[specify a different taskfile]:taskfile:_files' \
    '(-v --verbose)'{-v,--verbose}'[verbose mode]' \
    '(-w --watch)'{-w,--watch}'[watch-mode for given tasks, re-run when inputs change]' \
    + '(operation)' \
        {-l,--list}'[list describable tasks]' \
        {-a,--list-all}'[list all tasks]' \
        {-i,--init}'[create new Taskfile.yml]' \
        '(-*)'{-h,--help}'[show help]' \
        '(-*)--version[show version and exit]' \
        '*: :__task_list'
