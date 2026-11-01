#!/usr/bin/env zsh
# Smoke-tests how the zsh wrapper routes each directive, by stubbing _describe,
# _files and _path_files. Requires TASK_BIN and TASK_FIXTURE.

export TASK_EXE=$TASK_BIN
cd $TASK_FIXTURE

integer fails=0
local CAP
compdef() { }   # no-op: we call _task directly, not through compinit

# Mirrors the real signature — `_describe [-12JVoOx] [-t tag] descr array
# [compadd-opt ...]` — so an option landing in the wrong zone is visible: the
# trailing zone goes to compadd, where -J and -V swallow the next argument.
_describe() {
    local -a flags
    while [[ $1 == -* ]]; do
        case $1 in
            (-t) flags+=($1 $2); shift 2 ;;
            (*)  flags+=($1); shift ;;
        esac
    done
    local arr=$2                       # $1 is descr
    CAP+="describe_flags:[${flags[*]}]"$'\n'
    CAP+="compadd_opts:[${@[3,-1]}]"$'\n'
    local c; for c in ${(P)arr}; do CAP+="cand:$c"$'\n'; done
}
_files()      { CAP+="files:$*"$'\n' }
_path_files() { CAP+="path_files:$*"$'\n' }

completion_dir=${0:A:h}/../../completion/zsh
source $completion_dir/_task

run() {
    CAP=""
    local -a words=("$@")
    integer CURRENT=$#words
    local curcontext=":completion:complete:task:"
    _task
}

has() { # LABEL PATTERN
    if [[ "$CAP" == *"$2"* ]]; then
        echo "  ok   $1"
    else
        echo "  FAIL $1 — expected '$2' in:"$'\n'"$CAP"
        (( fails++ ))
    fi
}
hasnot() { # LABEL PATTERN
    if [[ "$CAP" == *"$2"* ]]; then
        echo "  FAIL $1 — '$2' should be absent in:"$'\n'"$CAP"
        (( fails++ ))
    else
        echo "  ok   $1"
    fi
}

echo "zsh: :4 (NoFileComp) forwards candidates, no file fallback"
run task ''
has    "candidate forwarded"  "cand:build"
hasnot "no file fallback"     "files:"

echo "zsh: arguments are passed without shell quoting"
run task --dir "'with space'" ''
has "single-quoted dir" "cand:spaced"
run task --dir '"with space"' ''
has "double-quoted dir" "cand:spaced"
run task --dir 'with\ space' ''
has "escaped dir" "cand:spaced"
run task --taskfile "'with space/Taskfile.yml'" ''
has "quoted taskfile" "cand:spaced"

echo "zsh: autoload completes on the first invocation"
unfunction _task
fpath=($completion_dir $fpath)
autoload -Uz _task
run task ''
has "first autoload call" "cand:build"

# In the compadd zone, -V would take the next argument as a group name and
# swallow _describe's own `-d`, offering its internal variables as candidates.
echo "zsh: :2|:32 (NoSpace|KeepOrder) reach the right option zones"
run task deploy ''
has    "KeepOrder -> _describe -V"  "describe_flags:[-V"
has    "NoSpace -> compadd -S"      "compadd_opts:[-S ]"

echo "zsh: :8 (FilterFileExt) routes to extension-filtered files"
run task --taskfile ''
has    "files glob"            "files:"
has    "yml in glob"           "yml"

echo "zsh: :16 (FilterDirs) routes to directory completion"
run task --dir ''
has    "path_files -/"         "path_files:-/"

echo "zsh: :0 (Default) falls back to files"
run task build -- ''
has    "files default"         "files:"

if (( fails )); then
    echo "zsh: $fails failure(s)"
    exit 1
fi
echo "zsh: all passed"
