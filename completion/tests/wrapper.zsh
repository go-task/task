#!/usr/bin/env zsh
# Tests the zsh wrapper by stubbing the completion-system functions it calls
# (_describe / _files / _path_files) and asserting how it routes each directive.
# This is deterministic and needs no TTY.
#
# Requires: TASK_BIN (task binary), TASK_FIXTURE (dir with a Taskfile.yml).

export TASK_EXE=$TASK_BIN
cd $TASK_FIXTURE

integer fails=0
local CAP
compdef() { }   # no-op: we call _task directly, not through compinit

_describe() {
    local arr=$4
    CAP+="describe_opts:${@[5,-1]}"$'\n'
    local c; for c in ${(P)arr}; do CAP+="cand:$c"$'\n'; done
}
_files()      { CAP+="files:$*"$'\n' }
_path_files() { CAP+="path_files:$*"$'\n' }

# Sourcing (not autoloading) defines _task and avoids the autoload first-call
# quirk; the trailing `compdef` call is stubbed above.
source ${0:A:h}/../zsh/_task

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

echo "zsh: task names (no file fallback)"
run task ''
has    "lists tasks"        "cand:build"
has    "lists aliases"      "cand:dep"
hasnot "no file fallback"   "files:"

echo "zsh: task variables"
run task deploy ''
has    "required vars"      "cand:ENV=dev"
has    "NoSpace -> -S"      "describe_opts:-S"
has    "KeepOrder -> -V"    "-V"

echo "zsh: --dir routes to directory completion"
run task --dir ''
has    "path_files -/"      "path_files:-/"

echo "zsh: --taskfile routes to extension-filtered files"
run task --taskfile ''
has    "files glob"         "files:"
has    "yml in glob"        "yml"

echo "zsh: after -- falls back to files"
run task build -- ''
has    "files after --"     "files:"

if (( fails )); then
    echo "zsh: $fails failure(s)"
    exit 1
fi
echo "zsh: all passed"
