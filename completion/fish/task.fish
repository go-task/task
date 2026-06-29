# Thin wrapper around `task __complete`. All suggestion logic lives in the
# Go engine — do not add completion logic here.

set -l GO_TASK_PROGNAME (if set -q GO_TASK_PROGNAME; echo $GO_TASK_PROGNAME; else if set -q TASK_EXE; echo $TASK_EXE; else; echo task; end)

function __task_complete --inherit-variable GO_TASK_PROGNAME
    set -l tokens (commandline -opc)
    set -l current (commandline -ct)
    set -l args
    if test (count $tokens) -gt 1
        set args $tokens[2..-1]
    end
    set args $args $current

    set -l output ($GO_TASK_PROGNAME __complete $args 2>/dev/null)
    set -l count (count $output)
    if test $count -eq 0
        return
    end

    set -l last $output[$count]
    if not string match -q ':*' -- $last
        # Protocol violation: emit raw lines as a fallback.
        for line in $output
            echo $line
        end
        return
    end

    set -l directive (string replace -r '^:' '' -- $last)
    # FilterFileExt / FilterDirs are handled by fish's native file completion
    # via the separate `complete` registrations below.
    if test (math "$directive & 8") -ne 0; or test (math "$directive & 16") -ne 0
        return
    end

    if test $count -gt 1
        for line in $output[1..(math $count - 1)]
            echo $line
        end
    end
end

complete -c $GO_TASK_PROGNAME --no-files -a "(__task_complete)"
complete -c $GO_TASK_PROGNAME -s t -l taskfile -r -k -a "(__fish_complete_suffix .yml .yaml)"
complete -c $GO_TASK_PROGNAME -s d -l dir -xa "(__fish_complete_directories)"
